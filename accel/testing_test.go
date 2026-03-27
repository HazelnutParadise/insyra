package accel

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
