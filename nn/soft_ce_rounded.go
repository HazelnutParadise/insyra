package nn

import (
	"fmt"
	"math"
)

// The soft-target losses in PrecisionFloat32Rounded. Every line below is one
// correctly rounded operation, or an exact sum of float32 products rounded
// once, in a fixed order, so a device that computes the same lines returns the
// same bits (the sequences are in the nn-rounded-soft-targets spec).

// roundedTerms returns the exact value of the sum of a[i]*b[i], rounded once.
func roundedTerms(a, b []float32) float32 {
	var fast float64ProductSum
	for i := range a {
		fast.add(a[i], b[i])
	}
	if result, ok := fast.result(); ok {
		return result
	}
	var exact exactAccumulator
	for i := range a {
		exact.addProduct(a[i], b[i])
	}
	return exact.float32()
}

// roundedSoftmaxRow holds a row's quantities. peak is the first index of the
// largest logit; d[c] = x[c] - x[peak] and e[c] = exp(d[c]); o is the exact sum
// of the e[c] other than the peak's, rounded once; s the exact sum of all of
// them; ell = log1p(o) is log(s) without 1 + o being rounded first; q[c] = e[c]/s.
type roundedSoftmaxRow struct {
	peak    int
	d, e, q []float32
	o, s    float32
	ell     float32
	finite  bool
}

func roundedSoftmaxRowOf(x []float32) roundedSoftmaxRow {
	row := roundedSoftmaxRow{finite: softmaxCEFiniteRow(x)}
	if !row.finite {
		return row
	}
	for c, v := range x {
		if v > x[row.peak] {
			row.peak = c
		}
	}
	row.d = make([]float32, len(x))
	row.e = make([]float32, len(x))
	row.q = make([]float32, len(x))
	for c, v := range x {
		row.d[c] = sum2Rounded(v, 1, -1, x[row.peak])
		row.e[c] = expFloat32(row.d[c])
	}
	row.s = roundedSumOf(row.e)
	others := make([]float32, 0, len(x))
	others = append(others, row.e[:row.peak]...)
	others = append(others, row.e[row.peak+1:]...)
	row.o = roundedSumOf(others)
	row.ell = log1pFloat32(row.o)
	for c := range x {
		row.q[c] = row.e[c] / row.s
	}
	return row
}

// softLossRow is the exact sum of p[c]*ell - p[c]*d[c] rounded once.
func softLossRow(p []float32, row roundedSoftmaxRow) float32 {
	a := make([]float32, 0, 2*len(p))
	b := make([]float32, 0, 2*len(p))
	for c := range p {
		// A class with probability 0 adds nothing, even when its centred
		// difference overflowed to -Inf and 0 * Inf would be NaN.
		if p[c] == 0 {
			continue
		}
		a = append(a, p[c], p[c])
		b = append(b, row.ell, -row.d[c])
	}
	return roundedTerms(a, b)
}

// softGradientRow writes the gradient of the loss for a row whose target
// probabilities are p, with mass the factor on q (the sum of p for a target
// row, 1 for a teacher's softmax): u[c] = q[c]*mass - p[c] rounded once for
// every class but the peak, whose gradient is minus the rounded sum of the
// others, each scaled by scale.
func softGradientRow(out []float32, p []float32, row roundedSoftmaxRow, mass, scale float32) {
	u := make([]float32, 0, len(p))
	for c := range p {
		if c == row.peak {
			continue
		}
		v := sum2Rounded(row.q[c], mass, -p[c], 1)
		out[c] = mulRounded(v, scale)
		u = append(u, v)
	}
	out[row.peak] = mulRounded(-roundedSumOf(u), scale)
}

func checkSoftCEOperands(op string, logits, other *Tensor) error {
	if err := requireFloat32(logits, op+" logits"); err != nil {
		return err
	}
	if err := requireFloat32(other, op+" second operand"); err != nil {
		return err
	}
	if len(logits.shape) != 2 || !sameShape(logits.shape, other.shape) {
		return fmt.Errorf("%s requires two [N,C] tensors of the same shape, got %v and %v", op, logits.shape, other.shape)
	}
	rows := logits.shape[0]
	if rows == 0 || logits.shape[1] == 0 {
		return fmt.Errorf("%s requires non-empty logits", op)
	}
	if rows >= 1<<24 {
		return fmt.Errorf("%s in PrecisionFloat32Rounded supports fewer than 2^24 rows, got %d", op, rows)
	}
	return nil
}

func meanOfRowLosses(losses []float32) (*Tensor, error) {
	return newFloat32Tensor(nil, []float32{roundedSumOf(losses) / float32(len(losses))})
}

// softTargetRoundedForward is the mean over rows of softLossRow with the
// targets as p.
func softTargetRoundedForward(logits, targets *Tensor) (*Tensor, error) {
	if err := checkSoftCEOperands("soft-target cross-entropy", logits, targets); err != nil {
		return nil, err
	}
	rows, classes := logits.shape[0], logits.shape[1]
	losses := make([]float32, rows)
	for r := range losses {
		row := roundedSoftmaxRowOf(logits.data[r*classes : (r+1)*classes])
		if !row.finite {
			losses[r] = float32(math.NaN())
			continue
		}
		losses[r] = softLossRow(targets.data[r*classes:(r+1)*classes], row)
	}
	return meanOfRowLosses(losses)
}

func softTargetRoundedVJP(logits, targets, upstream *Tensor) (*Tensor, error) {
	if err := requireScalarFloat32(upstream, "soft-target cross-entropy upstream"); err != nil {
		return nil, err
	}
	rows, classes := logits.shape[0], logits.shape[1]
	gradient, err := newZeroFloat32Tensor(logits.shape)
	if err != nil {
		return nil, err
	}
	scale := upstream.data[0] / float32(rows)
	for r := 0; r < rows; r++ {
		out := gradient.data[r*classes : (r+1)*classes]
		row := roundedSoftmaxRowOf(logits.data[r*classes : (r+1)*classes])
		if !row.finite {
			for c := range out {
				out[c] = float32(math.NaN())
			}
			continue
		}
		p := targets.data[r*classes : (r+1)*classes]
		softGradientRow(out, p, row, roundedSumOf(p), scale)
	}
	return gradient, nil
}

// distillationRoundedForward is the mean over rows of softLossRow with the
// teacher's softmax probabilities as p.
func distillationRoundedForward(student, teacher *Tensor) (*Tensor, error) {
	if err := checkSoftCEOperands("distillation loss", student, teacher); err != nil {
		return nil, err
	}
	rows, classes := student.shape[0], student.shape[1]
	losses := make([]float32, rows)
	for r := range losses {
		s := roundedSoftmaxRowOf(student.data[r*classes : (r+1)*classes])
		t := roundedSoftmaxRowOf(teacher.data[r*classes : (r+1)*classes])
		if !s.finite || !t.finite {
			losses[r] = float32(math.NaN())
			continue
		}
		losses[r] = softLossRow(t.q, s)
	}
	return meanOfRowLosses(losses)
}

// distillationRowGradient writes a row's gradient. When every teacher
// probability is above 0 and every centred difference is at most 1 it is
// Barron's (arXiv 2610.31642, Eq. 9):
//
//	delta[c] = zS[c] - zS[0] - zT[c] + zT[0], exactly, rounded once
//	mu = sum(p*delta), d[c] = delta[c] - mu
//	a = sum(p*expm1(d)), sigma = log1p(a), s[c] = d[c] - sigma
//	u[c] = p[c]*expm1(s[c]), w[c] = u[c] - q[c]*sum(u)
//
// otherwise it is q - p through softGradientRow with mass 1.
func distillationRowGradient(out, zS, zT []float32, s, t roundedSoftmaxRow, scale float32) {
	p, q := t.q, s.q
	n := len(p)
	for c := range p {
		if !(p[c] > 0) {
			softGradientRow(out, p, s, 1, scale)
			return
		}
	}
	delta := make([]float32, n)
	for c := range delta {
		delta[c] = roundedTerms([]float32{zS[c], zS[0], zT[c], zT[0]}, []float32{1, -1, -1, 1})
	}
	mu := roundedTerms(p, delta)
	d := make([]float32, n)
	for c := range d {
		d[c] = sum2Rounded(delta[c], 1, -1, mu)
		if !(float32(math.Abs(float64(d[c]))) <= 1) {
			softGradientRow(out, p, s, 1, scale)
			return
		}
	}
	expm1d := make([]float32, n)
	for c := range d {
		expm1d[c] = expm1Float32(d[c])
	}
	sigma := log1pFloat32(roundedTerms(p, expm1d))
	u := make([]float32, n)
	for c := range u {
		u[c] = mulRounded(p[c], expm1Float32(sum2Rounded(d[c], 1, -1, sigma)))
	}
	total := roundedSumOf(u)
	for c := range out {
		out[c] = mulRounded(sum2Rounded(u[c], 1, -q[c], total), scale)
	}
}

func distillationRoundedVJP(student, teacher, upstream *Tensor) (*Tensor, error) {
	if err := requireScalarFloat32(upstream, "distillation upstream"); err != nil {
		return nil, err
	}
	rows, classes := student.shape[0], student.shape[1]
	gradient, err := newZeroFloat32Tensor(student.shape)
	if err != nil {
		return nil, err
	}
	scale := upstream.data[0] / float32(rows)
	for r := 0; r < rows; r++ {
		out := gradient.data[r*classes : (r+1)*classes]
		zS, zT := student.data[r*classes:(r+1)*classes], teacher.data[r*classes:(r+1)*classes]
		s, t := roundedSoftmaxRowOf(zS), roundedSoftmaxRowOf(zT)
		if !s.finite || !t.finite {
			for c := range out {
				out[c] = float32(math.NaN())
			}
			continue
		}
		distillationRowGradient(out, zS, zT, s, t, scale)
	}
	return gradient, nil
}
