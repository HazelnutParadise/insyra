package nn

import (
	"fmt"
	"math"
)

// SoftTargetCrossEntropy is the mean over rows of -sum_k p_k*log(softmax(z)_k)
// for logits [N,C] and a probability row per sample in targets [N,C]. Rows of
// targets must be finite, non-negative and sum to 1 within 1e-5; the gradient
// of the logits is (q*sum(p) - p)/N, and targets receive none.
func (t *Tape) SoftTargetCrossEntropy(logits, targets *Tensor) (*Tensor, error) {
	if err := t.softCEOperands("soft-target cross-entropy", logits, targets); err != nil {
		return nil, err
	}
	rows, classes := logits.shape[0], logits.shape[1]
	for r := 0; r < rows; r++ {
		if err := checkProbabilityRow(targets.data[r*classes:(r+1)*classes], r); err != nil {
			return nil, err
		}
	}
	return t.softCE("SoftTargetCrossEntropy", false, logits, targets, func(r int, student, target []float32, p []float64) {
		for c, v := range target {
			p[c] = float64(v)
		}
	})
}

// DistillationLoss is the mean over rows of -sum_k p_k*log(q_k) with
// p = softmax(teacher) and q = softmax(student), both [N,C]. The teacher is a
// constant. Give logits already divided by the temperature; the gradient is
// then the temperature-scaled one, to be multiplied by T^2 by the caller if
// wanted. Equal logits give a gradient of exactly 0.
func (t *Tape) DistillationLoss(student, teacher *Tensor) (*Tensor, error) {
	if err := t.softCEOperands("distillation loss", student, teacher); err != nil {
		return nil, err
	}
	return t.softCE("DistillationLoss", true, student, teacher, func(r int, student, teacher []float32, p []float64) {
		_ = softmaxRow64(teacher, p, nil)
	})
}

func (t *Tape) softCEOperands(op string, logits, other *Tensor) error {
	resident, err := t.residency(op, logits, other)
	if err != nil {
		return err
	}
	if resident {
		return fmt.Errorf("autodiff %s has no device kernel yet; call ToHost", op)
	}
	if err := requireFloat32(logits, op+" logits"); err != nil {
		return err
	}
	if err := requireFloat32(other, op+" second operand"); err != nil {
		return err
	}
	if len(logits.shape) != 2 || !sameShape(logits.shape, other.shape) {
		return fmt.Errorf("%s requires two [N,C] tensors of the same shape, got %v and %v", op, logits.shape, other.shape)
	}
	if logits.shape[0] == 0 || logits.shape[1] == 0 {
		return fmt.Errorf("%s requires non-empty logits", op)
	}
	return nil
}

func checkProbabilityRow(row []float32, index int) error {
	var sum float64
	for c, v := range row {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return fmt.Errorf("soft-target cross-entropy targets row %d, class %d is not finite", index, c)
		}
		if v < 0 {
			return fmt.Errorf("soft-target cross-entropy targets row %d, class %d is negative (%g)", index, c, v)
		}
		sum += float64(v)
	}
	if math.Abs(sum-1) > 1e-5 {
		return fmt.Errorf("soft-target cross-entropy targets row %d sums to %g, want 1", index, sum)
	}
	return nil
}

// softmaxRow64 writes softmax(x) into q and, when logSoft is not nil, the log
// of it into logSoft, all in float64, and returns the index of the largest
// logit. The log-sum-exp is max + log1p(sum of the other exponentials), so the
// largest class's log-probability keeps the small terms that adding them to 1
// first would lose (Barron, arXiv 2610.31642). A row with a non-finite value
// is NaN.
func softmaxRow64(x []float32, q, logSoft []float64) int {
	peak := 0
	for c, v := range x {
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			for c := range q {
				q[c] = math.NaN()
				if logSoft != nil {
					logSoft[c] = math.NaN()
				}
			}
			return peak
		}
		if v > x[peak] {
			peak = c
		}
	}
	maxValue := float64(x[peak])
	var others float64
	for c, v := range x {
		if c != peak {
			others += math.Exp(float64(v) - maxValue)
		}
	}
	logOthers := math.Log1p(others)
	for c, v := range x {
		// Shifted by the maximum first, so a large common offset in the
		// logits cannot round the normalisation away.
		l := (float64(v) - maxValue) - logOthers
		if c == peak {
			l = -logOthers
		}
		if logSoft != nil {
			logSoft[c] = l
		}
		q[c] = math.Exp(l)
	}
	return peak
}

// softCE runs both losses: target fills p, the target probabilities of a row,
// from the second operand. The loss skips p_k = 0, so 0*log 0 is 0, and the
// gradient sums to zero in float64 before it is rounded: q*sum(p) - p for a
// target row, and for a teacher's softmax q - p with q*sum(q - p) taken off,
// which is exactly 0 when the logits are equal.
func (t *Tape) softCE(name string, distill bool, logits, second *Tensor, target func(row int, logits, second []float32, p []float64)) (*Tensor, error) {
	rows, classes := logits.shape[0], logits.shape[1]
	p := make([]float64, classes)
	q := make([]float64, classes)
	logSoft := make([]float64, classes)
	var total float64
	for r := 0; r < rows; r++ {
		z := logits.data[r*classes : (r+1)*classes]
		target(r, z, second.data[r*classes:(r+1)*classes], p)
		softmaxRow64(z, q, logSoft)
		for c := range p {
			if p[c] != 0 {
				total -= p[c] * logSoft[c]
			}
		}
	}
	loss, err := newFloat32Tensor(nil, []float32{float32(total / float64(rows))})
	if err != nil {
		return nil, err
	}
	t.record(name, []*Tensor{logits, second}, loss, func(upstream *Tensor) ([]*Tensor, error) {
		if err := requireScalarFloat32(upstream, name+" upstream"); err != nil {
			return nil, err
		}
		gradient, err := newZeroFloat32Tensor(logits.shape)
		if err != nil {
			return nil, err
		}
		scale := float64(upstream.data[0]) / float64(rows)
		for r := 0; r < rows; r++ {
			z := logits.data[r*classes : (r+1)*classes]
			target(r, z, second.data[r*classes:(r+1)*classes], p)
			peak := softmaxRow64(z, q, nil)
			g := gradient.data[r*classes : (r+1)*classes]
			if distill && distillNearEqualGradient(z, second.data[r*classes:(r+1)*classes], p, q, g, scale) {
				continue
			}
			var mass float64 = 1
			if !distill {
				mass = 0
				for _, v := range p {
					mass += v
				}
			}
			// The largest class's gradient is minus the sum of the others':
			// q*mass - p there is a difference of two numbers near 1 that
			// rounds to 0 once the margin passes the precision cliff, while
			// the others' gradients do not.
			var rest float64
			for c := range p {
				if c != peak {
					d := (q[c]*mass - p[c]) * scale
					g[c] = float32(d)
					rest += d
				}
			}
			g[peak] = float32(0 - rest)
		}
		return []*Tensor{gradient, nil}, nil
	})
	return loss, nil
}

// distillNearEqualGradient writes the gradient of a distillation row from the
// student-teacher differences when they are small, where q - p subtracts two
// probabilities that are almost equal and loses a difference that float32
// could hold (Barron, arXiv 2610.31642, Eq. 9). With delta = z_S - z_T,
// d = delta - sum(p*delta), s = d - log1p(sum(p*expm1(d))) = log(q/p),
// u = p*expm1(s) = q - p, the gradient is u - q*sum(u). p holds softmax of the
// teacher and q softmax of the student. It reports false, leaving g alone,
// when some p underflows to 0 or d is larger than 1, where q - p is
// accurate and expm1 would not be.
func distillNearEqualGradient(student, teacher []float32, p, q []float64, g []float32, scale float64) bool {
	// Each row is shifted by its own maximum first, so a common offset in
	// either row cannot erase the class differences.
	studentMax, teacherMax := float64(student[0]), float64(teacher[0])
	for c := range p {
		studentMax = math.Max(studentMax, float64(student[c]))
		teacherMax = math.Max(teacherMax, float64(teacher[c]))
	}
	delta := make([]float64, len(p))
	var mean float64
	for c := range p {
		if !(p[c] > 0) {
			return false
		}
		delta[c] = (float64(student[c]) - studentMax) - (float64(teacher[c]) - teacherMax)
		mean += p[c] * delta[c]
	}
	d := make([]float64, len(p))
	for c := range p {
		d[c] = delta[c] - mean
		if !(math.Abs(d[c]) <= 1) {
			return false
		}
	}
	var drift float64
	for c := range p {
		drift += p[c] * math.Expm1(d[c])
	}
	shift := math.Log1p(drift)
	u := make([]float64, len(p))
	var total float64
	for c := range p {
		u[c] = p[c] * math.Expm1(d[c]-shift)
		total += u[c]
	}
	for c := range p {
		g[c] = float32((u[c] - q[c]*total) * scale)
	}
	return true
}
