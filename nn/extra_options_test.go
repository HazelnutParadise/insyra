package nn

import "testing"

// A trailing optional value stands for one value. A second seed or a second
// options struct used to be dropped (NewTape) or to throw every option away in
// favour of the defaults (the layer constructors). Both are errors now,
// reported where each already reports its failures.
func TestMoreThanOneOptionalValueIsRefused(t *testing.T) {
	tape := NewTape(1, 2)
	value, err := NewTensor([]int{1}, []float32{1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tape.Param(value); err == nil {
		t.Error("a tape given two seeds accepted a parameter")
	}
	loss, err := NewTensor(nil, []float32{1})
	if err != nil {
		t.Fatal(err)
	}
	if err := NewTape().Backward(loss); err != nil {
		t.Fatalf("a scalar loss does not run backward on a plain tape, so the check below proves nothing: %v", err)
	}
	if err := tape.Backward(loss); err == nil {
		t.Error("a tape given two seeds ran backward")
	}

	layers := map[string]Layer{
		"Conv2D":    Conv2D(1, 1, 3, ConvOptions{}, ConvOptions{}),
		"MaxPool2D": MaxPool2D(2, PoolOptions{}, PoolOptions{}),
		"AvgPool2D": AvgPool2D(2, PoolOptions{}, PoolOptions{}),
	}
	for name, layer := range layers {
		if err := layer.Build(NewTape()); err == nil {
			t.Errorf("%s given two options built", name)
		}
	}
}
