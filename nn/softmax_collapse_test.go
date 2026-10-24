package nn

import (
	"math"
	"testing"
)

// The label's gradient is -e^-m/(1+e^-m) for a margin m. Written as p-1 it
// is exactly 0 in float32 once e^-m falls below 2^-24 (m about 16.6), while
// the other classes' gradients stay nonzero, so the row stops summing to zero
// (Barron, "A Fix for Softmax and Cross-Entropy", arXiv 2610.31642).
func TestSoftmaxCrossEntropyGradientSurvivesALargeMargin(t *testing.T) {
	for _, margin := range []float32{10, 20, 30, 40, 60} {
		logits, _ := NewTensor([]int{1, 2}, []float32{margin, 0})
		labels, _ := NewInt64Tensor([]int{1}, []int64{0})
		upstream, _ := NewTensor(nil, []float32{1})
		wantLabel := -math.Exp(-float64(margin)) / (1 + math.Exp(-float64(margin)))
		loss, err := softmaxCrossEntropyForward(logits, labels)
		if err != nil {
			t.Fatal(err)
		}
		wantLoss := math.Log1p(math.Exp(-float64(margin)))
		if got := float64(loss.data[0]); math.Abs(got-wantLoss) > 1e-6*wantLoss {
			t.Errorf("margin %g: loss = %g, want %g", margin, got, wantLoss)
		}
		{
			name := "default"
			g, err := softmaxCrossEntropyVJP(logits, labels, upstream)
			if err != nil {
				t.Fatal(err)
			}
			got := float64(g.data[0])
			if got == 0 || math.Abs(got-wantLabel) > 1e-6*math.Abs(wantLabel) {
				t.Errorf("%s margin %g: label gradient = %g, want %g", name, margin, got, wantLabel)
			}
			sum := float64(g.data[0]) + float64(g.data[1])
			if math.Abs(sum) > 1e-6*math.Abs(wantLabel) {
				t.Errorf("%s margin %g: row gradient sums to %g, want 0", name, margin, sum)
			}
		}
	}
}
