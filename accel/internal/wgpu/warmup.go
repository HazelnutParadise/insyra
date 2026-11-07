package wgpu

import "sync"

// builtinKernels lists every built-in kernel, in the order warmup builds
// them: the ones a training step needs first, then the rest.
// TestBuiltinKernelsAreAllWarmed checks that no kernel is missing.
var builtinKernels = []*kernel{
	matMulRoundedKernel, addRoundedKernel, sumRowsRoundedKernel, sgdRoundedKernel,
	adamRoundedKernel, divRoundedKernel, sqrtRoundedKernel,
	expRoundedKernel, logRoundedKernel, expm1RoundedKernel, log1pRoundedKernel, tanhRoundedKernel,
	sceMaxKernel, sceExpKernel, sceRowKernel, sceMeanKernel, sceGradKernel,
	sceStatsKernel, sceDivKernel, sceSoftLossKernel, sceMassKernel, sceSoftUKernel,
	sceTotalKernel, sceSoftGradKernel, sceDeltaKernel, sceDistMeanKernel,
	sceDistCentreKernel, sceDistShiftKernel, sceDistUKernel,
	edgeSumRoundedKernel, edgeSumWeightsRoundedKernel,
}

// warmMu is held while warmup compiles, without submitMu, and while acquire
// releases a lost device, so a device is never released under a compile.
var warmMu sync.Mutex

// warmInFlight is the kernel warmup has compiled and not yet installed or
// released. Guarded by warmMu. acquire releases it with a lost device's other
// children, since a lost device is not replaced while any of them is alive.
var warmInFlight *compiledKernel

// warmRunMu lets one warmup run at a time, since warmInFlight has room for
// one kernel: a second device opened after a loss waits for the first
// device's warmup, which stops at its next kernel.
var warmRunMu sync.Mutex

// releasedDevices counts the lost devices releaseLostDevice has released.
// Guarded by warmMu. A warmup records it when it starts and compiles only
// while it is unchanged, so it never compiles on a device already released.
var releasedDevices int

// warmOnFirstBuffer starts warmup when the first resident buffer is made on
// a device; a test that needs kernels left unbuilt can turn it off.
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
	// submitMu before warmMu, as everywhere.
	warmMu.Lock()
	generation := releasedDevices
	warmMu.Unlock()
	go warm(h, generation)
}

// warm builds each built-in kernel on h that is not built yet. It compiles
// without holding submitMu, so computations, uploads and readbacks are not
// queued behind it, and takes the lock only to install the result. When a
// computation built the kernel first, or the device was retired meanwhile,
// warm releases what it compiled. A kernel warm fails to compile is left for
// its first use, which reports the error.
func warm(h *handle, generation int) {
	warmRunMu.Lock()
	defer warmRunMu.Unlock()
	for _, k := range builtinKernels {
		submitMu.Lock()
		retired, built := h.retired, k.pipe != nil || k.err != nil
		submitMu.Unlock()
		if retired {
			return
		}
		if built {
			continue
		}
		warmMu.Lock()
		if releasedDevices != generation {
			// h was retired and its device released since the check above.
			warmMu.Unlock()
			return
		}
		c, err := k.compile(h)
		if err == nil {
			warmInFlight = c
		}
		warmMu.Unlock()
		if err != nil {
			continue
		}
		// submitMu before warmMu, the order a computation takes them in when
		// acquire reopens a lost device.
		submitMu.Lock()
		warmMu.Lock()
		if warmInFlight == c {
			warmInFlight = nil
			used := false
			if !h.retired {
				k.once.Do(func() {
					builtKernels[k] = struct{}{}
					k.install(c)
					used = true
				})
			}
			if !used {
				c.release()
			}
		}
		warmMu.Unlock()
		submitMu.Unlock()
	}
}
