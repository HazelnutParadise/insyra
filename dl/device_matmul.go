package dl

import "sync/atomic"

// DeviceMatMul is the optional device implementation for a two-dimensional
// float32 matrix product. Returning an error makes MatMul use its CPU
// implementation.
type DeviceMatMul func(a []float32, aRows, aCols int, b []float32, bRows, bCols int) ([]float32, error)

// deviceMatMulMACFloor is measured, not guessed. The hardware ladder on the
// 8-core M3 / Metal (best of 5, upload+dispatch+readback included) crosses
// over near 4M MACs, but 4M–8M sit inside the noise band (device/CPU 0.90 and
// 0.96); 16M is the first rung that wins consistently (0.74, improving to
// 0.135 at 268M). Below the floor the all-core CPU path is faster or the win
// is not dependable.
const deviceMatMulMACFloor = 16_777_216

var deviceMatMulHook atomic.Value // of DeviceMatMul

// RegisterDeviceMatMul installs or clears the optional device MatMul hook.
// Passing nil restores the dependency-free CPU path.
func RegisterDeviceMatMul(fn DeviceMatMul) {
	deviceMatMulHook.Store(fn)
}

func registeredDeviceMatMul() DeviceMatMul {
	stored := deviceMatMulHook.Load()
	if stored == nil {
		return nil
	}
	fn, _ := stored.(DeviceMatMul)
	return fn
}

func matMulMACsAtLeast(factors ...int) bool {
	work := 1
	for _, factor := range factors {
		if factor <= 0 {
			return false
		}
		if work > deviceMatMulMACFloor/factor {
			return true
		}
		work *= factor
	}
	return work >= deviceMatMulMACFloor
}
