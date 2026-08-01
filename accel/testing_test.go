package accel

import (
	"maps"
	"os"
	"sync"
	"testing"
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
