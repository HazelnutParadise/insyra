package nn

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
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
		for name, vjp := range map[string]func(a, b, c *Tensor) (*Tensor, error){
			"default": softmaxCrossEntropyVJP,
			"rounded": softmaxCrossEntropyRoundedVJP,
		} {
			g, err := vjp(logits, labels, upstream)
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

func largeMarginCEGradient(t *testing.T, device accel.Device) []float32 {
	t.Helper()
	data := []float32{
		8, 0, -3, 0, 20, 1, 0, 40, 0, 0, 0, 60, 3, -50, 2, -30, 5, 90,
	}
	logits, _ := NewTensor([]int{6, 3}, data)
	labels, _ := NewInt64Tensor([]int{6}, []int64{0, 1, 1, 2, 0, 0})
	var err error
	if logits, err = logits.To(device); err != nil {
		t.Fatal(err)
	}
	defer logits.Release()
	if labels, err = labels.To(device); err != nil {
		t.Fatal(err)
	}
	defer labels.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	if _, err := tape.Param(logits); err != nil {
		t.Fatal(err)
	}
	loss, err := tape.SoftmaxCrossEntropy(logits, labels)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	grad, err := tape.Grad(logits)
	if err != nil {
		t.Fatal(err)
	}
	host, err := grad.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	return host.data
}

// The device and the host return the same bits past the precision cliff too,
// and each row sums to zero within rounding.
func TestRoundedCrossEntropyGradientPastTheCliffEqualsTheHostOnTheDevice(t *testing.T) {
	device := residentTestDevice(t)
	host := largeMarginCEGradient(t, accel.Device{})
	sameBitsSlice(t, "gradient", largeMarginCEGradient(t, device), host)
	for row := 0; row < 6; row++ {
		var sum, biggest float64
		for _, v := range host[row*3 : row*3+3] {
			sum += float64(v)
			biggest = math.Max(biggest, math.Abs(float64(v)))
		}
		if math.Abs(sum) > 1e-6*biggest {
			t.Errorf("row %d sums to %g", row, sum)
		}
	}
}

// A softmax whose largest probability rounds to 1 still has a gradient for
// that probability: with logits [m, 0] and upstream [1, 0], dy0/dz0 is
// y0*(1-y0) = e^-m/(1+e^-m)^2.
func TestSoftmaxGradientSurvivesALargeMargin(t *testing.T) {
	for _, margin := range []float32{10, 20, 30, 40} {
		e := math.Exp(-float64(margin))
		y, _ := NewTensor([]int{1, 2}, []float32{float32(1 / (1 + e)), float32(e / (1 + e))})
		upstream, _ := NewTensor([]int{1, 2}, []float32{1, 0})
		got := softmaxVJP(y, upstream, 1)
		want := e / ((1 + e) * (1 + e))
		if g := float64(got.data[0]); g == 0 || math.Abs(g-want) > 1e-6*want {
			t.Errorf("margin %g: dy0/dz0 = %g, want %g", margin, g, want)
		}
		if sum := float64(got.data[0]) + float64(got.data[1]); math.Abs(sum) > 1e-6*want {
			t.Errorf("margin %g: gradient sums to %g, want 0", margin, sum)
		}
	}
}

// A small upstream component beside a large one keeps its gradient: the pivot
// form is for the largest probability only.
func TestSoftmaxGradientKeepsASmallUpstreamComponent(t *testing.T) {
	y, _ := NewTensor([]int{1, 3}, []float32{0.5, 0.25, 0.25})
	upstream, _ := NewTensor([]int{1, 3}, []float32{1, -2, 1e-20})
	got := softmaxVJP(y, upstream, 1)
	// dot = 0.5 - 0.5 + 0.25e-20; g2 = 0.25*(1e-20 - dot).
	want := 0.25 * (1e-20 - 0.25e-20)
	if g := float64(got.data[2]); g == 0 || math.Abs(g-want) > 1e-5*want {
		t.Errorf("g[2] = %g, want %g", g, want)
	}
}
