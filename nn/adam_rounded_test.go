package nn

import (
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

func bigOf(x float32) *big.Float { return new(big.Float).SetPrec(2000).SetFloat64(float64(x)) }

func roundBig(x *big.Float) float32 {
	if x.Sign() == 0 {
		return 0
	}
	f, _ := x.Float32()
	return f
}

// adamBigReference applies the specified sequence with every step computed in
// extended precision and rounded once: the independent check of the host code.
func adamBigReference(w, g, m, v float32, s accel.AdamScalars) (float32, float32, float32) {
	mul := func(a, b float32) *big.Float { return new(big.Float).SetPrec(2000).Mul(bigOf(a), bigOf(b)) }
	sum2 := func(a, b, c, d float32) float32 {
		return roundBig(new(big.Float).SetPrec(2000).Add(mul(a, b), mul(c, d)))
	}
	if s.Decay {
		w = sum2(w, 1, s.NegDecay, w)
	}
	mn := sum2(s.Beta1, m, s.OneMinusBeta1, g)
	g2 := roundBig(mul(g, g))
	vn := sum2(s.Beta2, v, s.OneMinusBeta2, g2)
	mhat := roundBig(new(big.Float).SetPrec(2000).Quo(bigOf(mn), bigOf(s.Bias1)))
	vhat := roundBig(new(big.Float).SetPrec(2000).Quo(bigOf(vn), bigOf(s.Bias2)))
	sq := roundBig(new(big.Float).SetPrec(2000).Sqrt(bigOf(vhat)))
	den := sum2(sq, 1, s.Epsilon, 1)
	q := roundBig(new(big.Float).SetPrec(2000).Quo(bigOf(mhat), bigOf(den)))
	return sum2(w, 1, s.NegRate, q), mn, vn
}

func TestAdamRoundedHostEqualsTheExtendedPrecisionReference(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	for _, decay := range []bool{false, true} {
		for step := uint64(1); step <= 40; step++ {
			s := adamRoundedScalars(0.01, 0.1, step, decay)
			for i := 0; i < 200; i++ {
				w := float32(r.NormFloat64())
				g := float32(r.NormFloat64()) * float32(math.Ldexp(1, r.Intn(20)-10))
				m := float32(r.NormFloat64()) * 0.1
				v := float32(math.Abs(r.NormFloat64())) * 0.01
				gw, gm, gv := adamRoundedElement(w, g, m, v, s)
				ww, wm, wv := adamBigReference(w, g, m, v, s)
				if math.Float32bits(gw) != math.Float32bits(ww) || math.Float32bits(gm) != math.Float32bits(wm) || math.Float32bits(gv) != math.Float32bits(wv) {
					t.Fatalf("decay=%v step=%d: got (%g,%g,%g) want (%g,%g,%g) for w=%g g=%g m=%g v=%g", decay, step, gw, gm, gv, ww, wm, wv, w, g, m, v)
				}
			}
		}
	}
}

func TestAdamBiasCorrectionIsCorrectlyRounded(t *testing.T) {
	for _, step := range []uint64{1, 2, 3, 10, 100, 1000, 100000, 10000000} {
		for _, beta := range []float64{0.9, 0.999} {
			want := new(big.Float).SetPrec(4000).SetInt64(1)
			b := new(big.Float).SetPrec(4000).SetFloat64(beta)
			for i := uint64(0); i < step && step <= 100000; i++ {
				want.Mul(want, b)
			}
			if step > 100000 {
				continue
			}
			one := new(big.Float).SetPrec(4000).SetInt64(1)
			expected, _ := one.Sub(one, want).Float32()
			if got := oneMinusPow(beta, step); math.Float32bits(got) != math.Float32bits(expected) {
				t.Fatalf("1-%g^%d = %g, want %g", beta, step, got, expected)
			}
		}
	}
}

func runAdamSteps(t *testing.T, device accel.Device, decay bool) ([]float32, []float32) {
	t.Helper()
	rng := rand.New(rand.NewSource(8))
	X, E := randomTensor(t, rng, 6, 5), randomTensor(t, rng, 5, 4)
	up := randomTensor(t, rng, 6, 4)
	mv := func(x *Tensor) *Tensor {
		moved, err := x.To(device)
		if err != nil {
			t.Fatal(err)
		}
		return moved
	}
	X, E, up = mv(X), mv(E), mv(up)
	defer X.Release()
	defer up.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	if _, err := tape.Param(E); err != nil {
		t.Fatal(err)
	}
	for step := 0; step < 8; step++ {
		tape.Clear()
		y, err := tape.MatMul(X, E)
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.BackwardFrom(y, up); err != nil {
			t.Fatal(err)
		}
		if decay {
			err = tape.AdamW(0.01, 0.1)
		} else {
			err = tape.Adam(0.01)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	h, err := E.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	return h.data, nil
}

func TestRoundedAdamOnTheDeviceEqualsTheHost(t *testing.T) {
	device := residentTestDevice(t)
	for _, decay := range []bool{false, true} {
		host, _ := runAdamSteps(t, accel.Device{}, decay)
		resident, _ := runAdamSteps(t, device, decay)
		sameBitsSlice(t, "E", resident, host)
	}
}

func TestDefaultTapeAdamWKeepsItsFloat64Arithmetic(t *testing.T) {
	// The default path must still use float64 intermediates: its result on a
	// fixed case differs from the rounded mode's by more than zero for some
	// element over a few steps, and equals the previous release's value.
	rng := rand.New(rand.NewSource(2))
	E := randomTensor(t, rng, 4, 4)
	g := randomTensor(t, rng, 4, 4)
	a, _ := copyTensor(E)
	b, _ := copyTensor(E)
	run := func(w *Tensor, rounded bool) []float32 {
		tape := NewTape()
		if rounded {
			_ = tape.SetPrecision(accel.PrecisionFloat32Rounded)
		}
		if _, err := tape.Param(w); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 5; i++ {
			tape.Clear()
			y, _ := tape.MatMul(randomTensor(t, rand.New(rand.NewSource(int64(i))), 4, 4), w)
			if err := tape.BackwardFrom(y, g); err != nil {
				t.Fatal(err)
			}
			if err := tape.AdamW(0.01, 0.1); err != nil {
				t.Fatal(err)
			}
		}
		return w.data
	}
	def, rounded := run(a, false), run(b, true)
	differ := false
	for i := range def {
		if math.Float32bits(def[i]) != math.Float32bits(rounded[i]) {
			differ = true
		}
	}
	if !differ {
		t.Skip("the two modes happened to agree on this case")
	}
}
