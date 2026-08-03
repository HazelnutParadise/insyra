package dl

import (
	"slices"
	"testing"
)

func TestClipSupportsOptionalBounds(t *testing.T) {
	input := mustTestTensor(t, []int{4}, []float32{-2, -0.5, 0.5, 2})

	withoutBounds, err := Clip(input, nil, nil)
	if err != nil {
		t.Fatalf("Clip without bounds: %v", err)
	}
	assertTestTensor(t, withoutBounds, []int{4}, input.data, 0)

	minimum := mustTestTensor(t, nil, []float32{-1})
	maximum := mustTestTensor(t, []int{1}, []float32{1})
	withBounds, err := Clip(input, minimum, maximum)
	if err != nil {
		t.Fatalf("Clip with bounds: %v", err)
	}
	assertTestTensor(t, withBounds, []int{4}, []float32{-1, -0.5, 0.5, 1}, 0)
}

func TestModelClipAcceptsInitializerAndRuntimeBounds(t *testing.T) {
	input := mustTestTensor(t, []int{4}, []float32{-2, -0.5, 0.5, 2})
	minimum := mustTestTensor(t, nil, []float32{-1})
	maximum := mustTestTensor(t, nil, []float32{1})
	node := modelNode{name: "clip-runtime", opType: "Clip", inputs: []string{"X", "min", "max"}, outputs: []string{"Y"}}

	initializerModel := &Model{
		inputSpecs:  []ValueInfo{{Name: "X", DType: DTypeFloat32, Shape: []int{4}, HasShape: true}},
		outputSpecs: []ValueInfo{{Name: "Y", DType: DTypeFloat32, Shape: []int{4}, HasShape: true}},
		nodes:       []modelNode{node},
		initializers: map[string]*Tensor{
			"min": minimum,
			"max": maximum,
		},
	}
	outputs, err := initializerModel.Run(map[string]*Tensor{"X": input})
	if err != nil {
		t.Fatalf("Clip initializer bounds: %v", err)
	}
	assertTestTensor(t, outputs["Y"], []int{4}, []float32{-1, -0.5, 0.5, 1}, 0)

	runtimeModel := &Model{
		inputSpecs: []ValueInfo{
			{Name: "X", DType: DTypeFloat32, Shape: []int{4}, HasShape: true},
			{Name: "min", DType: DTypeFloat32, Shape: nil, HasShape: true},
			{Name: "max", DType: DTypeFloat32, Shape: nil, HasShape: true},
		},
		outputSpecs: []ValueInfo{{Name: "Y", DType: DTypeFloat32, Shape: []int{4}, HasShape: true}},
		nodes:       []modelNode{node},
	}
	outputs, err = runtimeModel.Run(map[string]*Tensor{"X": input, "min": minimum, "max": maximum})
	if err != nil {
		t.Fatalf("Clip runtime bounds: %v", err)
	}
	assertTestTensor(t, outputs["Y"], []int{4}, []float32{-1, -0.5, 0.5, 1}, 0)
}

func TestConstantOfShapeSupportsDefaultAndTypedValues(t *testing.T) {
	shape := mustTestInt64Tensor(t, []int{2}, []int64{2, 3})
	defaultValue, err := ConstantOfShape(shape, nil)
	if err != nil {
		t.Fatalf("ConstantOfShape default: %v", err)
	}
	assertTestTensor(t, defaultValue, []int{2, 3}, make([]float32, 6), 0)

	value := mustTestInt64Tensor(t, nil, []int64{7})
	typedValue, err := ConstantOfShape(shape, value)
	if err != nil {
		t.Fatalf("ConstantOfShape int64: %v", err)
	}
	if typedValue.DType() != DTypeInt64 || !slices.Equal(typedValue.int64Data, []int64{7, 7, 7, 7, 7, 7}) {
		t.Fatalf("ConstantOfShape int64 = dtype %s data %v", typedValue.DType(), typedValue.int64Data)
	}
}

func TestRuntimeShapeGraphWaitsForControlProducers(t *testing.T) {
	model := &Model{
		inputSpecs:  []ValueInfo{{Name: "X", DType: DTypeFloat32, Shape: []int{2, 3}, HasShape: true}},
		outputSpecs: []ValueInfo{{Name: "Y", DType: DTypeFloat32, Shape: []int{3, 2}, HasShape: true}},
		initializers: map[string]*Tensor{
			"index": mustTestInt64Tensor(t, nil, []int64{1}),
			"axes":  mustTestInt64Tensor(t, []int{1}, []int64{0}),
			"tail":  mustTestInt64Tensor(t, []int{1}, []int64{2}),
		},
		// The nodes intentionally appear in reverse dependency order. Every
		// shape tensor after Shape is produced during this execution.
		nodes: []modelNode{
			{name: "reshape", opType: "Reshape", inputs: []string{"X", "target"}, outputs: []string{"Y"}},
			{name: "concat", opType: "Concat", inputs: []string{"unsqueezed", "tail"}, outputs: []string{"target"}, attributes: map[string]protoAttribute{"axis": {hasInt: true, intValue: 0}}},
			{name: "unsqueeze", opType: "Unsqueeze", inputs: []string{"selected", "axes"}, outputs: []string{"unsqueezed"}},
			{name: "gather", opType: "Gather", inputs: []string{"shape", "index"}, outputs: []string{"selected"}},
			{name: "shape", opType: "Shape", inputs: []string{"X"}, outputs: []string{"shape"}},
		},
	}
	outputs, err := model.Run(map[string]*Tensor{"X": mustTestTensor(t, []int{2, 3}, []float32{1, 2, 3, 4, 5, 6})})
	if err != nil {
		t.Fatalf("runtime shape graph: %v", err)
	}
	assertTestTensor(t, outputs["Y"], []int{3, 2}, []float32{1, 2, 3, 4, 5, 6}, 0)
}
