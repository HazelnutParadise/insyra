package nn

import (
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// #265 (NN-1): LayerNorm took interface{}, and anything that was not an int or
// a []int became a layer normalizing a dimension of size 0.
func TestLayerNormTakesATypedSize(t *testing.T) {
	if got := reflect.TypeOf(LayerNorm).In(0).Kind(); got != reflect.Int {
		t.Fatalf("LayerNorm takes a %v, want int", got)
	}

	single, err := NewSequential(NewTape(3), LayerNorm(4))
	if err != nil {
		t.Fatal(err)
	}
	if got := single.NamedParameters()["0.weight"].Value().Shape(); !slices.Equal(got, []int{4}) {
		t.Errorf("LayerNorm(4) weight shape = %v, want [4]", got)
	}

	dims := []int{2, 3}
	layer := LayerNormShape(dims)
	dims[0] = 7 // the layer keeps its own copy
	shaped, err := NewSequential(NewTape(3), layer)
	if err != nil {
		t.Fatal(err)
	}
	named := shaped.NamedParameters()
	for _, name := range []string{"0.weight", "0.bias"} {
		if got := named[name].Value().Shape(); !slices.Equal(got, []int{2, 3}) {
			t.Errorf("LayerNormShape([2 3]) %s shape = %v, want [2 3]", name, got)
		}
	}
	input := mustTestTensor(t, []int{2, 2, 3}, []float32{1, 2, 3, 4, 5, 6, -1, 0, 2, 8, -3, 5})
	got, err := shaped.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	want, err := LayerNormalization(input, named["0.weight"].Value(), named["0.bias"].Value(), 1, 1e-5)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Shape(), want.Shape()) || len(got.data) != len(want.data) {
		t.Fatalf("LayerNormShape output shape %v, want %v", got.Shape(), want.Shape())
	}
	for i := range got.data {
		if math.Float32bits(got.data[i]) != math.Float32bits(want.data[i]) {
			t.Fatalf("LayerNormShape output[%d] = %v, want %v", i, got.data[i], want.data[i])
		}
	}

	refused := []struct {
		name  string
		layer Layer
	}{
		{"LayerNorm(0)", LayerNorm(0)},
		{"LayerNorm(-2)", LayerNorm(-2)},
		{"LayerNormShape(nil)", LayerNormShape(nil)},
		{"LayerNormShape(empty)", LayerNormShape([]int{})},
		{"LayerNormShape with a zero", LayerNormShape([]int{2, 0})},
	}
	for _, tc := range refused {
		if _, err := NewSequential(NewTape(1), tc.layer); err == nil || !strings.Contains(err.Error(), "layer 0 (LayerNorm)") {
			t.Errorf("%s: NewSequential error = %v, want one naming layer 0 (LayerNorm)", tc.name, err)
		}
	}
}

func TestDeprecatedNewLayerNormKeepsItsMeaning(t *testing.T) {
	weightShape := func(layer Layer) ([]int, error) {
		model, err := NewSequential(NewTape(1), layer)
		if err != nil {
			return nil, err
		}
		return model.NamedParameters()["0.weight"].Value().Shape(), nil
	}
	if got, err := weightShape(NewLayerNorm(4)); err != nil || !slices.Equal(got, []int{4}) {
		t.Errorf("NewLayerNorm(4) = %v, %v; want [4]", got, err)
	}
	if got, err := weightShape(NewLayerNorm([]int{2, 3})); err != nil || !slices.Equal(got, []int{2, 3}) {
		t.Errorf("NewLayerNorm([2 3]) = %v, %v; want [2 3]", got, err)
	}
	if _, err := weightShape(NewLayerNorm(int64(4))); err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Errorf("NewLayerNorm(int64(4)) error = %v, want the dimension error it gave before", err)
	}
}
