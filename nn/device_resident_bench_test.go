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

// BenchmarkClassifierStep times one forward, backward and AdamW step of a
// two-layer classifier: the default tape on the host, the rounded mode on the
// host, and the rounded mode on a device.
func BenchmarkClassifierStep(b *testing.B) {
	rows, in, hidden, classes := 1024, 96, 128, 96
	rng := rand.New(rand.NewSource(1))
	mk := func(shape ...int) *Tensor {
		n := 1
		for _, d := range shape {
			n *= d
		}
		data := make([]float32, n)
		for i := range data {
			data[i] = float32(rng.NormFloat64()) * 0.1
		}
		x, _ := NewTensor(shape, data)
		return x
	}
	X, W1, W2 := mk(rows, in), mk(in, hidden), mk(hidden, classes)
	labels := make([]int64, rows)
	for i := range labels {
		labels[i] = int64(rng.Intn(classes))
	}
	L, _ := NewInt64Tensor([]int{rows}, labels)
	run := func(b *testing.B, device accel.Device, rounded bool) {
		x, _ := X.To(device)
		w1, _ := W1.To(device)
		w2, _ := W2.To(device)
		l, _ := L.To(device)
		tape := NewTape()
		if rounded {
			_ = tape.SetPrecision(accel.PrecisionFloat32Rounded)
		}
		_, _ = tape.Param(w1)
		_, _ = tape.Param(w2)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tape.Clear()
			h, err := tape.MatMul(x, w1)
			if err != nil {
				b.Fatal(err)
			}
			y, err := tape.MatMul(h, w2)
			if err != nil {
				b.Fatal(err)
			}
			loss, err := tape.SoftmaxCrossEntropy(y, l)
			if err != nil {
				b.Fatal(err)
			}
			if err := tape.Backward(loss); err != nil {
				b.Fatal(err)
			}
			if err := tape.AdamW(0.001, 0.01); err != nil {
				b.Fatal(err)
			}
			if device.ID != "" {
				if err := accel.SynchronizeResident(); err != nil {
					b.Fatal(err)
				}
			}
		}
		b.StopTimer()
		tape.Close()
		x.Release()
		l.Release()
	}
	b.Run("host-default", func(b *testing.B) { run(b, accel.Device{}, false) })
	b.Run("host-rounded", func(b *testing.B) { run(b, accel.Device{}, true) })
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") == "1" {
		if device := accel.DefaultDevice(); device.ID != "" {
			b.Run("device-rounded", func(b *testing.B) { run(b, device, true) })
		}
	}
}
