package accel

import (
	"context"
	"maps"
	"os"
	"sync"
	"testing"
	"time"
)

// isolateBuiltinProbes reports every builtin backend as unavailable so a real
// host GPU cannot be discovered during a test. Without it a test that overrides
// only some backends still picks up whatever the machine actually has — on a
// Mac the Metal probe finds the Apple GPU and every device count comes out one
// too high. Tests call this first, then override the backends they exercise.
//
// Only the native probe is overridden; the env-stub probe is left in place so
// INSYRA_ACCEL_STUB_* keeps working.
func isolateBuiltinProbes(t *testing.T) {
	t.Helper()
	resetBuiltinProbeOverridesForTest()
	t.Cleanup(resetBuiltinProbeOverridesForTest)
	unavailable := func(Config) ([]Device, error) { return nil, ErrNativeProbeUnavailable }
	for _, backend := range []Backend{BackendCUDA, BackendMetal, BackendWebGPU} {
		setBuiltinProbeOverrideForTest(backend, unavailable, nil)
	}
}

func setBuiltinProbeOverrideForTest(backend Backend, native builtinProbeFunc, stub builtinProbeFunc) {
	builtinProbeOverrides[backend] = builtinProbeOverride{native: native, stub: stub}
}

func resetBuiltinProbeOverridesForTest() {
	builtinProbeOverrides = map[Backend]builtinProbeOverride{}
}

func setHostMemoryBytesForTest(bytes uint64) func() {
	previous := hostMemoryBytesFunc
	hostMemoryBytesFunc = func() uint64 { return bytes }
	return func() {
		hostMemoryBytesFunc = previous
	}
}

// builtinExecutors holds the registrations the package makes for itself at
// startup, captured the first time a test clears the registry.
var (
	builtinExecutors    map[Backend]BackendExecutor
	captureBuiltinsOnce sync.Once
)

// resetBackendExecutorsForTest restores the registry to what package
// initialisation left in it.
//
// It must restore rather than empty. Clearing it outright leaves the builtin
// device backend unregistered for every later test, which is invisible until a
// GPU-gated test runs after one of them and fails with no-backend-executor —
// meaning the bit-parity gate silently stops covering anything in a full run.
func resetBackendExecutorsForTest() {
	backendExecutorsMu.Lock()
	defer backendExecutorsMu.Unlock()
	captureBuiltinsOnce.Do(func() {
		builtinExecutors = maps.Clone(backendExecutors)
	})
	backendExecutors = maps.Clone(builtinExecutors)
}

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

// requireGPU keeps the hardware tests off machines that have no device, and
// off race-enabled builds.
//
// -race turns on checkptr, which aborts the process inside gogpu's Metal
// completion-block trampoline (hal/metal/objc.go:958, reached through goffi's
// callback path) with "checkptr: pointer arithmetic result points to invalid
// allocation". The violation is upstream and predates the session lock —
// reproduced at the previous commit — so the guard is here rather than a fix.
// Everything that does not touch a device still runs under -race.
func requireGPU(t *testing.T) {
	t.Helper()
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1 to run tests against a real GPU")
	}
	if raceDetectorEnabled {
		t.Skip("gogpu's Metal path trips checkptr under -race; run device tests without it")
	}
}

func gpuTestsEnabled(b *testing.B) bool {
	b.Helper()
	if raceDetectorEnabled {
		b.Skip("gogpu's Metal path trips checkptr under -race; benchmark without it")
		return false
	}
	session, err := Open(Config{})
	if err != nil {
		b.Skipf("cannot open an accel session: %v", err)
		return false
	}
	devices := len(session.Devices())
	_ = session.Close()
	if devices == 0 {
		b.Skip("no GPU discovered on this host")
		return false
	}
	return true
}
