package nn

import (
	"math"
	"os"
	"sync"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

// TestDeviceExpAndLogEqualTheHost compares the device's exp and log with the
// host's on every float32 bit pattern, in chunks. It needs a GPU and
// INSYRA_NN_EXHAUSTIVE=1; with INSYRA_NN_SAMPLE=1 it compares a sample instead.
func TestDeviceExpAndLogEqualTheHost(t *testing.T) {
	device := residentTestDevice(t)
	exhaustive := os.Getenv("INSYRA_NN_EXHAUSTIVE") == "1"
	if !exhaustive && os.Getenv("INSYRA_NN_SAMPLE") != "1" {
		t.Skip("set INSYRA_NN_EXHAUSTIVE=1 (all 2^32 inputs) or INSYRA_NN_SAMPLE=1")
	}
	const chunk = 1 << 22
	step := uint64(chunk)
	stride := uint64(1)
	if !exhaustive {
		stride = 61 // a prime stride visits chunks across the whole range
	}
	for _, name := range []string{"exp", "log"} {
		var mismatches int
		inputs := make([]float32, chunk)
		for base := uint64(0); base < 1<<32; base += step * stride {
			for i := range inputs {
				inputs[i] = math.Float32frombits(uint32(base) + uint32(i))
			}
			host, err := NewTensor([]int{chunk}, inputs)
			if err != nil {
				t.Fatal(err)
			}
			resident, err := host.To(device)
			if err != nil {
				t.Fatal(err)
			}
			var out *Tensor
			if name == "exp" {
				buffer, err := accel.RoundedExp(resident.device)
				if err != nil {
					t.Fatal(err)
				}
				out, err = newResidentTensor([]int{chunk}, buffer)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				buffer, err := accel.RoundedLog(resident.device)
				if err != nil {
					t.Fatal(err)
				}
				out, err = newResidentTensor([]int{chunk}, buffer)
				if err != nil {
					t.Fatal(err)
				}
			}
			got, err := out.ToHost()
			if err != nil {
				t.Fatal(err)
			}
			var wg sync.WaitGroup
			var mu sync.Mutex
			workers := 8
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func(w int) {
					defer wg.Done()
					for i := w; i < chunk; i += workers {
						var want float32
						if name == "exp" {
							want = expFloat32(inputs[i])
						} else {
							want = logFloat32(inputs[i])
						}
						g := got.data[i]
						if math.IsNaN(float64(want)) && math.IsNaN(float64(g)) {
							continue
						}
						if math.Float32bits(g) != math.Float32bits(want) {
							mu.Lock()
							if mismatches < 10 {
								t.Errorf("%s(%#08x) = %#08x on the device, %#08x on the host", name, math.Float32bits(inputs[i]), math.Float32bits(g), math.Float32bits(want))
							}
							mismatches++
							mu.Unlock()
						}
					}
				}(w)
			}
			wg.Wait()
			resident.Release()
			out.Release()
		}
		if mismatches > 0 {
			t.Fatalf("%s: %d mismatches", name, mismatches)
		}
	}
}
