package wgpu

// builtinKernels lists every built-in kernel, in the order warmup builds
// them: the ones a training step needs first, then the rest.
// TestBuiltinKernelsAreAllWarmed checks that no kernel is missing.
var builtinKernels = []*kernel{
	matMulRoundedKernel, addRoundedKernel, sumRowsRoundedKernel, sgdRoundedKernel,
	adamRoundedKernel, divRoundedKernel, sqrtRoundedKernel,
	expRoundedKernel, logRoundedKernel, expm1RoundedKernel, log1pRoundedKernel,
	sceMaxKernel, sceExpKernel, sceRowKernel, sceMeanKernel, sceGradKernel,
	sceStatsKernel, sceDivKernel, sceSoftLossKernel, sceMassKernel, sceSoftUKernel,
	sceTotalKernel, sceSoftGradKernel, sceDeltaKernel, sceDistMeanKernel,
	sceDistCentreKernel, sceDistShiftKernel, sceDistUKernel,
}

// warmOnFirstBuffer starts warmup when the first resident buffer is made on
// a device. Tests that must see kernels unbuilt turn it off.
var warmOnFirstBuffer = true

// startWarmupLocked starts building the built-in kernels in the background
// the first time a resident buffer is made on h, so the first computation
// does not wait for its kernels to compile: measured from 40 ms to 1.8 s per
// kernel on a cold driver cache (#584). submitMu must be held.
func startWarmupLocked(h *handle) {
	if !warmOnFirstBuffer || h == nil || h.warmStarted {
		return
	}
	h.warmStarted = true
	go warm(h)
}

// warm builds each built-in kernel on h, taking submitMu for one kernel at a
// time so a computation waits for at most one build. It stops once h is
// retired; a kernel that fails to build keeps its error, as it would on
// first use.
func warm(h *handle) {
	for _, k := range builtinKernels {
		submitMu.Lock()
		if h.retired {
			submitMu.Unlock()
			return
		}
		k.build(h)
		submitMu.Unlock()
	}
}
