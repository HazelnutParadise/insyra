package nn

import (
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

// The reference below follows the specified sequence line by line with the
// independent oracles for exp, log1p and expm1 and big-float exact sums.

func bigSum(a, b []float32) float32 {
	const prec = 3000
	total := new(big.Float).SetPrec(prec)
	for i := range a {
		x := new(big.Float).SetPrec(prec).SetFloat64(float64(a[i]))
		y := new(big.Float).SetPrec(prec).SetFloat64(float64(b[i]))
		total.Add(total, x.Mul(x, y))
	}
	if total.Sign() == 0 {
		return 0
	}
	f, _ := total.Float32()
	return f
}

type refSoftmaxRow struct {
	peak      int
	d, e, q   []float32
	o, s, ell float32
}

func refRow(x []float32) refSoftmaxRow {
	r := refSoftmaxRow{}
	for c, v := range x {
		if v > x[r.peak] {
			r.peak = c
		}
	}
	ones := func(n int) []float32 {
		o := make([]float32, n)
		for i := range o {
			o[i] = 1
		}
		return o
	}
	r.d, r.e, r.q = make([]float32, len(x)), make([]float32, len(x)), make([]float32, len(x))
	for c, v := range x {
		r.d[c] = bigSum([]float32{v, x[r.peak]}, []float32{1, -1})
		r.e[c] = expLogOracleExp(r.d[c])
	}
	r.s = bigSum(r.e, ones(len(x)))
	var others []float32
	others = append(others, r.e[:r.peak]...)
	others = append(others, r.e[r.peak+1:]...)
	r.o = bigSum(others, ones(len(others)))
	r.ell = log1pOracle(r.o)
	for c := range x {
		r.q[c] = r.e[c] / r.s
	}
	return r
}

func refSoftLoss(p []float32, r refSoftmaxRow) float32 {
	var a, b []float32
	for c := range p {
		a = append(a, p[c], p[c])
		b = append(b, r.ell, -r.d[c])
	}
	return bigSum(a, b)
}

func refSoftGradient(p []float32, r refSoftmaxRow, mass, scale float32) []float32 {
	out := make([]float32, len(p))
	var u []float32
	for c := range p {
		if c == r.peak {
			continue
		}
		v := bigSum([]float32{r.q[c], -p[c]}, []float32{mass, 1})
		out[c] = mulRounded(v, scale)
		u = append(u, v)
	}
	ones := make([]float32, len(u))
	for i := range ones {
		ones[i] = 1
	}
	out[r.peak] = mulRounded(-bigSum(u, ones), scale)
	return out
}

func refDistillGradient(zS, zT []float32, scale float32) []float32 {
	s, t := refRow(zS), refRow(zT)
	p, q := t.q, s.q
	n := len(p)
	plain := func() []float32 { return refSoftGradient(p, s, 1, scale) }
	for c := range p {
		if !(p[c] > 0) {
			return plain()
		}
	}
	delta := make([]float32, n)
	for c := range delta {
		delta[c] = bigSum([]float32{zS[c], zS[0], zT[c], zT[0]}, []float32{1, -1, -1, 1})
	}
	mu := bigSum(p, delta)
	d := make([]float32, n)
	for c := range d {
		d[c] = bigSum([]float32{delta[c], mu}, []float32{1, -1})
		if !(math.Abs(float64(d[c])) <= 1) {
			return plain()
		}
	}
	e := make([]float32, n)
	for c := range e {
		e[c] = expm1Oracle(d[c])
	}
	sigma := log1pOracle(bigSum(p, e))
	u := make([]float32, n)
	ones := make([]float32, n)
	for c := range u {
		u[c] = mulRounded(p[c], expm1Oracle(bigSum([]float32{d[c], sigma}, []float32{1, -1})))
		ones[c] = 1
	}
	total := bigSum(u, ones)
	out := make([]float32, n)
	for c := range out {
		out[c] = mulRounded(bigSum([]float32{u[c], -q[c]}, []float32{1, total}), scale)
	}
	return out
}

func softCERoundedCases() (logits, other [][]float32) {
	rng := rand.New(rand.NewSource(404))
	row := func(n int, spread float64) []float32 {
		x := make([]float32, n)
		for i := range x {
			x[i] = float32(rng.NormFloat64() * spread)
		}
		return x
	}
	for i := 0; i < 12; i++ {
		logits = append(logits, row(5, 2))
		other = append(other, row(5, 2))
	}
	// A label far past the cliff, a perturbation beside a large logit, equal
	// rows, a wide teacher range, a one-class row and ties.
	logits = append(logits, []float32{40, 0, -3}, []float32{1e-20, 0, 1}, []float32{0.5, -1, 2}, []float32{0, 1, 0}, []float32{2}, []float32{1, 1, 0}, []float32{90, 0, 0, 3})
	other = append(other, []float32{39, 0.5, -3}, []float32{0, 0, 1}, []float32{0.5, -1, 2}, []float32{1000, 0, 0}, []float32{3}, []float32{1, 0, 1}, []float32{90.5, 0, 0, 3})
	return
}

func normalised(x []float32) []float32 {
	r := refRow(x)
	return r.q
}

func TestRoundedSoftTargetAndDistillationFollowTheSpecifiedSequence(t *testing.T) {
	logits, others := softCERoundedCases()
	for i := range logits {
		zS, zT := logits[i], others[i]
		classes := len(zS)
		scale := float32(-0.75) / 3
		// Soft target: the second operand as a probability row.
		p := normalised(zT)
		r := refRow(zS)
		if got, want := softLossRow(p, roundedSoftmaxRowOf(zS)), refSoftLoss(p, r); math.Float32bits(got) != math.Float32bits(want) {
			t.Errorf("case %d soft loss = %g (%#x), want %g (%#x)", i, got, math.Float32bits(got), want, math.Float32bits(want))
		}
		out := make([]float32, classes)
		mass := bigSum(p, onesLike(p))
		softGradientRow(out, p, roundedSoftmaxRowOf(zS), mass, scale)
		for c, want := range refSoftGradient(p, r, mass, scale) {
			if math.Float32bits(out[c]) != math.Float32bits(want) {
				t.Errorf("case %d soft grad[%d] = %g, want %g", i, c, out[c], want)
			}
		}
		// Distillation.
		dist := make([]float32, classes)
		distillationRowGradient(dist, zS, zT, roundedSoftmaxRowOf(zS), roundedSoftmaxRowOf(zT), scale)
		for c, want := range refDistillGradient(zS, zT, scale) {
			if math.Float32bits(dist[c]) != math.Float32bits(want) {
				t.Errorf("case %d distill grad[%d] = %g (%#x), want %g (%#x)", i, c, dist[c], math.Float32bits(dist[c]), want, math.Float32bits(want))
			}
		}
	}
}

func onesLike(x []float32) []float32 {
	o := make([]float32, len(x))
	for i := range o {
		o[i] = 1
	}
	return o
}

// The loss of a row whose label leads by far past the float32 cliff no longer
// reads 0, and the gradient survives, on the host in PrecisionFloat32Rounded.
func TestRoundedLossesPastTheCliff(t *testing.T) {
	logits, _ := NewTensor([]int{1, 2}, []float32{40, 0})
	labels, _ := NewInt64Tensor([]int{1}, []int64{0})
	loss, err := softmaxCrossEntropyRoundedForward(logits, labels)
	if err != nil {
		t.Fatal(err)
	}
	want := float32(math.Log1p(math.Exp(-40)))
	if loss.data[0] != want {
		t.Errorf("hard loss = %g, want %g", loss.data[0], want)
	}
	targets, _ := NewTensor([]int{1, 2}, []float32{1, 0})
	soft, err := softTargetRoundedForward(logits, targets)
	if err != nil {
		t.Fatal(err)
	}
	if soft.data[0] != want {
		t.Errorf("soft loss = %g, want %g", soft.data[0], want)
	}
}

func TestRoundedDistillationOfEqualLogitsIsExactlyZero(t *testing.T) {
	z, _ := NewTensor([]int{2, 3}, []float32{1, 2, 3, -5, 0.25, 7})
	up, _ := NewTensor(nil, []float32{1})
	g, err := distillationRoundedVJP(z, z, up)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range g.data {
		if math.Float32bits(v) != 0 {
			t.Errorf("grad[%d] = %g (%#x), want +0", i, v, math.Float32bits(v))
		}
	}
}

func TestRoundedSoftLossesRefuseABadShape(t *testing.T) {
	a, _ := NewTensor([]int{2, 2}, make([]float32, 4))
	b, _ := NewTensor([]int{3, 2}, make([]float32, 6))
	if _, err := softTargetRoundedForward(a, b); err == nil {
		t.Error("shape mismatch: want an error")
	}
	_ = accel.PrecisionFloat32Rounded
}
