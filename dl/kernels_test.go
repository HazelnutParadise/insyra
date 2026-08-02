package dl

import (
	"math"
	"strings"
	"testing"
)

func TestGemmAndMatMul(t *testing.T) {
	a := mustTestTensor(t, []int{2, 2}, []float32{1, 2, 3, 4})
	b := mustTestTensor(t, []int{2, 2}, []float32{5, 6, 7, 8})
	c := mustTestTensor(t, []int{2}, []float32{1, 2})

	gemm, err := Gemm(a, b, c)
	if err != nil {
		t.Fatalf("Gemm: %v", err)
	}
	assertTestTensor(t, gemm, []int{2, 2}, []float32{20, 24, 44, 52}, 0)

	matmul, err := MatMul(a, b)
	if err != nil {
		t.Fatalf("MatMul: %v", err)
	}
	assertTestTensor(t, matmul, []int{2, 2}, []float32{19, 22, 43, 50}, 0)

	transposed, err := Gemm(a, b, nil, GemmOptions{Alpha: 1, Beta: 1, TransA: true})
	if err != nil {
		t.Fatalf("Gemm TransA: %v", err)
	}
	assertTestTensor(t, transposed, []int{2, 2}, []float32{26, 30, 38, 44}, 0)
}

func TestElementwiseKernelsBroadcast(t *testing.T) {
	left := mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6})
	right := mustTestTensor(t, []int{3}, []float32{10, 20, 30})

	add, err := Add(left, right)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	assertTestTensor(t, add, []int{2, 3}, []float32{11, 22, 33, 14, 25, 36}, 0)
	sub, err := Sub(left, right)
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	assertTestTensor(t, sub, []int{2, 3}, []float32{-9, -18, -27, -6, -15, -24}, 0)
	mul, err := Mul(left, right)
	if err != nil {
		t.Fatalf("Mul: %v", err)
	}
	assertTestTensor(t, mul, []int{2, 3}, []float32{10, 40, 90, 40, 100, 180}, 0)

	bad := mustTestTensor(t, []int{2, 4}, []float32{1, 2, 3, 4, 5, 6, 7, 8})
	if _, err := Add(left, bad); err == nil || !strings.Contains(err.Error(), "[2 3]") || !strings.Contains(err.Error(), "[2 4]") {
		t.Fatalf("Add shape error = %v, want both shapes", err)
	}
}

func TestUnaryAndSoftmaxKernels(t *testing.T) {
	input := mustTestTensor(t, []int{1, 3}, []float32{-1, 0, 1})
	relu, err := Relu(input)
	if err != nil {
		t.Fatalf("Relu: %v", err)
	}
	assertTestTensor(t, relu, []int{1, 3}, []float32{0, 0, 1}, 0)

	sigmoid, err := Sigmoid(input)
	if err != nil {
		t.Fatalf("Sigmoid: %v", err)
	}
	assertTestTensor(t, sigmoid, []int{1, 3}, []float32{0.26894143, 0.5, 0.7310586}, 1e-6)

	tanh, err := Tanh(input)
	if err != nil {
		t.Fatalf("Tanh: %v", err)
	}
	assertTestTensor(t, tanh, []int{1, 3}, []float32{-0.7615942, 0, 0.7615942}, 1e-6)

	softmax, err := Softmax(input)
	if err != nil {
		t.Fatalf("Softmax: %v", err)
	}
	assertTestTensor(t, softmax, []int{1, 3}, []float32{0.09003057, 0.24472848, 0.66524094}, 1e-6)
}

func TestShapeAndCopyKernels(t *testing.T) {
	input := mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6})
	reshaped, err := Reshape(input, []int{3, -1})
	if err != nil {
		t.Fatalf("Reshape: %v", err)
	}
	assertTestTensor(t, reshaped, []int{3, 2}, []float32{1, 2, 3, 4, 5, 6}, 0)

	flattened, err := Flatten(mustTestTensor(t, []int{2, 3, 2}, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}))
	if err != nil {
		t.Fatalf("Flatten: %v", err)
	}
	assertTestTensor(t, flattened, []int{2, 6}, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, 0)

	transposed, err := Transpose(input, []int{1, 0})
	if err != nil {
		t.Fatalf("Transpose: %v", err)
	}
	assertTestTensor(t, transposed, []int{3, 2}, []float32{1, 4, 2, 5, 3, 6}, 0)

	identity, err := Identity(input)
	if err != nil {
		t.Fatalf("Identity: %v", err)
	}
	identity.data[0] = 99
	if input.data[0] != 1 {
		t.Fatal("Identity returned shared storage")
	}
	constant, err := Constant(input)
	if err != nil {
		t.Fatalf("Constant: %v", err)
	}
	assertTestTensor(t, constant, []int{2, 3}, input.data, 0)

	cast, err := Cast(input, DTypeFloat32)
	if err != nil {
		t.Fatalf("Cast float32: %v", err)
	}
	assertTestTensor(t, cast, []int{2, 3}, input.data, 0)
	if _, err := Cast(input, DTypeFloat64); err == nil || !strings.Contains(err.Error(), "float64") {
		t.Fatalf("Cast float64 error = %v, want dtype name", err)
	}
}

func TestModelRunsReadyNodesByNameAndValidatesInputs(t *testing.T) {
	modelBytes := testONNXModel([]testONNXNode{
		{opType: "Identity", input: "H", output: "Y"},
		{opType: "Relu", input: "X", output: "H"},
	}, true)
	model, err := LoadONNX(strings.NewReader(string(modelBytes)))
	if err != nil {
		t.Fatalf("LoadONNX: %v", err)
	}
	input := mustTestTensor(t, []int{2, 3}, []float32{-1, 2, -3, 4, -5, 6})
	outputs, err := model.Run(map[string]*Tensor{"X": input})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	assertTestTensor(t, outputs["Y"], []int{2, 3}, []float32{0, 2, 0, 4, 0, 6}, 0)

	if _, err := model.Run(nil); err == nil || !strings.Contains(err.Error(), "X") {
		t.Fatalf("missing input error = %v, want X", err)
	}
	wrongShape := mustTestTensor(t, []int{2, 2}, []float32{1, 2, 3, 4})
	if _, err := model.Run(map[string]*Tensor{"X": wrongShape}); err == nil || !strings.Contains(err.Error(), "X") {
		t.Fatalf("shape error = %v, want X", err)
	}
	wrongType, err := newInt64Tensor([]int{2, 3}, []int64{1, 2, 3, 4, 5, 6})
	if err != nil {
		t.Fatalf("wrong type tensor: %v", err)
	}
	if _, err := model.Run(map[string]*Tensor{"X": wrongType}); err == nil || !strings.Contains(err.Error(), "int64") {
		t.Fatalf("dtype error = %v, want int64", err)
	}
}

func TestModelNamesNodeWhenADataShapeFailsMidGraph(t *testing.T) {
	model := &Model{
		inputSpecs: []ValueInfo{
			{Name: "A", DType: DTypeFloat32, Shape: []int{2, 3}, HasShape: true},
			{Name: "B", DType: DTypeFloat32, Shape: []int{2, 4}, HasShape: true},
		},
		outputSpecs: []ValueInfo{{Name: "Y", DType: DTypeFloat32, Shape: []int{2, 3}, HasShape: true}},
		nodes: []modelNode{{
			name:    "bad-add",
			opType:  "Add",
			inputs:  []string{"A", "B"},
			outputs: []string{"Y"},
		}},
	}
	_, err := model.Run(map[string]*Tensor{
		"A": mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}),
		"B": mustTestTensor(t, []int{2, 4}, []float32{1, 2, 3, 4, 5, 6, 7, 8}),
	})
	if err == nil || !strings.Contains(err.Error(), `node "bad-add"`) {
		t.Fatalf("mid-graph error = %v, want node name", err)
	}
}

func mustTestTensor(t *testing.T, shape []int, data []float32) *Tensor {
	t.Helper()
	tensor, err := NewTensor(shape, data)
	if err != nil {
		t.Fatalf("NewTensor: %v", err)
	}
	return tensor
}

func assertTestTensor(t *testing.T, got *Tensor, shape []int, want []float32, tolerance float32) {
	t.Helper()
	if got == nil {
		t.Fatal("got nil tensor")
	}
	if len(got.shape) != len(shape) {
		t.Fatalf("shape = %v, want %v", got.shape, shape)
	}
	for index := range shape {
		if got.shape[index] != shape[index] {
			t.Fatalf("shape = %v, want %v", got.shape, shape)
		}
	}
	if len(got.data) != len(want) {
		t.Fatalf("data length = %d, want %d", len(got.data), len(want))
	}
	for index := range want {
		if math.Abs(float64(got.data[index]-want[index])) > float64(tolerance) {
			t.Fatalf("data[%d] = %g, want %g", index, got.data[index], want[index])
		}
	}
}
