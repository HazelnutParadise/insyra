package dl

import (
	"fmt"
	"math"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestParallelMACThreshold(t *testing.T) {
	if workers := parallelWorkerCountForMACs(parallelMACThreshold); workers != 1 {
		t.Fatalf("worker count at threshold = %d, want serial", workers)
	}
	if workers := parallelWorkerCountForMACs(parallelMACThreshold + 1); workers != runtime.NumCPU() {
		t.Fatalf("worker count above threshold = %d, want %d", workers, runtime.NumCPU())
	}
}

func TestBitIdenticalMatMul2D(t *testing.T) {
	a := mustTestTensor(t, []int{64, 64}, patternedTestData(64*64))
	b := mustTestTensor(t, []int{64, 64}, patternedTestData(64*64))
	if workers := parallelWorkerCountForMACs(64, 64, 64); workers != runtime.NumCPU() {
		t.Fatalf("2-D MatMul worker count = %d, want %d above threshold", workers, runtime.NumCPU())
	}

	serial, err := matMul2DWithWorkers(a, b, 1)
	if err != nil {
		t.Fatalf("serial 2-D MatMul: %v", err)
	}
	parallel, err := matMul2DWithWorkers(a, b, runtime.NumCPU())
	if err != nil {
		t.Fatalf("parallel 2-D MatMul: %v", err)
	}
	public, err := MatMul(a, b)
	if err != nil {
		t.Fatalf("public 2-D MatMul: %v", err)
	}
	assertExactTensorEqual(t, parallel, serial)
	assertExactTensorEqual(t, public, serial)
}

func TestBitIdenticalBatchedMatMul(t *testing.T) {
	a := mustTestTensor(t, []int{2, 1, 32, 32}, patternedTestData(2*32*32))
	b := mustTestTensor(t, []int{1, 3, 32, 32}, patternedTestData(3*32*32))
	if workers := parallelWorkerCountForMACs(6, 32, 32, 32); workers != runtime.NumCPU() {
		t.Fatalf("batched MatMul worker count = %d, want %d above threshold", workers, runtime.NumCPU())
	}

	serial, err := matMulBatchedWithWorkers(a, b, 1)
	if err != nil {
		t.Fatalf("serial batched MatMul: %v", err)
	}
	parallel, err := matMulBatchedWithWorkers(a, b, runtime.NumCPU())
	if err != nil {
		t.Fatalf("parallel batched MatMul: %v", err)
	}
	public, err := MatMul(a, b)
	if err != nil {
		t.Fatalf("public batched MatMul: %v", err)
	}
	assertExactTensorEqual(t, parallel, serial)
	assertExactTensorEqual(t, public, serial)
}

func TestBitIdenticalConvWithGroupsAndDilation(t *testing.T) {
	input := mustTestTensor(t, []int{4, 4, 16, 16}, patternedTestData(4*4*16*16))
	weights := mustTestTensor(t, []int{8, 2, 3, 3}, patternedTestData(8*2*3*3))
	bias := mustTestTensor(t, []int{8}, patternedTestData(8))
	options := ConvOptions{Pads: []int{2, 2, 2, 2}, Dilations: []int{2, 2}, Group: 2}
	if workers := parallelWorkerCountForMACs(4, 8, 16, 16, 2, 3, 3); workers != runtime.NumCPU() {
		t.Fatalf("Conv worker count = %d, want %d above threshold", workers, runtime.NumCPU())
	}

	serial, err := convWithWorkers(input, weights, bias, 1, options)
	if err != nil {
		t.Fatalf("serial Conv: %v", err)
	}
	parallel, err := convWithWorkers(input, weights, bias, runtime.NumCPU(), options)
	if err != nil {
		t.Fatalf("parallel Conv: %v", err)
	}
	public, err := Conv(input, weights, bias, options)
	if err != nil {
		t.Fatalf("public Conv: %v", err)
	}
	assertExactTensorEqual(t, parallel, serial)
	assertExactTensorEqual(t, public, serial)
}

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

func TestConvSupportsPaddingStrideDilationAndDepthwiseGroups(t *testing.T) {
	input := mustTestTensor(t, []int{1, 1, 3, 3}, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9})
	weights := mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, 1, 1, 1})
	bias := mustTestTensor(t, []int{1}, []float32{0.5})
	got, err := Conv(input, weights, bias, ConvOptions{Pads: []int{1, 1, 1, 1}, Strides: []int{2, 2}})
	if err != nil {
		t.Fatalf("Conv: %v", err)
	}
	assertTestTensor(t, got, []int{1, 1, 2, 2}, []float32{1.5, 5.5, 11.5, 28.5}, 0)

	depthwiseInput := mustTestTensor(t, []int{1, 2, 2, 2}, []float32{1, 2, 3, 4, 10, 20, 30, 40})
	depthwiseWeights := mustTestTensor(t, []int{2, 1, 1, 1}, []float32{2, 3})
	depthwiseBias := mustTestTensor(t, []int{2}, []float32{1, -1})
	depthwise, err := Conv(depthwiseInput, depthwiseWeights, depthwiseBias, ConvOptions{Group: 2})
	if err != nil {
		t.Fatalf("depthwise Conv: %v", err)
	}
	assertTestTensor(t, depthwise, []int{1, 2, 2, 2}, []float32{3, 5, 7, 9, 29, 59, 89, 119}, 0)

	dilated, err := Conv(
		mustTestTensor(t, []int{1, 1, 5, 5}, make([]float32, 25)),
		mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, 2, 3, 4}),
		nil,
		ConvOptions{AutoPad: "SAME_UPPER", Strides: []int{2, 2}, Dilations: []int{2, 2}},
	)
	if err != nil {
		t.Fatalf("dilated Conv: %v", err)
	}
	if got := dilated.Shape(); fmt.Sprint(got) != "[1 1 3 3]" {
		t.Fatalf("dilated Conv shape = %v, want [1 1 3 3]", got)
	}
}

func TestConvRejectsMismatchedShapes(t *testing.T) {
	_, err := Conv(
		mustTestTensor(t, []int{1, 2, 3, 3}, make([]float32, 18)),
		mustTestTensor(t, []int{4, 3, 1, 1}, make([]float32, 12)),
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "input channels") || !strings.Contains(err.Error(), "[1 2 3 3]") || !strings.Contains(err.Error(), "[4 3 1 1]") {
		t.Fatalf("Conv mismatch error = %v, want both input shapes", err)
	}
}

func TestPoolingSupportsPaddingAndAverageCountModes(t *testing.T) {
	input := mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, 2, 3, 4})
	options := PoolOptions{Pads: []int{1, 1, 1, 1}}

	maximum, err := MaxPool(input, []int{2, 2}, options)
	if err != nil {
		t.Fatalf("MaxPool: %v", err)
	}
	assertTestTensor(t, maximum, []int{1, 1, 3, 3}, []float32{1, 2, 2, 3, 4, 4, 3, 4, 4}, 0)

	withoutPadCount, err := AveragePool(input, []int{2, 2}, options)
	if err != nil {
		t.Fatalf("AveragePool count_include_pad=0: %v", err)
	}
	assertTestTensor(t, withoutPadCount, []int{1, 1, 3, 3}, []float32{1, 1.5, 2, 2, 2.5, 3, 3, 3.5, 4}, 0)

	withPadCount, err := AveragePool(input, []int{2, 2}, PoolOptions{Pads: []int{1, 1, 1, 1}, CountIncludePad: true})
	if err != nil {
		t.Fatalf("AveragePool count_include_pad=1: %v", err)
	}
	assertTestTensor(t, withPadCount, []int{1, 1, 3, 3}, []float32{0.25, 0.75, 0.5, 1, 2.5, 1.5, 0.75, 1.75, 1}, 0)

	global, err := GlobalAveragePool(input)
	if err != nil {
		t.Fatalf("GlobalAveragePool: %v", err)
	}
	assertTestTensor(t, global, []int{1, 1, 1, 1}, []float32{2.5}, 0)
}

func TestPoolingRejectsUnsupportedAttributes(t *testing.T) {
	input := mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, 2, 3, 4})
	if _, err := MaxPool(input, []int{2, 2}, PoolOptions{CeilMode: 1}); err == nil || !strings.Contains(err.Error(), "ceil_mode") {
		t.Fatalf("MaxPool ceil_mode error = %v", err)
	}
	if _, err := MaxPool(input, []int{2, 2}, PoolOptions{StorageOrder: 1}); err == nil || !strings.Contains(err.Error(), "storage_order") {
		t.Fatalf("MaxPool storage_order error = %v", err)
	}
}

func TestBatchNormalizationAndConstantPad(t *testing.T) {
	input := mustTestTensor(t, []int{1, 2, 1, 2}, []float32{1, 3, 10, 14})
	scale := mustTestTensor(t, []int{2}, []float32{2, 0.5})
	bias := mustTestTensor(t, []int{2}, []float32{1, -1})
	mean := mustTestTensor(t, []int{2}, []float32{1, 12})
	variance := mustTestTensor(t, []int{2}, []float32{4, 4})
	normalized, err := BatchNormalization(input, scale, bias, mean, variance, 0)
	if err != nil {
		t.Fatalf("BatchNormalization: %v", err)
	}
	assertTestTensor(t, normalized, []int{1, 2, 1, 2}, []float32{1, 3, -1.5, -0.5}, 0)

	padded, err := Pad(mustTestTensor(t, []int{2, 2}, []float32{1, 2, 3, 4}), []int{1, 0, 1, 2}, -1)
	if err != nil {
		t.Fatalf("Pad: %v", err)
	}
	assertTestTensor(t, padded, []int{4, 4}, []float32{
		-1, -1, -1, -1,
		1, 2, -1, -1,
		3, 4, -1, -1,
		-1, -1, -1, -1,
	}, 0)
}

func TestCNNNodeErrorsNameNodesAndUnsupportedForms(t *testing.T) {
	input := mustTestTensor(t, []int{1, 1, 2, 2}, []float32{1, 2, 3, 4})
	values := map[string]*Tensor{"X": input}

	_, err := executeNode(modelNode{
		name:    "bad-max-output",
		opType:  "MaxPool",
		inputs:  []string{"X"},
		outputs: []string{"Y", "Indices"},
		attributes: map[string]protoAttribute{
			"kernel_shape": {ints: []int64{2, 2}},
		},
	}, values, nil)
	if err == nil || !strings.Contains(err.Error(), `node "bad-max-output"`) || !strings.Contains(err.Error(), "Indices") {
		t.Fatalf("MaxPool output error = %v", err)
	}

	_, err = executeNode(modelNode{
		name:    "bad-pad-mode",
		opType:  "Pad",
		inputs:  []string{"X"},
		outputs: []string{"Y"},
		attributes: map[string]protoAttribute{
			"mode": {string: []byte("reflect")},
			"pads": {ints: []int64{1, 0, 1, 0}},
		},
	}, values, nil)
	if err == nil || !strings.Contains(err.Error(), `node "bad-pad-mode"`) || !strings.Contains(err.Error(), "reflect") {
		t.Fatalf("Pad mode error = %v", err)
	}

	_, err = executeNode(modelNode{
		name:    "bad-batch-output",
		opType:  "BatchNormalization",
		inputs:  []string{"X", "scale", "bias", "mean", "variance"},
		outputs: []string{"Y", "mean-output"},
	}, values, nil)
	if err == nil || !strings.Contains(err.Error(), `node "bad-batch-output"`) || !strings.Contains(err.Error(), "training-mode") {
		t.Fatalf("BatchNormalization output error = %v", err)
	}

	_, err = executeNode(modelNode{
		name:    "bad-conv-bias",
		opType:  "Conv",
		inputs:  []string{"X", "W", "B"},
		outputs: []string{"Y"},
	}, map[string]*Tensor{
		"X": input,
		"W": mustTestTensor(t, []int{1, 1, 1, 1}, []float32{1}),
		"B": mustTestTensor(t, []int{2}, []float32{1, 2}),
	}, nil)
	if err == nil || !strings.Contains(err.Error(), `node "bad-conv-bias"`) || !strings.Contains(err.Error(), `"B"`) {
		t.Fatalf("Conv bias error = %v", err)
	}
}

func TestMatMulBatchedBroadcast(t *testing.T) {
	a := mustTestTensor(t, []int{2, 1, 2, 3}, []float32{
		1, 2, 3, 4, 5, 6,
		7, 8, 9, 10, 11, 12,
	})
	b := mustTestTensor(t, []int{1, 3, 3, 2}, []float32{
		1, 2, 3, 4, 5, 6,
		2, 1, 4, 3, 6, 5,
		3, 2, 5, 4, 7, 6,
	})
	got, err := MatMul(a, b)
	if err != nil {
		t.Fatalf("MatMul: %v", err)
	}
	if !slices.Equal(got.Shape(), []int{2, 3, 2, 2}) {
		t.Fatalf("shape = %v, want [2 3 2 2]", got.Shape())
	}
	// The first A batch is reused across B's three batches. The second A
	// batch is checked as well so both broadcast dimensions participate.
	want := []float32{
		22, 28, 49, 64,
		28, 22, 64, 49,
		34, 28, 79, 64,
		76, 100, 103, 136,
		100, 76, 136, 103,
		124, 100, 169, 136,
	}
	if !slices.Equal(got.Data(), want) {
		t.Fatalf("data = %v, want %v", got.Data(), want)
	}
}

func TestMatMulRejectsIncompatibleBatchShapes(t *testing.T) {
	a := mustTestTensor(t, []int{2, 2, 3}, make([]float32, 12))
	b := mustTestTensor(t, []int{3, 4, 3, 2}, make([]float32, 72))
	if _, err := MatMul(a, b); err == nil || !strings.Contains(err.Error(), "[2]") || !strings.Contains(err.Error(), "[3 4]") {
		t.Fatalf("MatMul batch error = %v, want both batch shapes", err)
	}
}

func TestMatMulSupportsVectorEdges(t *testing.T) {
	vector := mustTestTensor(t, []int{3}, []float32{1, 2, 3})
	column := mustTestTensor(t, []int{3, 1}, []float32{4, 5, 6})
	got, err := MatMul(vector, column)
	if err != nil {
		t.Fatalf("vector-column MatMul: %v", err)
	}
	if !slices.Equal(got.Shape(), []int{1}) || !slices.Equal(got.Data(), []float32{32}) {
		t.Fatalf("vector-column result = shape %v data %v", got.Shape(), got.Data())
	}
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

func TestSliceHandlesNegativeStepBounds(t *testing.T) {
	input := mustTestTensor(t, []int{4}, []float32{0, 1, 2, 3})
	reversed, err := Slice(input, []int64{3}, []int64{-5}, []int64{0}, []int64{-1})
	if err != nil {
		t.Fatalf("reverse Slice: %v", err)
	}
	assertTestTensor(t, reversed, []int{4}, []float32{3, 2, 1, 0}, 0)
	clamped, err := Slice(input, []int64{-5}, []int64{-5}, []int64{0}, []int64{-1})
	if err != nil {
		t.Fatalf("clamped reverse Slice: %v", err)
	}
	assertTestTensor(t, clamped, []int{1}, []float32{0}, 0)
	empty, err := Slice(mustTestTensor(t, []int{0}, nil), []int64{0}, []int64{-1}, []int64{0}, []int64{-1})
	if err != nil {
		t.Fatalf("empty reverse Slice: %v", err)
	}
	assertTestTensor(t, empty, []int{0}, nil, 0)
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

func TestModelRejectsRuntimeControlInputs(t *testing.T) {
	cases := []struct {
		name   string
		node   modelNode
		inputs []ValueInfo
		values map[string]*Tensor
	}{
		{
			name: "Squeeze axes",
			node: modelNode{
				name:    "squeeze-runtime-axes",
				opType:  "Squeeze",
				inputs:  []string{"X", "axes"},
				outputs: []string{"Y"},
			},
			inputs: []ValueInfo{
				{Name: "X", DType: DTypeFloat32, Shape: []int{1, 2, 1}, HasShape: true},
				{Name: "axes", DType: DTypeInt64, Shape: []int{1}, HasShape: true},
			},
			values: map[string]*Tensor{
				"X":    mustTestTensor(t, []int{1, 2, 1}, []float32{1, 2}),
				"axes": mustTestInt64Tensor(t, []int{1}, []int64{-1}),
			},
		},
		{
			name: "ReduceMean axes",
			node: modelNode{
				name:    "reduce-runtime-axes",
				opType:  "ReduceMean",
				inputs:  []string{"X", "axes"},
				outputs: []string{"Y"},
			},
			inputs: []ValueInfo{
				{Name: "X", DType: DTypeFloat32, Shape: []int{2, 3}, HasShape: true},
				{Name: "axes", DType: DTypeInt64, Shape: []int{1}, HasShape: true},
			},
			values: map[string]*Tensor{
				"X":    mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}),
				"axes": mustTestInt64Tensor(t, []int{1}, []int64{1}),
			},
		},
		{
			name: "Slice starts",
			node: modelNode{
				name:    "slice-runtime-starts",
				opType:  "Slice",
				inputs:  []string{"X", "starts", "ends"},
				outputs: []string{"Y"},
			},
			inputs: []ValueInfo{
				{Name: "X", DType: DTypeFloat32, Shape: []int{2, 3}, HasShape: true},
				{Name: "starts", DType: DTypeInt64, Shape: []int{1}, HasShape: true},
				{Name: "ends", DType: DTypeInt64, Shape: []int{1}, HasShape: true},
			},
			values: map[string]*Tensor{
				"X":      mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6}),
				"starts": mustTestInt64Tensor(t, []int{1}, []int64{0}),
				"ends":   mustTestInt64Tensor(t, []int{1}, []int64{1}),
			},
		},
		{
			name: "Split sizes",
			node: modelNode{
				name:    "split-runtime-sizes",
				opType:  "Split",
				inputs:  []string{"X", "split"},
				outputs: []string{"Y1", "Y2"},
				attributes: map[string]protoAttribute{
					"axis": {hasInt: true, intValue: 1},
				},
			},
			inputs: []ValueInfo{
				{Name: "X", DType: DTypeFloat32, Shape: []int{1, 4}, HasShape: true},
				{Name: "split", DType: DTypeInt64, Shape: []int{2}, HasShape: true},
			},
			values: map[string]*Tensor{
				"X":     mustTestTensor(t, []int{1, 4}, []float32{1, 2, 3, 4}),
				"split": mustTestInt64Tensor(t, []int{2}, []int64{2, 2}),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := &Model{
				inputSpecs:  tc.inputs,
				outputSpecs: []ValueInfo{{Name: tc.node.outputs[0], DType: DTypeFloat32, HasShape: false}},
				nodes:       []modelNode{tc.node},
			}
			outputs, err := model.Run(tc.values)
			if err == nil {
				t.Fatal("Run unexpectedly accepted a runtime-computed control input")
			}
			if outputs != nil {
				t.Fatalf("Run returned outputs with error: %v", outputs)
			}
			if !strings.Contains(err.Error(), tc.node.name) || !strings.Contains(err.Error(), "runtime-computed") {
				t.Fatalf("Run error = %v, want node name and runtime-computed", err)
			}
		})
	}
}

func TestModelRejectsLayerNormalizationStatisticsOutputs(t *testing.T) {
	model := &Model{
		inputSpecs: []ValueInfo{
			{Name: "X", DType: DTypeFloat32, Shape: []int{1, 2}, HasShape: true},
			{Name: "scale", DType: DTypeFloat32, Shape: []int{2}, HasShape: true},
			{Name: "bias", DType: DTypeFloat32, Shape: []int{2}, HasShape: true},
		},
		outputSpecs: []ValueInfo{
			{Name: "Y", DType: DTypeFloat32, Shape: []int{1, 2}, HasShape: true},
			{Name: "Mean", DType: DTypeFloat32, Shape: []int{1, 1}, HasShape: true},
			{Name: "InvStdDev", DType: DTypeFloat32, Shape: []int{1, 1}, HasShape: true},
		},
		nodes: []modelNode{{
			name:    "encoder-norm-stats",
			opType:  "LayerNormalization",
			inputs:  []string{"X", "scale", "bias"},
			outputs: []string{"Y", "Mean", "InvStdDev"},
		}},
	}
	outputs, err := model.Run(map[string]*Tensor{
		"X":     mustTestTensor(t, []int{1, 2}, []float32{1, 2}),
		"scale": mustTestTensor(t, []int{2}, []float32{1, 1}),
		"bias":  mustTestTensor(t, []int{2}, []float32{0, 0}),
	})
	if err == nil {
		t.Fatal("Run unexpectedly accepted LayerNormalization statistics outputs")
	}
	if outputs != nil {
		t.Fatalf("Run returned outputs with error: %v", outputs)
	}
	if !strings.Contains(err.Error(), "encoder-norm-stats") || !strings.Contains(err.Error(), "Mean and InvStdDev") {
		t.Fatalf("Run error = %v, want node name and unsupported statistics outputs", err)
	}
}

func TestModelRunsLayerNormalizationWhenOnlyPrimaryOutputIsConsumed(t *testing.T) {
	model := &Model{
		inputSpecs: []ValueInfo{
			{Name: "X", DType: DTypeFloat32, Shape: []int{1, 2}, HasShape: true},
			{Name: "scale", DType: DTypeFloat32, Shape: []int{2}, HasShape: true},
			{Name: "bias", DType: DTypeFloat32, Shape: []int{2}, HasShape: true},
		},
		outputSpecs: []ValueInfo{{Name: "Y", DType: DTypeFloat32, HasShape: false}},
		nodes: []modelNode{{
			name:    "encoder-norm-primary",
			opType:  "LayerNormalization",
			inputs:  []string{"X", "scale", "bias"},
			outputs: []string{"Y", "", ""},
		}},
	}
	outputs, err := model.Run(map[string]*Tensor{
		"X":     mustTestTensor(t, []int{1, 2}, []float32{1, 2}),
		"scale": mustTestTensor(t, []int{2}, []float32{1, 1}),
		"bias":  mustTestTensor(t, []int{2}, []float32{0, 0}),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if outputs["Y"] == nil {
		t.Fatal("Run did not return the primary LayerNormalization output")
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

func patternedTestData(size int) []float32 {
	data := make([]float32, size)
	for index := range data {
		data[index] = float32((index%19)-9)*0.125 + float32(index%7)*0.03125
	}
	return data
}

func assertExactTensorEqual(t *testing.T, got, want *Tensor) {
	t.Helper()
	if got == nil || want == nil {
		t.Fatalf("tensor comparison received nil: got=%v want=%v", got, want)
	}
	if !slices.Equal(got.shape, want.shape) {
		t.Fatalf("shape = %v, want %v", got.shape, want.shape)
	}
	if len(got.data) != len(want.data) {
		t.Fatalf("data length = %d, want %d", len(got.data), len(want.data))
	}
	for index := range want.data {
		if got.data[index] != want.data[index] || math.Float32bits(got.data[index]) != math.Float32bits(want.data[index]) {
			t.Fatalf("data[%d] = %g (%08x), want %g (%08x)", index, got.data[index], math.Float32bits(got.data[index]), want.data[index], math.Float32bits(want.data[index]))
		}
	}
}
