package dl

import (
	"slices"
	"strings"
	"testing"
)

func TestTensorCarriesDTypeAndRowMajorLayout(t *testing.T) {
	input := []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}
	tensor, err := NewTensorWithDType(DTypeFloat32, []int{2, 3, 4}, input)
	if err != nil {
		t.Fatalf("NewTensorWithDType: %v", err)
	}

	if tensor.DType() != DTypeFloat32 {
		t.Fatalf("DType() = %q, want %q", tensor.DType(), DTypeFloat32)
	}
	if got := tensor.Shape(); !slices.Equal(got, []int{2, 3, 4}) {
		t.Fatalf("Shape() = %v, want [2 3 4]", got)
	}
	if got := tensor.Strides(); !slices.Equal(got, []int{12, 4, 1}) {
		t.Fatalf("Strides() = %v, want [12 4 1]", got)
	}
	if tensor.Len() != len(input) {
		t.Fatalf("Len() = %d, want %d", tensor.Len(), len(input))
	}

	input[0] = 99
	shape := tensor.Shape()
	shape[0] = 99
	data := tensor.Data()
	data[0] = 99
	if tensor.Data()[0] != 1 || tensor.Shape()[0] != 2 {
		t.Fatal("tensor exposed mutable storage")
	}
}

func TestTensorRejectsUnsupportedDTypeByName(t *testing.T) {
	_, err := NewTensorWithDType(DTypeFloat64, []int{2}, []float32{1, 2})
	if err == nil {
		t.Fatal("NewTensorWithDType accepted float64")
	}
	if !strings.Contains(err.Error(), "float64") {
		t.Fatalf("error %q does not name the unsupported dtype", err)
	}
}

func TestBroadcastBinaryUsesNumpyTrailingDimensions(t *testing.T) {
	left, err := NewTensor([]int{2, 3}, []float32{1, 2, 3, 4, 5, 6})
	if err != nil {
		t.Fatalf("left tensor: %v", err)
	}
	right, err := NewTensor([]int{3}, []float32{10, 20, 30})
	if err != nil {
		t.Fatalf("right tensor: %v", err)
	}

	got, err := tensorBroadcastBinary(left, right, "add", func(a, b float32) float32 { return a + b })
	if err != nil {
		t.Fatalf("broadcastBinary: %v", err)
	}
	if !slices.Equal(got.Shape(), []int{2, 3}) {
		t.Fatalf("result shape = %v, want [2 3]", got.Shape())
	}
	if want := []float32{11, 22, 33, 14, 25, 36}; !slices.Equal(got.Data(), want) {
		t.Fatalf("result data = %v, want %v", got.Data(), want)
	}
}

func TestBroadcastBinarySupportsScalars(t *testing.T) {
	left, err := NewTensor([]int{2, 2}, []float32{1, 2, 3, 4})
	if err != nil {
		t.Fatalf("left tensor: %v", err)
	}
	right, err := NewTensor(nil, []float32{0.5})
	if err != nil {
		t.Fatalf("scalar tensor: %v", err)
	}

	got, err := tensorBroadcastBinary(left, right, "mul", func(a, b float32) float32 { return a * b })
	if err != nil {
		t.Fatalf("broadcastBinary: %v", err)
	}
	if want := []float32{0.5, 1, 1.5, 2}; !slices.Equal(got.Data(), want) {
		t.Fatalf("result data = %v, want %v", got.Data(), want)
	}
}

func TestBroadcastBinaryNamesBothIncompatibleShapes(t *testing.T) {
	left, err := NewTensor([]int{2, 3}, []float32{1, 2, 3, 4, 5, 6})
	if err != nil {
		t.Fatalf("left tensor: %v", err)
	}
	right, err := NewTensor([]int{2, 4}, []float32{1, 2, 3, 4, 5, 6, 7, 8})
	if err != nil {
		t.Fatalf("right tensor: %v", err)
	}

	_, err = tensorBroadcastBinary(left, right, "add", func(a, b float32) float32 { return a + b })
	if err == nil {
		t.Fatal("broadcastBinary accepted incompatible shapes")
	}
	if !strings.Contains(err.Error(), "[2 3]") || !strings.Contains(err.Error(), "[2 4]") {
		t.Fatalf("error %q does not name both shapes", err)
	}
}
