package wgpu

import "sync"

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

// warmMu is held while warmup compiles, without submitMu, and while acquire
// releases a lost device, so a device is never released under a compile.
var warmMu sync.Mutex

// deviceGeneration counts the lost devices acquire has released. Guarded by
// warmMu: warmup compares it to know whether the device it compiled on is
// still there before releasing what it compiled.
var deviceGeneration int

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

// warm builds each built-in kernel on h that is not built yet. It compiles
// without holding submitMu, so computations, uploads and readbacks are not
// queued behind it, and takes the lock only to install the result. When a
// computation built the kernel first, or the device was retired meanwhile,
// warm releases what it compiled. A kernel warm fails to compile is left for
// its first use, which reports the error.
func warm(h *handle) {
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
		generation := deviceGeneration
		c, err := k.compile(h)
		warmMu.Unlock()
		if err != nil {
			continue
		}
		// submitMu is taken without warmMu: a computation holding submitMu
		// can take warmMu inside acquire.
		used := false
		submitMu.Lock()
		if !h.retired {
			k.once.Do(func() {
				builtKernels[k] = struct{}{}
				k.install(c)
				used = true
			})
		}
		submitMu.Unlock()
		if !used {
			// Released only while its device is: once acquire has released
			// the lost device, its children went with it.
			warmMu.Lock()
			if deviceGeneration == generation {
				c.release()
			}
			warmMu.Unlock()
		}
	}
}
