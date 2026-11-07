package nn

import (
	"math"
	"math/rand"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

// runRoundedTanhSteps trains two SGD steps of tanh(X·E + b)·R and returns the
// final numbers. device is the zero Device for the host run.
func runRoundedTanhSteps(t *testing.T, device accel.Device, seed int64) stepResult {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	X, E, R, bias := randomTensor(t, rng, 5, 8), randomTensor(t, rng, 8, 6), randomTensor(t, rng, 6, 4), randomTensor(t, rng, 6)
	upstream := randomTensor(t, rng, 5, 4)
	move := func(x *Tensor) *Tensor {
		moved, err := x.To(device)
		if err != nil {
			t.Fatal(err)
		}
		return moved
	}
	X, E, R, bias, upstream = move(X), move(E), move(R), move(bias), move(upstream)
	defer X.Release()
	defer upstream.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*Tensor{E, R, bias} {
		if _, err := tape.Param(p); err != nil {
			t.Fatal(err)
		}
	}
	host := func(x *Tensor) []float32 {
		h, err := x.ToHost()
		if err != nil {
			t.Fatal(err)
		}
		return h.data
	}
	var out *Tensor
	for step := 0; step < 2; step++ {
		tape.Clear()
		h, err := tape.MatMul(X, E)
		if err != nil {
			t.Fatal(err)
		}
		pre, err := tape.Add(h, bias)
		if err != nil {
			t.Fatal(err)
		}
		hidden, err := tape.Tanh(pre)
		if err != nil {
			t.Fatal(err)
		}
		if hidden.onDevice() != (device.ID != "") {
			t.Fatalf("tanh output on the device: %v, want %v", hidden.onDevice(), device.ID != "")
		}
		out, err = tape.MatMul(hidden, R)
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.BackwardFrom(out, upstream); err != nil {
			t.Fatal(err)
		}
		if step == 0 {
			continue
		}
		if err := tape.SGD(0.05); err != nil {
			t.Fatal(err)
		}
	}
	grad, err := tape.Grad(E)
	if err != nil {
		t.Fatal(err)
	}
	result := stepResult{output: host(out), e: host(E), r: host(R), bias: host(bias), gradE: host(grad)}
	grad.Release()
	return result
}

func TestRoundedTanhStepOnTheDeviceEqualsTheHostBitForBit(t *testing.T) {
	device := residentTestDevice(t)
	for seed := int64(1); seed <= 4; seed++ {
		host := runRoundedTanhSteps(t, accel.Device{}, seed)
		resident := runRoundedTanhSteps(t, device, seed)
		sameBitsSlice(t, "output", resident.output, host.output)
		sameBitsSlice(t, "E", resident.e, host.e)
		sameBitsSlice(t, "R", resident.r, host.r)
		sameBitsSlice(t, "bias", resident.bias, host.bias)
		sameBitsSlice(t, "grad E", resident.gradE, host.gradE)
	}
}

func TestRoundedTanhGradientOnTheDeviceEqualsTheHost(t *testing.T) {
	device := residentTestDevice(t)
	negZero := float32(math.Copysign(0, -1))
	inputs := []float32{0, negZero, 20, -20, 0.5, -3, float32(math.Inf(1)), float32(math.NaN()), 1e-40}
	ups := []float32{-1, -2, -0.5, 3, negZero, 1e-38, -1e-30, 2, 7}
	rng := rand.New(rand.NewSource(5))
	for i := 0; i < 4000; i++ {
		inputs = append(inputs, float32(rng.NormFloat64()*4))
		ups = append(ups, float32(rng.NormFloat64()*math.Ldexp(1, rng.Intn(80)-40)))
	}
	run := func(dev accel.Device) ([]float32, []float32) {
		x, err := NewTensor([]int{len(inputs)}, inputs)
		if err != nil {
			t.Fatal(err)
		}
		u, err := NewTensor([]int{len(ups)}, ups)
		if err != nil {
			t.Fatal(err)
		}
		if x, err = x.To(dev); err != nil {
			t.Fatal(err)
		}
		defer x.Release()
		if u, err = u.To(dev); err != nil {
			t.Fatal(err)
		}
		defer u.Release()
		tape := NewTape()
		defer tape.Close()
		if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
			t.Fatal(err)
		}
		if _, err := tape.Param(x); err != nil {
			t.Fatal(err)
		}
		y, err := tape.Tanh(x)
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.BackwardFrom(y, u); err != nil {
			t.Fatal(err)
		}
		g, err := tape.Grad(x)
		if err != nil {
			t.Fatal(err)
		}
		yh, err := y.ToHost()
		if err != nil {
			t.Fatal(err)
		}
		gh, err := g.ToHost()
		if err != nil {
			t.Fatal(err)
		}
		g.Release()
		return yh.data, gh.data
	}
	hostY, hostG := run(accel.Device{})
	devY, devG := run(device)
	same := func(name string, got, want []float32) {
		for i := range want {
			if math.IsNaN(float64(want[i])) && math.IsNaN(float64(got[i])) {
				continue
			}
			if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
				t.Fatalf("%s[%d] (x=%g, u=%g) = %#x on the device, %#x on the host", name, i, inputs[i], ups[i], math.Float32bits(got[i]), math.Float32bits(want[i]))
			}
		}
	}
	same("tanh", devY, hostY)
	same("gradient", devG, hostG)
}

// In PrecisionFloat32Rounded a zero gradient is +0, as every other operation
// in that mode gives; the default tape keeps IEEE's -0.
func TestRoundedTanhGradientZeroIsPositive(t *testing.T) {
	x, err := NewTensor([]int{2}, []float32{20, -20}) // tanh rounds to +-1, so 1 - y*y is 0
	if err != nil {
		t.Fatal(err)
	}
	u, err := NewTensor([]int{2}, []float32{-1, -3})
	if err != nil {
		t.Fatal(err)
	}
	for _, rounded := range []bool{false, true} {
		tape := NewTape()
		if rounded {
			if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := tape.Param(x); err != nil {
			t.Fatal(err)
		}
		y, err := tape.Tanh(x)
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.BackwardFrom(y, u); err != nil {
			t.Fatal(err)
		}
		g, err := tape.Grad(x)
		if err != nil {
			t.Fatal(err)
		}
		for i, v := range g.data {
			if v != 0 || math.Signbit(float64(v)) == rounded {
				t.Errorf("rounded=%v: gradient[%d] = %g (sign bit %v)", rounded, i, v, math.Signbit(float64(v)))
			}
		}
		tape.Close()
	}
}
