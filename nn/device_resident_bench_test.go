package nn

import (
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

// BenchmarkRoundedMatMul measures one product at the shapes of the issue's
// encoder/readout step. host-fast is the default all-core float32 MatMul,
// host-rounded the exact CPU reference, device the resident exact kernel
// (kernel plus a one-value readback to wait for it).
func BenchmarkRoundedMatMul(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	for _, shape := range [][3]int{{4096, 96, 128}, {4096, 128, 96}, {1024, 256, 256}} {
		m, k, n := shape[0], shape[1], shape[2]
		mk := func(r, c int) *Tensor {
			data := make([]float32, r*c)
			for i := range data {
				data[i] = float32(rng.NormFloat64())
			}
			x, _ := NewTensor([]int{r, c}, data)
			return x
		}
		a, w := mk(m, k), mk(k, n)
		name := func(kind string) string {
			return kind + "/" + strconv.Itoa(m) + "x" + strconv.Itoa(k) + "x" + strconv.Itoa(n)
		}
		b.Run(name("host-fast"), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := matMul2DWithWorkers(a, w, 0); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(name("host-rounded"), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := matMulRounded(a, w, false, false); err != nil {
					b.Fatal(err)
				}
			}
		})
		if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
			continue
		}
		device := accel.DefaultDevice()
		if device.ID == "" {
			continue
		}
		ra, _ := a.To(device)
		rw, _ := w.To(device)
		b.Run(name("device"), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tape := NewTape()
				_ = tape.SetPrecision(accel.PrecisionFloat32Rounded)
				out, err := tape.MatMul(ra, rw)
				if err != nil {
					b.Fatal(err)
				}
				if err := accel.SynchronizeResident(); err != nil {
					b.Fatal(err)
				}
				tape.Clear()
				_ = out
			}
		})
		ra.Release()
		rw.Release()
	}
}
