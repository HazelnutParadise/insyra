package nn

import (
	"math"
	"math/big"
	"strings"
	"testing"
)

// softCEReference is an independent computation in 200-bit arithmetic: the
// loss -sum p_k*log(q_k) and the gradient q_k - p_k per row.
func softCEReference(t *testing.T, z []float32, p []float64) (float64, []float64) {
	t.Helper()
	const prec = 200
	exp := func(x *big.Float) *big.Float {
		// e^x by halving and Taylor series; x is moderate after max-shift.
		halves := 0
		y := new(big.Float).SetPrec(prec).Set(x)
		for y.Cmp(big.NewFloat(0.5)) > 0 || y.Cmp(big.NewFloat(-0.5)) < 0 {
			y.Quo(y, big.NewFloat(2))
			halves++
		}
		sum, term := new(big.Float).SetPrec(prec).SetInt64(1), new(big.Float).SetPrec(prec).SetInt64(1)
		for n := int64(1); n < 80; n++ {
			term.Mul(term, y)
			term.Quo(term, new(big.Float).SetPrec(prec).SetInt64(n))
			sum.Add(sum, term)
		}
		for ; halves > 0; halves-- {
			sum.Mul(sum, sum)
		}
		return sum
	}
	maxZ := z[0]
	for _, v := range z {
		if v > maxZ {
			maxZ = v
		}
	}
	e := make([]*big.Float, len(z))
	total := new(big.Float).SetPrec(prec)
	for i, v := range z {
		e[i] = exp(new(big.Float).SetPrec(prec).SetFloat64(float64(v) - float64(maxZ)))
		total.Add(total, e[i])
	}
	q := make([]float64, len(z))
	for i := range z {
		r := new(big.Float).SetPrec(prec).Quo(e[i], total)
		q[i], _ = r.Float64()
	}
	logTotal, _ := total.Float64()
	lse := float64(maxZ) + math.Log(logTotal)
	var loss float64
	for i := range z {
		if p[i] != 0 {
			loss -= p[i] * (float64(z[i]) - lse)
		}
	}
	grad := make([]float64, len(z))
	for i := range z {
		grad[i] = q[i] - p[i]
	}
	return loss, grad
}

func softCEBackward(t *testing.T, tape *Tape, loss *Tensor, wrt *Tensor) *Tensor {
	t.Helper()
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	g, err := tape.Grad(wrt)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestSoftTargetCrossEntropyMatchesTheReference(t *testing.T) {
	z := []float32{2, -1, 0.5, 30, 0, -4}
	p := []float32{0.7, 0.2, 0.1, 0.9, 0.1, 0}
	logits, _ := NewTensor([]int{2, 3}, z)
	targets, _ := NewTensor([]int{2, 3}, p)
	tape := NewTape()
	defer tape.Close()
	if _, err := tape.Param(logits); err != nil {
		t.Fatal(err)
	}
	loss, err := tape.SoftTargetCrossEntropy(logits, targets)
	if err != nil {
		t.Fatal(err)
	}
	var wantLoss float64
	var wantGrad []float64
	for r := 0; r < 2; r++ {
		p64 := []float64{float64(p[r*3]), float64(p[r*3+1]), float64(p[r*3+2])}
		l, g := softCEReference(t, z[r*3:r*3+3], p64)
		wantLoss += l / 2
		for _, v := range g {
			wantGrad = append(wantGrad, v/2)
		}
	}
	if got := float64(loss.data[0]); math.Abs(got-wantLoss) > 1e-6*math.Abs(wantLoss) {
		t.Errorf("loss = %g, want %g", got, wantLoss)
	}
	g := softCEBackward(t, tape, loss, logits)
	for i, want := range wantGrad {
		if got := float64(g.data[i]); math.Abs(got-want) > 1e-6*math.Max(math.Abs(want), 1e-3) {
			t.Errorf("grad[%d] = %g, want %g", i, got, want)
		}
	}
}

func TestDistillationLossMatchesTheReferenceAndIsZeroAtEquality(t *testing.T) {
	student := []float32{1, 2, 3, 0.1, 0.2, 0.3}
	teacher := []float32{1.5, 1, 3.5, 0.1, 0.2, 0.3}
	s, _ := NewTensor([]int{2, 3}, student)
	te, _ := NewTensor([]int{2, 3}, teacher)
	tape := NewTape()
	defer tape.Close()
	if _, err := tape.Param(s); err != nil {
		t.Fatal(err)
	}
	loss, err := tape.DistillationLoss(s, te)
	if err != nil {
		t.Fatal(err)
	}
	var wantLoss float64
	var wantGrad []float64
	for r := 0; r < 2; r++ {
		pt := make([]float64, 3)
		_ = softmaxRow64(teacher[r*3:r*3+3], pt, nil)
		l, g := softCEReference(t, student[r*3:r*3+3], pt)
		wantLoss += l / 2
		for _, v := range g {
			wantGrad = append(wantGrad, v/2)
		}
	}
	if got := float64(loss.data[0]); math.Abs(got-wantLoss) > 1e-6*math.Abs(wantLoss) {
		t.Errorf("loss = %g, want %g", got, wantLoss)
	}
	g := softCEBackward(t, tape, loss, s)
	for i := 0; i < 3; i++ {
		if got, want := float64(g.data[i]), wantGrad[i]; math.Abs(got-want) > 1e-6*math.Abs(want) {
			t.Errorf("grad[%d] = %g, want %g", i, got, want)
		}
	}
	for i := 3; i < 6; i++ {
		if g.data[i] != 0 {
			t.Errorf("equal logits: grad[%d] = %g, want exactly 0", i, g.data[i])
		}
	}
	for r := 0; r < 2; r++ {
		if sum := float64(g.data[r*3]) + float64(g.data[r*3+1]) + float64(g.data[r*3+2]); math.Abs(sum) > 1e-7 {
			t.Errorf("row %d sums to %g", r, sum)
		}
	}
}

func TestDistillationGradientNearEquality(t *testing.T) {
	teacher := []float32{3, -2, 1}
	student := []float32{3 + 1e-4, -2, 1}
	s, _ := NewTensor([]int{1, 3}, student)
	te, _ := NewTensor([]int{1, 3}, teacher)
	tape := NewTape()
	defer tape.Close()
	if _, err := tape.Param(s); err != nil {
		t.Fatal(err)
	}
	loss, err := tape.DistillationLoss(s, te)
	if err != nil {
		t.Fatal(err)
	}
	pt := make([]float64, 3)
	_ = softmaxRow64(teacher, pt, nil)
	_, want := softCEReference(t, student, pt)
	g := softCEBackward(t, tape, loss, s)
	for i := range want {
		if got := float64(g.data[i]); got == 0 || math.Abs(got-want[i]) > 1e-4*math.Abs(want[i]) {
			t.Errorf("grad[%d] = %g, want %g", i, got, want[i])
		}
	}
}

func TestDistillationWideTeacherRangeIsFinite(t *testing.T) {
	s, _ := NewTensor([]int{1, 3}, []float32{0, 1, 0})
	te, _ := NewTensor([]int{1, 3}, []float32{1000, 0, 0})
	tape := NewTape()
	defer tape.Close()
	if _, err := tape.Param(s); err != nil {
		t.Fatal(err)
	}
	loss, err := tape.DistillationLoss(s, te)
	if err != nil {
		t.Fatal(err)
	}
	g := softCEBackward(t, tape, loss, s)
	if math.IsNaN(float64(loss.data[0])) || math.IsInf(float64(loss.data[0]), 0) {
		t.Errorf("loss = %g", loss.data[0])
	}
	for i, v := range g.data {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Errorf("grad[%d] = %g", i, v)
		}
	}
}

func TestSoftTargetCrossEntropyLargeMarginKeepsTheGradient(t *testing.T) {
	z, _ := NewTensor([]int{1, 2}, []float32{40, 0})
	p, _ := NewTensor([]int{1, 2}, []float32{1, 0})
	tape := NewTape()
	defer tape.Close()
	if _, err := tape.Param(z); err != nil {
		t.Fatal(err)
	}
	loss, err := tape.SoftTargetCrossEntropy(z, p)
	if err != nil {
		t.Fatal(err)
	}
	g := softCEBackward(t, tape, loss, z)
	want := math.Exp(-40) / (1 + math.Exp(-40))
	if got, wantLoss := float64(loss.data[0]), math.Log1p(math.Exp(-40)); got == 0 || math.Abs(got-wantLoss) > 1e-6*wantLoss {
		t.Errorf("loss = %g, want %g", got, wantLoss)
	}
	if got := float64(g.data[1]); math.Abs(got-want) > 1e-6*want {
		t.Errorf("grad[1] = %g, want %g", got, want)
	}
	if got := float64(g.data[0]); got == 0 || math.Abs(got+want) > 1e-6*want {
		t.Errorf("grad[0] = %g, want %g", got, -want)
	}
}

func TestSoftCrossEntropyRefusesBadInput(t *testing.T) {
	z, _ := NewTensor([]int{2, 2}, []float32{0, 1, 2, 3})
	cases := map[string][]float32{
		"sums to":    {0.5, 0.2, 0.5, 0.5},
		"negative":   {1.5, -0.5, 0.5, 0.5},
		"not finite": {float32(math.NaN()), 1, 0.5, 0.5},
	}
	for want, data := range cases {
		targets, _ := NewTensor([]int{2, 2}, data)
		tape := NewTape()
		if _, err := tape.SoftTargetCrossEntropy(z, targets); err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "row 0") {
			t.Errorf("%q: error = %v", want, err)
		}
		tape.Close()
	}
	other, _ := NewTensor([]int{3, 2}, make([]float32, 6))
	tape := NewTape()
	defer tape.Close()
	if _, err := tape.DistillationLoss(z, other); err == nil {
		t.Error("shape mismatch: want an error")
	}
}

// Differences far below float64's resolution of a probability near 1/2 still
// give a gradient (Codex on #564).
func TestDistillationGradientOfAVanishingDifference(t *testing.T) {
	s, _ := NewTensor([]int{1, 2}, []float32{1e-20, 0})
	te, _ := NewTensor([]int{1, 2}, []float32{0, 0})
	tape := NewTape()
	defer tape.Close()
	if _, err := tape.Param(s); err != nil {
		t.Fatal(err)
	}
	loss, err := tape.DistillationLoss(s, te)
	if err != nil {
		t.Fatal(err)
	}
	g := softCEBackward(t, tape, loss, s)
	if got := float64(g.data[0]); math.Abs(got-2.5e-21) > 1e-6*2.5e-21 {
		t.Errorf("grad[0] = %g, want 2.5e-21", got)
	}
	if got := float64(g.data[1]); math.Abs(got+2.5e-21) > 1e-6*2.5e-21 {
		t.Errorf("grad[1] = %g, want -2.5e-21", got)
	}
}

// A large common offset in the logits changes nothing (Codex on #564).
func TestSoftTargetCrossEntropyIgnoresACommonOffset(t *testing.T) {
	z, _ := NewTensor([]int{1, 2}, []float32{1e20, 1e20})
	p, _ := NewTensor([]int{1, 2}, []float32{0, 1})
	tape := NewTape()
	defer tape.Close()
	if _, err := tape.Param(z); err != nil {
		t.Fatal(err)
	}
	loss, err := tape.SoftTargetCrossEntropy(z, p)
	if err != nil {
		t.Fatal(err)
	}
	g := softCEBackward(t, tape, loss, z)
	if got := float64(loss.data[0]); math.Abs(got-math.Ln2) > 1e-6 {
		t.Errorf("loss = %g, want ln 2", got)
	}
	if g.data[0] != 0.5 || g.data[1] != -0.5 {
		t.Errorf("grad = %v, want [0.5 -0.5]", g.data)
	}
}

// A common offset in the student or teacher row changes nothing (Codex on #564).
func TestDistillationIgnoresCommonOffsets(t *testing.T) {
	grad := func(student, teacher []float32) []float32 {
		s, _ := NewTensor([]int{1, 2}, student)
		te, _ := NewTensor([]int{1, 2}, teacher)
		tape := NewTape()
		defer tape.Close()
		if _, err := tape.Param(s); err != nil {
			t.Fatal(err)
		}
		loss, err := tape.DistillationLoss(s, te)
		if err != nil {
			t.Fatal(err)
		}
		return softCEBackward(t, tape, loss, s).data
	}
	got := grad([]float32{1e20, 1e20}, []float32{1, 0})
	want := grad([]float32{0, 0}, []float32{1, 0})
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("grad = %v, want %v", got, want)
			break
		}
	}
	if math.Abs(float64(got[0])+0.23105858) > 1e-6 {
		t.Errorf("grad[0] = %g, want -0.23105858", got[0])
	}
}
