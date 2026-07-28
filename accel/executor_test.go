package accel

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// fakeExecutor stands in for a real backend so the runtime's routing,
// eligibility, and failure reporting can be tested on machines with no GPU.
type fakeExecutor struct {
	name     string
	calls    int
	lastReq  ExecuteRequest
	err      error
	response ExecuteResponse
}

func sumColumns(req ExecuteRequest) (map[string]float64, uint64) {
	sums := make(map[string]float64, len(req.Columns))
	bytes := uint64(0)
	for _, column := range req.Columns {
		var sum float64
		for _, value := range column.Values {
			sum += float64(value)
		}
		sums[column.Name] = sum
		bytes += uint64(len(column.Values) * 4)
	}
	return sums, bytes
}

func (e *fakeExecutor) Name() string {
	if e.name == "" {
		return "fake"
	}
	return e.name
}

func (e *fakeExecutor) Execute(_ context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	e.calls++
	e.lastReq = req
	if e.err != nil {
		return ExecuteResponse{}, e.err
	}
	response := e.response
	sums, bytes := sumColumns(req)
	if response.Reductions == nil {
		response.Reductions = sums
	}
	if response.BytesUploaded == 0 {
		response.BytesUploaded = bytes
	}
	if response.Transfer == 0 {
		response.Transfer = time.Millisecond
	}
	if response.Dispatch == 0 {
		response.Dispatch = 100 * time.Microsecond
	}
	if response.Readback == 0 {
		response.Readback = 500 * time.Microsecond
	}
	return response, nil
}

// singleDeviceSession opens a session with exactly one env-stub CUDA device and
// no host devices leaking in.
func singleDeviceSession(t *testing.T, cfg Config) *Session {
	t.Helper()
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	isolateBuiltinProbes(t)
	resetBackendExecutorsForTest()
	t.Cleanup(resetBackendExecutorsForTest)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")

	session, err := Open(cfg)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})
	return session
}

func float64Dataset(name string, values []float64, nulls []bool) *Dataset {
	dataset := &Dataset{
		Name:    name,
		Lineage: "test",
		Rows:    1024,
		Buffers: []Buffer{
			{
				Name:   name,
				Type:   DataTypeFloat64,
				Values: values,
				Nulls:  nulls,
				Len:    len(values),
			},
		},
	}
	assignDatasetFingerprint(dataset)
	return dataset
}

func TestExecuteRoutesToRegisteredBackendExecutor(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	executor := &fakeExecutor{name: "cuda-fake"}
	if err := RegisterBackendExecutor(BackendCUDA, executor); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := float64Dataset("numbers", []float64{1, 2, 3, 4}, nil)
	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Accelerated {
		t.Fatalf("expected accelerated result, got fallback %q", result.FallbackReason)
	}
	if executor.calls != 1 {
		t.Fatalf("expected the executor to be called once, got %d", executor.calls)
	}
	if executor.lastReq.Op != OpSum {
		t.Fatalf("expected the operation to reach the backend, got %q", executor.lastReq.Op)
	}
	if len(executor.lastReq.Columns) != 1 || executor.lastReq.Columns[0].Name != "numbers" {
		t.Fatalf("expected the column to reach the backend, got %+v", executor.lastReq.Columns)
	}
	if executor.lastReq.Device.ID == "" {
		t.Fatal("expected the device to reach the backend")
	}
	if got := result.Reductions["numbers"]; got != 10 {
		t.Fatalf("expected the computed value to come back, got %v", got)
	}
	if result.Counts["numbers"] != 4 {
		t.Fatalf("expected 4 non-null values, got %d", result.Counts["numbers"])
	}
	if result.Executor != "cuda-fake" {
		t.Fatalf("expected the executor name in the result, got %q", result.Executor)
	}
	if result.ExecutorKind != ExecutorKindRegistered {
		t.Fatalf("expected a registered executor kind, got %q", result.ExecutorKind)
	}
	if result.Precision != PrecisionFloat32 {
		t.Fatalf("expected the result to record its precision, got %q", result.Precision)
	}
	if len(result.DeviceIDs) != 1 {
		t.Fatalf("this change executes on one device; got %d", len(result.DeviceIDs))
	}
}

func TestExecuteWithoutRegisteredExecutorFallsBack(t *testing.T) {
	session := singleDeviceSession(t, Config{})

	dataset := float64Dataset("numbers", []float64{1, 2, 3, 4}, nil)
	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute should not error outside strict mode: %v", err)
	}
	if result.Accelerated {
		t.Fatal("expected no acceleration without a registered executor")
	}
	if result.FallbackReason != FallbackReasonNoBackendExecutor {
		t.Fatalf("expected no-backend-executor reason, got %q", result.FallbackReason)
	}
	if result.Reductions != nil {
		t.Fatal("expected no reductions when nothing ran")
	}
}

func TestExecuteRefusesFloat64WithoutPrecisionOptIn(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	executor := &fakeExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, executor); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := float64Dataset("numbers", []float64{1, 2, 3, 4}, nil)
	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute should not error outside strict mode: %v", err)
	}
	if result.Accelerated {
		t.Fatal("expected float64 to be refused without an explicit precision opt-in")
	}
	if result.FallbackReason != FallbackReasonPrecisionNotAccepted {
		t.Fatalf("expected precision-not-accepted reason, got %q", result.FallbackReason)
	}
	if executor.calls != 0 {
		t.Fatalf("expected the backend never to be called, got %d calls", executor.calls)
	}
}

func TestExecuteRefusesColumnTypeWithNoDeviceRepresentation(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	if err := RegisterBackendExecutor(BackendCUDA, &fakeExecutor{}); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := &Dataset{
		Name:    "labels",
		Lineage: "test",
		Rows:    1024,
		Buffers: []Buffer{{Name: "labels", Type: DataTypeString, Values: []string{"a", "b"}, Len: 2}},
	}
	assignDatasetFingerprint(dataset)

	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute should not error outside strict mode: %v", err)
	}
	if result.FallbackReason != FallbackReasonDTypeNotEligible {
		t.Fatalf("expected dtype-not-eligible reason, got %q", result.FallbackReason)
	}
}

func TestExecuteEmptyColumnSkipsDispatch(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	executor := &fakeExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, executor); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := float64Dataset("empty", []float64{}, nil)
	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Rows: 1024, Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !result.Accelerated {
		t.Fatalf("expected an accelerated result, got fallback %q", result.FallbackReason)
	}
	if executor.calls != 0 {
		t.Fatalf("an empty column must not dispatch, got %d calls", executor.calls)
	}
	if got := result.Reductions["empty"]; got != 0 {
		t.Fatalf("expected the additive identity for an empty column, got %v", got)
	}
}

func TestExecuteAllNullColumnSumsToZero(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	executor := &fakeExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, executor); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := float64Dataset("nulls", []float64{0, 0, 0}, []bool{true, true, true})
	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if got := result.Reductions["nulls"]; got != 0 {
		t.Fatalf("expected 0 for an all-null column, got %v", got)
	}
	if result.Counts["nulls"] != 0 {
		t.Fatalf("expected 0 non-null values, got %d", result.Counts["nulls"])
	}
}

func TestExecuteMapsBackendErrorsToFallbackReasons(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want FallbackReason
	}{
		{"shader compile", fmt.Errorf("metal said no: %w", ErrShaderCompile), FallbackReasonShaderCompileFailed},
		{"buffer too large", fmt.Errorf("column too big: %w", ErrBufferTooLarge), FallbackReasonBufferTooLarge},
		{"readback timeout", fmt.Errorf("map stalled: %w", ErrReadbackTimeout), FallbackReasonReadbackTimeout},
		{"deadline exceeded", fmt.Errorf("wrapped: %w", context.DeadlineExceeded), FallbackReasonReadbackTimeout},
		{"anything else", errors.New("device lost"), FallbackReasonExecutionFailed},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			session := singleDeviceSession(t, Config{})
			if err := RegisterBackendExecutor(BackendCUDA, &fakeExecutor{err: testCase.err}); err != nil {
				t.Fatalf("register executor failed: %v", err)
			}

			dataset := float64Dataset("numbers", []float64{1, 2, 3, 4}, nil)
			result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32})
			if err != nil {
				t.Fatalf("execute should not error outside strict mode: %v", err)
			}
			if result.Accelerated {
				t.Fatal("expected a backend failure to stop acceleration")
			}
			if result.FallbackReason != testCase.want {
				t.Fatalf("expected %q, got %q", testCase.want, result.FallbackReason)
			}
		})
	}
}

func TestExecuteStrictModeReturnsTheBackendError(t *testing.T) {
	session := singleDeviceSession(t, Config{Mode: ModeStrictGPU})
	backendErr := errors.New("device lost")
	if err := RegisterBackendExecutor(BackendCUDA, &fakeExecutor{err: backendErr}); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := float64Dataset("numbers", []float64{1, 2, 3, 4}, nil)
	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32})
	if err == nil {
		t.Fatal("expected strict mode to surface the backend error")
	}
	if !errors.Is(err, backendErr) {
		t.Fatalf("expected the backend error to be wrapped, got %v", err)
	}
	if result.FallbackReason != FallbackReasonExecutionFailed {
		t.Fatalf("expected execution-failed reason, got %q", result.FallbackReason)
	}
}

func TestExecuteReportsMeasuredCostOnlyWhenSomethingRan(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	if err := RegisterBackendExecutor(BackendCUDA, &fakeExecutor{}); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := float64Dataset("numbers", []float64{1, 2, 3, 4}, nil)
	if _, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32}); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	metrics := session.Report().Metrics
	if metrics["execution.bytes_uploaded"] <= 0 {
		t.Fatalf("expected measured uploaded bytes, got %v", metrics["execution.bytes_uploaded"])
	}
	if metrics["execution.transfer_ms"] <= 0 {
		t.Fatalf("expected a measured transfer duration, got %v", metrics["execution.transfer_ms"])
	}

	// A refused workload must not leave cost figures behind.
	refused := float64Dataset("numbers", []float64{1, 2, 3, 4}, nil)
	if _, err := session.ExecuteProjectedDataset(refused, WorkloadEstimate{}); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	metrics = session.Report().Metrics
	if _, ok := metrics["execution.transfer_ms"]; ok {
		t.Fatal("expected no cost figures when nothing ran on a device")
	}
	if _, ok := metrics["execution.bytes_uploaded"]; ok {
		t.Fatal("expected no uploaded-byte figure when nothing ran on a device")
	}
}

func TestExecuteRecordsDeviceResidencyAfterRealUpload(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	if err := RegisterBackendExecutor(BackendCUDA, &fakeExecutor{}); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := float64Dataset("numbers", []float64{1, 2, 3, 4}, nil)
	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	snapshot := session.CacheSnapshot()
	if len(snapshot.Entries) != 1 {
		t.Fatalf("expected 1 cache entry, got %d", len(snapshot.Entries))
	}
	entry := snapshot.Entries[0]
	if len(entry.DeviceIDs) != 1 || entry.DeviceIDs[0] != result.DeviceIDs[0] {
		t.Fatalf("expected residency on the executing device, got %v", entry.DeviceIDs)
	}
	if entry.DeviceResidentBytes[result.DeviceIDs[0]] == 0 {
		t.Fatal("expected resident bytes on the executing device")
	}
}

func TestExecuteRejectsUnsupportedOperation(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	executor := &fakeExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, executor); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := float64Dataset("numbers", []float64{1, 2, 3, 4}, nil)
	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Op: Op("mean"), Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute should not error outside strict mode: %v", err)
	}
	if result.FallbackReason != FallbackReasonWorkloadUnsupported {
		t.Fatalf("expected workload-unsupported reason, got %q", result.FallbackReason)
	}
	if executor.calls != 0 {
		t.Fatalf("expected no dispatch for an unsupported operation, got %d calls", executor.calls)
	}
}

// The seam used to submit one request per column, so a DataTable paid a full
// upload/dispatch/readback round trip per column. This pins the batching.
func TestExecuteSubmitsOneRequestForAllColumns(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	executor := &fakeExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, executor); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := &Dataset{
		Name: "table", Lineage: "test", Rows: 1024,
		Buffers: []Buffer{
			{Name: "a", Type: DataTypeFloat64, Values: []float64{1, 2}, Len: 2},
			{Name: "b", Type: DataTypeFloat64, Values: []float64{10, 20}, Len: 2},
			{Name: "c", Type: DataTypeFloat64, Values: []float64{100, 200}, Len: 2},
		},
	}
	assignDatasetFingerprint(dataset)

	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("expected one submission for three columns, got %d", executor.calls)
	}
	if len(executor.lastReq.Columns) != 3 {
		t.Fatalf("expected all three columns in one request, got %d", len(executor.lastReq.Columns))
	}
	for i, want := range []string{"a", "b", "c"} {
		if executor.lastReq.Columns[i].Name != want {
			t.Fatalf("column order not preserved: position %d is %q, want %q", i, executor.lastReq.Columns[i].Name, want)
		}
	}
	for name, want := range map[string]float64{"a": 3, "b": 30, "c": 300} {
		if got := result.Reductions[name]; got != want {
			t.Fatalf("column %q: got %v want %v", name, got, want)
		}
	}
}

// Empty columns are answered without reaching the device, but must not stop the
// non-empty ones in the same dataset from being submitted.
func TestExecuteMixesEmptyAndNonEmptyColumns(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	executor := &fakeExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, executor); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	dataset := &Dataset{
		Name: "mixed", Lineage: "test", Rows: 1024,
		Buffers: []Buffer{
			{Name: "empty", Type: DataTypeFloat64, Values: []float64{}, Len: 0},
			{Name: "full", Type: DataTypeFloat64, Values: []float64{1, 2, 3}, Len: 3},
		},
	}
	assignDatasetFingerprint(dataset)

	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("expected one submission, got %d", executor.calls)
	}
	if len(executor.lastReq.Columns) != 1 || executor.lastReq.Columns[0].Name != "full" {
		t.Fatalf("expected only the non-empty column to be submitted, got %+v", executor.lastReq.Columns)
	}
	if result.Reductions["empty"] != 0 {
		t.Fatalf("expected the additive identity for an empty column, got %v", result.Reductions["empty"])
	}
	if result.Reductions["full"] != 6 {
		t.Fatalf("expected 6 for the full column, got %v", result.Reductions["full"])
	}
}
