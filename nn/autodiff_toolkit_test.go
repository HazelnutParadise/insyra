package nn

import (
	"math"
	"testing"
)

func TestMSELossFiniteDifference(t *testing.T) {
	values := []float32{0.2, -1.1, 0.7, 1.4, -0.6}
	target := []float32{-0.3, 0.4, 0.1, 1.8, -1.2}
	gradient, _ := toolkitLossGradient(t, values, target, func(tape *Tape, input, target *Tensor) (*Tensor, error) {
		return tape.MSELoss(input, target)
	})

	const h = float32(1e-3)
	for index, want := range gradient {
		plus := append([]float32(nil), values...)
		minus := append([]float32(nil), values...)
		plus[index] += h
		minus[index] -= h
		plusLoss := toolkitLoss(t, plus, target, func(tape *Tape, input, target *Tensor) (*Tensor, error) {
			return tape.MSELoss(input, target)
		})
		minusLoss := toolkitLoss(t, minus, target, func(tape *Tape, input, target *Tensor) (*Tensor, error) {
			return tape.MSELoss(input, target)
		})
		finite := (plusLoss - minusLoss) / (2 * h)
		if math.Abs(float64(want-finite)) > 3e-3 {
			t.Fatalf("MSE gradient[%d] = %g, finite difference = %g", index, want, finite)
		}
	}
}

func TestBCEWithLogitsLossFiniteDifference(t *testing.T) {
	values := []float32{-4.2, -0.8, 0.3, 2.1, 7.5}
	target := []float32{0, 1, 0.25, 1, 0}
	gradient, _ := toolkitLossGradient(t, values, target, func(tape *Tape, input, target *Tensor) (*Tensor, error) {
		return tape.BCEWithLogitsLoss(input, target)
	})

	const h = float32(1e-3)
	for index, want := range gradient {
		plus := append([]float32(nil), values...)
		minus := append([]float32(nil), values...)
		plus[index] += h
		minus[index] -= h
		plusLoss := toolkitLoss(t, plus, target, func(tape *Tape, input, target *Tensor) (*Tensor, error) {
			return tape.BCEWithLogitsLoss(input, target)
		})
		minusLoss := toolkitLoss(t, minus, target, func(tape *Tape, input, target *Tensor) (*Tensor, error) {
			return tape.BCEWithLogitsLoss(input, target)
		})
		finite := (plusLoss - minusLoss) / (2 * h)
		if math.Abs(float64(want-finite)) > 3e-3 {
			t.Fatalf("BCE gradient[%d] = %g, finite difference = %g", index, want, finite)
		}
	}
}

func TestSGDMomentumHandComputed(t *testing.T) {
	parameter := mustTestTensor(t, []int{2}, []float32{1, -2})
	tape := NewTape()
	tracked, err := tape.Param(parameter)
	if err != nil {
		t.Fatal(err)
	}

	tape.grads[tracked.Value()] = mustTestTensor(t, []int{2}, []float32{2, -1})
	if err := tape.SGDMomentum(0.1, 0.9); err != nil {
		t.Fatal(err)
	}
	assertAutodiffValue(t, "momentum first step", parameter.data, []float32{0.8, -1.9})

	tape.grads[tracked.Value()] = mustTestTensor(t, []int{2}, []float32{-1, 3})
	if err := tape.SGDMomentum(0.1, 0.9); err != nil {
		t.Fatal(err)
	}
	assertAutodiffValue(t, "momentum second step", parameter.data, []float32{0.72, -2.11})
}

func TestCosineAnnealingLR(t *testing.T) {
	schedule, err := NewCosineAnnealingLR(2, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []float32{2, 1.7071068, 1, 0.2928932, 0}
	for step, expected := range want {
		if got := schedule.LR(step); math.Abs(float64(got-expected)) > 1e-6 {
			t.Fatalf("step %d lr = %g, want %g", step, got, expected)
		}
	}
}

func TestClipGradNorm(t *testing.T) {
	parameterA := mustTestTensor(t, []int{2}, []float32{1, 2})
	parameterB := mustTestTensor(t, []int{2}, []float32{3, 4})
	tape := NewTape()
	trackedA, err := tape.Param(parameterA)
	if err != nil {
		t.Fatal(err)
	}
	trackedB, err := tape.Param(parameterB)
	if err != nil {
		t.Fatal(err)
	}
	tape.grads[trackedA.Value()] = mustTestTensor(t, []int{2}, []float32{3, 4})
	tape.grads[trackedB.Value()] = mustTestTensor(t, []int{2}, []float32{0, 12})

	norm, err := tape.ClipGradNorm(6.5)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(float64(norm-13)) > 1e-6 {
		t.Fatalf("pre-clip norm = %g, want 13", norm)
	}
	assertAutodiffValue(t, "clipped gradient A", tape.grads[parameterA].data, []float32{1.5, 2})
	assertAutodiffValue(t, "clipped gradient B", tape.grads[parameterB].data, []float32{0, 6})

	below := NewTape()
	parameter, err := below.Param(mustTestTensor(t, []int{2}, []float32{0, 0}))
	if err != nil {
		t.Fatal(err)
	}
	below.grads[parameter.Value()] = mustTestTensor(t, []int{2}, []float32{3, 4})
	norm, err = below.ClipGradNorm(5)
	if err != nil {
		t.Fatal(err)
	}
	if norm != 5 {
		t.Fatalf("below-threshold norm = %g, want 5", norm)
	}
	assertAutodiffValue(t, "unclipped gradient", below.grads[parameter.Value()].data, []float32{3, 4})
}

type toolkitLossBuilder func(*Tape, *Tensor, *Tensor) (*Tensor, error)

func toolkitLossGradient(t *testing.T, values, target []float32, build toolkitLossBuilder) ([]float32, float32) {
	t.Helper()
	input := mustTestTensor(t, []int{len(values)}, values)
	targetTensor := mustTestTensor(t, []int{len(target)}, target)
	tape := NewTape()
	parameter, err := tape.Param(input)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := build(tape, parameter.Value(), targetTensor)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	gradient, err := tape.Grad(parameter.Value())
	if err != nil {
		t.Fatal(err)
	}
	return gradient.data, loss.data[0]
}

func toolkitLoss(t *testing.T, values, target []float32, build toolkitLossBuilder) float32 {
	t.Helper()
	input := mustTestTensor(t, []int{len(values)}, values)
	targetTensor := mustTestTensor(t, []int{len(target)}, target)
	loss, err := build(NewTape(), input, targetTensor)
	if err != nil {
		t.Fatal(err)
	}
	return loss.data[0]
}
