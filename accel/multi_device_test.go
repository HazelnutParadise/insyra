package accel

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"
)

type lockedShortlistExecutor struct {
	mu    sync.Mutex
	inner shortlistExecutor
}

func (e *lockedShortlistExecutor) Name() string { return "locked-shortlist" }

func (e *lockedShortlistExecutor) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.inner.Execute(ctx, req)
}

type assignmentFailureExecutor struct {
	failedDevice string
	inner        lockedShortlistExecutor
}

func (e *assignmentFailureExecutor) Name() string { return "assignment-failure" }

func (e *assignmentFailureExecutor) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	if req.Device.ID == e.failedDevice {
		return ExecuteResponse{}, errors.New("test device failed")
	}
	return e.inner.Execute(ctx, req)
}

func multiDeviceTestSession(t *testing.T, cfg Config) *Session {
	t.Helper()
	disableNativeProbes(t)
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	isolateBuiltinProbes(t)
	RegisterDiscoverer(stubDiscoverer{
		name: "multi-device",
		devices: []Device{
			{ID: "cuda:a", Backend: BackendCUDA, Type: DeviceTypeDiscrete, MemoryClass: MemoryClassDevice, Score: 100},
			{ID: "cuda:b", Backend: BackendCUDA, Type: DeviceTypeDiscrete, MemoryClass: MemoryClassDevice, Score: 90},
		},
	})
	session, err := Open(cfg)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestShardStrategiesRespectMeasuredFloor(t *testing.T) {
	auto := multiDeviceTestSession(t, Config{})
	below := auto.PlanShardableWorkload(WorkloadEstimate{Class: WorkloadClassColumnar, Rows: 31_999, Dimensions: 32})
	if len(below.Assignments) != 1 || below.Assignments[0].Rows != 31_999 {
		t.Fatalf("auto below floor = %#v, want one 31999-row assignment", below.Assignments)
	}
	above := auto.PlanShardableWorkload(WorkloadEstimate{Class: WorkloadClassColumnar, Rows: 64_000, Dimensions: 32})
	if len(above.Assignments) != 2 {
		t.Fatalf("auto above floor produced %d assignments, want 2", len(above.Assignments))
	}
	for _, assignment := range above.Assignments {
		if assignment.Rows < saturationRows32 || assignment.RowEnd <= assignment.RowStart {
			t.Fatalf("auto assignment below measured floor or without range: %#v", assignment)
		}
	}
	heavy := auto.PlanShardableWorkload(WorkloadEstimate{Class: WorkloadClassColumnar, Rows: 16_000, Dimensions: 128})
	if len(heavy.Assignments) != 2 {
		t.Fatalf("auto heavy-shape plan produced %d assignments, want 2", len(heavy.Assignments))
	}
	for _, assignment := range heavy.Assignments {
		if assignment.Rows < saturationRows128 {
			t.Fatalf("heavy-shape assignment below measured floor: %#v", assignment)
		}
	}

	forced := multiDeviceTestSession(t, Config{ShardStrategy: ShardStrategyForced})
	plan := forced.PlanShardableWorkload(WorkloadEstimate{Class: WorkloadClassColumnar, Rows: 1_024, Dimensions: 128})
	if len(plan.Assignments) != 2 {
		t.Fatalf("forced plan produced %d assignments, want 2", len(plan.Assignments))
	}
	if plan.Assignments[0].Rows+plan.Assignments[1].Rows != 1_024 {
		t.Fatalf("forced rows do not cover workload: %#v", plan.Assignments)
	}
}

func TestMultiDeviceDispatchMergesRangesAndReportsPlacement(t *testing.T) {
	exerciseDeviceRegardlessOfProfit(t)
	session := multiDeviceTestSession(t, Config{ShardStrategy: ShardStrategyForced})
	backend := &lockedShortlistExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, backend); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	rnd := rand.New(rand.NewSource(20260805))
	dataset := exactDataset(1_024, 3, rnd)
	queries := exactQueries(40, 3, rnd)
	result, err := session.ExecuteNearestExact(dataset, queries, 2, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !result.Accelerated || len(result.Assignments) != 2 {
		t.Fatalf("result = accelerated %t assignments %d fallback %q", result.Accelerated, len(result.Assignments), result.FallbackReason)
	}
	for _, assignment := range result.Assignments {
		if assignment.FallbackReason != FallbackReasonNone || assignment.WallTime <= 0 {
			t.Fatalf("assignment did not report successful placement and wall time: %#v", assignment)
		}
	}
	assertMatchesReference(t, dataset, queries, 2, result.Index, result.Distance)
}

func TestMultiDeviceAssignmentFailureFallsBackOnlyItsRows(t *testing.T) {
	exerciseDeviceRegardlessOfProfit(t)
	session := multiDeviceTestSession(t, Config{ShardStrategy: ShardStrategyForced})
	backend := &assignmentFailureExecutor{failedDevice: "cuda:b"}
	if err := RegisterBackendExecutor(BackendCUDA, backend); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	rnd := rand.New(rand.NewSource(20260806))
	dataset := exactDataset(1_024, 3, rnd)
	queries := exactQueries(40, 3, rnd)
	result, err := session.ExecuteNearestExact(dataset, queries, 2, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("automatic per-assignment fallback should not error: %v", err)
	}
	if !result.Accelerated || result.FallbackReason != FallbackReasonExecutionFailed {
		t.Fatalf("partial failure result = accelerated %t fallback %q", result.Accelerated, result.FallbackReason)
	}
	for _, assignment := range result.Assignments {
		if assignment.DeviceID == "cuda:b" && assignment.FallbackReason != FallbackReasonExecutionFailed {
			t.Fatalf("failed assignment report = %#v", assignment)
		}
		if assignment.DeviceID == "cuda:a" && assignment.FallbackReason != FallbackReasonNone {
			t.Fatalf("healthy assignment was changed by peer failure: %#v", assignment)
		}
	}
	assertMatchesReference(t, dataset, queries, 2, result.Index, result.Distance)
}

func TestMultiDeviceParityConcurrentAndSequentialOnHardware(t *testing.T) {
	requireHardwareGPU(t)
	session, err := Open(Config{})
	if err != nil && (session == nil || len(session.Devices()) == 0) {
		t.Skipf("no usable GPU backend: %v", err)
	}
	if session == nil {
		t.Skip("no session")
	}
	defer func() { _ = session.Close() }()
	devices := session.Devices()
	if len(devices) == 0 {
		t.Skip("no hardware device discovered")
	}
	device := devices[0]
	executor, ok := lookupBackendExecutor(device.Backend)
	if !ok {
		t.Skipf("no backend executor for %q", device.Backend)
	}
	rnd := rand.New(rand.NewSource(20260807))
	dataset := exactDataset(2_048, 32, rnd)
	queries := exactQueries(128, 32, rnd)
	columns, rows, reason := narrowColumns(dataset)
	if reason != FallbackReasonNone {
		t.Fatalf("narrow columns failed: %s", reason)
	}
	host, _, reason := hostColumns(dataset)
	if reason != FallbackReasonNone {
		t.Fatalf("host columns failed: %s", reason)
	}
	narrowed := narrowQueries(queries)
	assignments := []ShardAssignment{
		{DeviceID: device.ID, Backend: device.Backend, RowStart: 0, RowEnd: rows / 2, Rows: rows / 2},
		{DeviceID: device.ID, Backend: device.Backend, RowStart: rows / 2, RowEnd: rows, Rows: rows - rows/2},
	}
	run := func(assignment ShardAssignment) assignmentExecution {
		return executeNearestAssignment(context.Background(), executor, device, true, assignment, columns, host, queries, narrowed, 2, 4)
	}
	sequential := []assignmentExecution{run(assignments[0]), run(assignments[1])}
	concurrent := make([]assignmentExecution, 2)
	var wg sync.WaitGroup
	for i, assignment := range assignments {
		wg.Add(1)
		go func(i int, assignment ShardAssignment) {
			defer wg.Done()
			concurrent[i] = run(assignment)
		}(i, assignment)
	}
	wg.Wait()
	merge := func(outcomes []assignmentExecution) ([]uint32, []float64) {
		index := make([]uint32, rows*2)
		distance := make([]float64, rows*2)
		for _, outcome := range outcomes {
			copy(index[outcome.Assignment.RowStart*2:outcome.Assignment.RowEnd*2], outcome.Index)
			copy(distance[outcome.Assignment.RowStart*2:outcome.Assignment.RowEnd*2], outcome.Distance)
		}
		return index, distance
	}
	seqIndex, seqDistance := merge(sequential)
	concurrentIndex, concurrentDistance := merge(concurrent)
	wantIndex, wantDistance, _, err := NearestExactCPU(dataset, queries, 2)
	if err != nil {
		t.Fatalf("reference failed: %v", err)
	}
	if !reflect.DeepEqual(seqIndex, concurrentIndex) || !reflect.DeepEqual(seqDistance, concurrentDistance) {
		t.Fatal("concurrent and sequential assignment outputs differ")
	}
	if !reflect.DeepEqual(concurrentIndex, wantIndex) || !reflect.DeepEqual(concurrentDistance, wantDistance) {
		t.Fatal("multi-assignment output differs from brute force")
	}
	if !reflect.DeepEqual(encodeNearest(seqIndex, seqDistance), encodeNearest(wantIndex, wantDistance)) {
		t.Fatal("multi-assignment output is not byte-identical to brute force")
	}
}

func encodeNearest(index []uint32, distance []float64) []byte {
	out := make([]byte, len(index)*4+len(distance)*8)
	for i, value := range index {
		binary.LittleEndian.PutUint32(out[i*4:], value)
	}
	base := len(index) * 4
	for i, value := range distance {
		binary.LittleEndian.PutUint64(out[base+i*8:], math.Float64bits(value))
	}
	return out
}
