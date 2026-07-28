package accel

import "testing"

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

func resetBackendExecutorsForTest() {
	backendExecutorsMu.Lock()
	defer backendExecutorsMu.Unlock()
	backendExecutors = map[Backend]BackendExecutor{}
}
