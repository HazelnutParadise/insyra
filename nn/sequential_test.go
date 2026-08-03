package nn

import (
	"strings"
	"testing"
)

func TestSequentialStructuralSurface(t *testing.T) {
	tape := NewTape(20260803)
	model, err := NewSequential(tape, Dense(3, 4), ReLU(), Dropout(0.5), Dense(4, 2))
	if err != nil {
		t.Fatal(err)
	}
	named := model.NamedParameters()
	for _, name := range []string{"0.weight", "0.bias", "3.weight", "3.bias"} {
		if named[name] == nil {
			t.Fatalf("NamedParameters missing %q: %v", name, named)
		}
	}
	if len(named) != 4 {
		t.Fatalf("NamedParameters returned %d entries, want 4", len(named))
	}

	input := mustTestTensor(t, []int{2, 3}, []float32{1, -2, 0.5, -0.25, 0.75, 2})
	withoutDropout, err := NewSequential(NewTape(20260803), Dense(3, 4), ReLU(), Dense(4, 2))
	if err != nil {
		t.Fatal(err)
	}
	for index, parameter := range withoutDropout.Parameters() {
		parameter.value.data = append([]float32(nil), model.Parameters()[index].Value().data...)
	}
	got, err := model.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	want, err := withoutDropout.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	if !sameFloat32(got.Data(), want.Data()) {
		t.Fatalf("Predict with Dropout changed output: got %v, want %v", got.Data(), want.Data())
	}
}

func TestSequentialDimensionMismatchNamesLayer(t *testing.T) {
	_, err := NewSequential(NewTape(), Dense(3, 4), NewSigmoid(), Dense(5, 2))
	if err == nil {
		t.Fatal("dimension mismatch was accepted")
	}
	if !strings.Contains(err.Error(), "layer 2") || !strings.Contains(err.Error(), "Dense") {
		t.Fatalf("dimension error %q does not name layer index and kind", err)
	}
}

func TestSequentialFuncResidual(t *testing.T) {
	model, err := NewSequential(NewTape(), Func(func(tape *Tape, x *Tensor) (*Tensor, error) {
		return tape.Add(x, x)
	}))
	if err != nil {
		t.Fatal(err)
	}
	input := mustTestTensor(t, []int{2, 2}, []float32{1, -2, 3, 4})
	output, err := model.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	if !sameFloat32(output.Data(), []float32{2, -4, 6, 8}) {
		t.Fatalf("residual output = %v", output.Data())
	}
}

func TestSequentialLoadWeightsRejectsNamesAndShapes(t *testing.T) {
	model, err := NewSequential(NewTape(), Dense(2, 3))
	if err != nil {
		t.Fatal(err)
	}
	weight := mustTestTensor(t, []int{3, 2}, make([]float32, 6))
	bias := mustTestTensor(t, []int{3}, make([]float32, 3))
	if err := model.LoadWeights(map[string]*Tensor{"wrong": weight, "0.bias": bias}); err == nil || !strings.Contains(err.Error(), "0.weight") {
		t.Fatalf("bad names error = %v", err)
	}
	badWeight := mustTestTensor(t, []int{2, 3}, make([]float32, 6))
	if err := model.LoadWeights(map[string]*Tensor{"0.weight": badWeight, "0.bias": bias}); err == nil || !strings.Contains(err.Error(), "weight shape") {
		t.Fatalf("bad shape error = %v", err)
	}
}

func sameFloat32(left, right []float32) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
