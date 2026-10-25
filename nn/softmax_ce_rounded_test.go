package nn

import (
	"math"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

// ceReference computes the specified sequence with every step done in
// extended precision and rounded once, using the independent exp and log
// oracles of exp_log_rounded_test.go.
func ceReference(x []float32, label int) (e []float32, s, loss float32) {
	const prec = 2000
	bf := func(v float32) *big.Float { return new(big.Float).SetPrec(prec).SetFloat64(float64(v)) }
	round := func(v *big.Float) float32 {
		if v.Sign() == 0 {
			return 0
		}
		f, _ := v.Float32()
		return f
	}
	maxValue := x[0]
	for _, v := range x[1:] {
		if v > maxValue {
			maxValue = v
		}
	}
	d := make([]float32, len(x))
	e = make([]float32, len(x))
	total := new(big.Float).SetPrec(prec)
	for c, v := range x {
		d[c] = round(new(big.Float).SetPrec(prec).Sub(bf(v), bf(maxValue)))
		e[c] = expLogOracleExp(d[c])
		total.Add(total, bf(e[c]))
	}
	s = round(total)
	l := expLogOracleLog(s)
	loss = round(new(big.Float).SetPrec(prec).Sub(bf(l), bf(d[label])))
	return e, s, loss
}

func TestRoundedSoftmaxCrossEntropyHostMatchesTheReference(t *testing.T) {
	rng := rand.New(rand.NewSource(12))
	rows, classes := 9, 7
	logits := randomTensor(t, rng, rows, classes)
	labels := make([]int64, rows)
	for i := range labels {
		labels[i] = int64(rng.Intn(classes))
	}
	labelTensor, _ := NewInt64Tensor([]int{rows}, labels)
	loss, err := softmaxCrossEntropyRoundedForward(logits, labelTensor)
	if err != nil {
		t.Fatal(err)
	}
	rowLosses := make([]float32, rows)
	for r := 0; r < rows; r++ {
		_, _, rowLosses[r] = ceReference(logits.data[r*classes:(r+1)*classes], int(labels[r]))
	}
	sum := new(big.Float).SetPrec(2000)
	for _, v := range rowLosses {
		sum.Add(sum, new(big.Float).SetPrec(2000).SetFloat64(float64(v)))
	}
	total, _ := sum.Float32()
	want := total / float32(rows)
	if math.Float32bits(loss.data[0]) != math.Float32bits(want) {
		t.Fatalf("loss = %g (%#x), want %g (%#x)", loss.data[0], math.Float32bits(loss.data[0]), want, math.Float32bits(want))
	}
}

func TestRoundedSoftmaxCrossEntropyGradientIsTheSpecifiedSequence(t *testing.T) {
	rng := rand.New(rand.NewSource(13))
	rows, classes := 5, 6
	logits := randomTensor(t, rng, rows, classes)
	labels := make([]int64, rows)
	for i := range labels {
		labels[i] = int64(rng.Intn(classes))
	}
	labelTensor, _ := NewInt64Tensor([]int{rows}, labels)
	upstream, _ := NewTensor(nil, []float32{-0.75})
	gradient, err := softmaxCrossEntropyRoundedVJP(logits, labelTensor, upstream)
	if err != nil {
		t.Fatal(err)
	}
	scale := float32(-0.75) / float32(rows)
	for r := 0; r < rows; r++ {
		e, s, _ := ceReference(logits.data[r*classes:(r+1)*classes], int(labels[r]))
		for c := 0; c < classes; c++ {
			g := mulRounded(e[c]/s, scale)
			if c == int(labels[r]) {
				var others []float32
				others = append(others, e[:c]...)
				others = append(others, e[c+1:]...)
				g = mulRounded(-(roundedSumOf(others) / s), scale)
			}
			if math.Float32bits(gradient.data[r*classes+c]) != math.Float32bits(g) {
				t.Fatalf("grad[%d,%d] = %g, want %g", r, c, gradient.data[r*classes+c], g)
			}
		}
	}
}

func TestRoundedSoftmaxCrossEntropyNonFiniteRowIsNaN(t *testing.T) {
	logits, _ := NewTensor([]int{2, 3}, []float32{1, float32(math.Inf(1)), 3, 1, 2, 3})
	labels, _ := NewInt64Tensor([]int{2}, []int64{0, 1})
	loss, err := softmaxCrossEntropyRoundedForward(logits, labels)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(float64(loss.data[0])) {
		t.Fatalf("loss = %g, want NaN", loss.data[0])
	}
}

func runClassifierSteps(t *testing.T, device accel.Device) ([]float32, []float32) {
	t.Helper()
	rng := rand.New(rand.NewSource(33))
	rows, in, hidden, classes := 12, 9, 8, 5
	X, W1, B1, W2 := randomTensor(t, rng, rows, in), randomTensor(t, rng, in, hidden), randomTensor(t, rng, hidden), randomTensor(t, rng, hidden, classes)
	labels := make([]int64, rows)
	for i := range labels {
		labels[i] = int64(rng.Intn(classes))
	}
	labelTensor, _ := NewInt64Tensor([]int{rows}, labels)
	mv := func(x *Tensor) *Tensor {
		moved, err := x.To(device)
		if err != nil {
			t.Fatal(err)
		}
		return moved
	}
	X, W1, B1, W2, labelTensor = mv(X), mv(W1), mv(B1), mv(W2), mv(labelTensor)
	defer X.Release()
	defer labelTensor.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*Tensor{W1, B1, W2} {
		if _, err := tape.Param(p); err != nil {
			t.Fatal(err)
		}
	}
	var losses []float32
	for step := 0; step < 4; step++ {
		tape.Clear()
		h, err := tape.MatMul(X, W1)
		if err != nil {
			t.Fatal(err)
		}
		h, err = tape.Add(h, B1)
		if err != nil {
			t.Fatal(err)
		}
		y, err := tape.MatMul(h, W2)
		if err != nil {
			t.Fatal(err)
		}
		loss, err := tape.SoftmaxCrossEntropy(y, labelTensor)
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.Backward(loss); err != nil {
			t.Fatal(err)
		}
		if err := tape.AdamW(0.01, 0.05); err != nil {
			t.Fatal(err)
		}
		hl, err := loss.ToHost()
		if err != nil {
			t.Fatal(err)
		}
		losses = append(losses, hl.data[0])
	}
	w, err := W2.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	return losses, w.data
}

func TestClassifierStepOnTheDeviceEqualsTheHost(t *testing.T) {
	device := residentTestDevice(t)
	hostLoss, hostW := runClassifierSteps(t, accel.Device{})
	devLoss, devW := runClassifierSteps(t, device)
	sameBitsSlice(t, "loss", devLoss, hostLoss)
	sameBitsSlice(t, "W2", devW, hostW)
}

func TestInt64LabelsMoveToTheDeviceOrAreRefused(t *testing.T) {
	device := residentTestDevice(t)
	labels, _ := NewInt64Tensor([]int{3}, []int64{2, 0, 1})
	moved, err := labels.To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer moved.Release()
	back, err := moved.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	if got := back.int64Data; got[0] != 2 || got[1] != 0 || got[2] != 1 {
		t.Fatalf("round trip = %v", got)
	}
	if _, err := moved.Int64Data(); err == nil || !strings.Contains(err.Error(), "ToHost") {
		t.Fatalf("Int64Data error = %v", err)
	}
	huge, _ := NewInt64Tensor([]int{1}, []int64{1 << 40})
	if _, err := huge.To(device); err == nil || !strings.Contains(err.Error(), "1099511627776") {
		t.Fatalf("a label past int32 must be refused naming it, got %v", err)
	}
	negative, _ := NewInt64Tensor([]int{1}, []int64{-1})
	if _, err := negative.To(device); err == nil {
		t.Fatal("a negative label must be refused")
	}
	// a label past the class count is refused before any kernel runs
	logits, _ := NewTensor([]int{3, 2}, make([]float32, 6))
	rl, _ := logits.To(device)
	defer rl.Release()
	tape := NewTape()
	defer tape.Close()
	_ = tape.SetPrecision(accel.PrecisionFloat32Rounded)
	if _, err := tape.SoftmaxCrossEntropy(rl, moved); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("label 2 with 2 classes must be refused, got %v", err)
	}
}
