package dl

import (
	"math"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestLoadONNXRejectsMalformedBytesWithoutPanicking(t *testing.T) {
	valid := testONNXModel([]testONNXNode{{opType: "Identity", input: "X", output: "Y"}}, true)
	truncatedGraph := appendTestField(nil, 7, protowire.BytesType, []byte{0x0a, 0x02, 0x01})
	cases := []struct {
		name string
		data []byte
	}{
		{name: "truncated protobuf", data: valid[:len(valid)-1]},
		{name: "truncated graph payload", data: truncatedGraph},
		{name: "wrong magic", data: []byte("ONNX")},
		{name: "invalid field number", data: []byte{0x00}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("LoadONNX panicked: %v", recovered)
				}
			}()
			if _, err := LoadONNX(strings.NewReader(string(tc.data))); err == nil {
				t.Fatal("LoadONNX accepted malformed bytes")
			}
		})
	}
}

func TestLoadONNXReportsEveryUnsupportedOperatorAtOnce(t *testing.T) {
	data := testONNXModel([]testONNXNode{
		{opType: "Conv", input: "X", output: "C"},
		{opType: "Attention", input: "C", output: "Y"},
	}, true)

	_, err := LoadONNX(strings.NewReader(string(data)))
	if err == nil {
		t.Fatal("LoadONNX accepted unsupported operators")
	}
	message := err.Error()
	for _, operator := range []string{"Conv", "Attention"} {
		if !strings.Contains(message, operator) {
			t.Errorf("error %q does not name unsupported operator %s", message, operator)
		}
	}
}

func TestLoadONNXReadsOpsetAndMaterialisesInitializer(t *testing.T) {
	data := testONNXModelWithFloatInitializer()
	model, err := LoadONNX(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("LoadONNX: %v", err)
	}
	if got := model.OpsetVersion(); got != 13 {
		t.Fatalf("OpsetVersion() = %d, want 13", got)
	}
	inputs := model.Inputs()
	if len(inputs) != 1 || inputs[0].Name != "X" {
		t.Fatalf("Inputs() = %#v, want one input named X", inputs)
	}
	if got := inputs[0].DType; got != DTypeFloat32 {
		t.Fatalf("input dtype = %s, want %s", got, DTypeFloat32)
	}
	if got := inputs[0].Shape; len(got) != 2 || got[0] != -1 || got[1] != 3 {
		t.Fatalf("input shape = %v, want [-1 3]", got)
	}
	initializer, ok := model.initializers["bias"]
	if !ok {
		t.Fatal("model did not materialise the bias initializer")
	}
	if got := initializer.Data(); len(got) != 1 || got[0] != 0.5 {
		t.Fatalf("bias initializer = %v, want [0.5]", got)
	}
}

func TestLoadONNXRejectsUnsupportedInitializerDTypeByName(t *testing.T) {
	data := testONNXModelWithInitializer(10)
	_, err := LoadONNX(strings.NewReader(string(data)))
	if err == nil {
		t.Fatal("LoadONNX accepted an unsupported float16 initializer")
	}
	if !strings.Contains(err.Error(), "float16") {
		t.Fatalf("error %q does not name float16", err)
	}
}

func TestLoadONNXRejectsInitializerElementCountMismatch(t *testing.T) {
	data := testONNXModelWithInitializer(7)
	_, err := LoadONNX(strings.NewReader(string(data)))
	if err == nil {
		t.Fatal("LoadONNX accepted an int64 initializer with the wrong element count")
	}
	if !strings.Contains(err.Error(), "int64 values, want 3") {
		t.Fatalf("error %q does not name the int64 element-count mismatch", err)
	}
}

func TestLoadONNXRejectsUnexpectedNodeOutputArity(t *testing.T) {
	data := testONNXModel([]testONNXNode{
		{opType: "Identity", input: "X", output: "Y", outputs: []string{"Y", "extra"}},
	}, true)
	_, err := LoadONNX(strings.NewReader(string(data)))
	if err == nil || !strings.Contains(err.Error(), "has 2 outputs, want 1") {
		t.Fatalf("output arity error = %v, want a named arity refusal", err)
	}
}

func TestLoadONNXAllowsOptionalLayerNormalizationOutputs(t *testing.T) {
	data := testONNXModel([]testONNXNode{{
		opType:  "LayerNormalization",
		input:   "X",
		outputs: []string{"Y", "", ""},
	}}, true)
	if _, err := LoadONNX(strings.NewReader(string(data))); err != nil {
		t.Fatalf("LoadONNX rejected optional LayerNormalization outputs: %v", err)
	}
}

func TestLoadONNXValidatesMLDomainOpsetImport(t *testing.T) {
	base := []testONNXNode{{opType: "Scaler", domain: "ai.onnx.ml", input: "X", output: "Y"}}
	withoutImport := testONNXModelWithMLImport(base, 0)
	if _, err := LoadONNX(strings.NewReader(string(withoutImport))); err == nil || !strings.Contains(err.Error(), "without an ai.onnx.ml opset import") {
		t.Fatalf("missing ai.onnx.ml import error = %v", err)
	}
	withUnsupportedImport := testONNXModelWithMLImport(base, 2)
	if _, err := LoadONNX(strings.NewReader(string(withUnsupportedImport))); err == nil || !strings.Contains(err.Error(), "unsupported ai.onnx.ml opset version 2") {
		t.Fatalf("unsupported ai.onnx.ml import error = %v", err)
	}
	withSupportedImport := testONNXModelWithMLImport(base, 3)
	if _, err := LoadONNX(strings.NewReader(string(withSupportedImport))); err != nil {
		t.Fatalf("supported ai.onnx.ml import rejected: %v", err)
	}
}

type testONNXNode struct {
	opType  string
	domain  string
	input   string
	output  string
	outputs []string
}

func testONNXModel(nodes []testONNXNode, withInput bool) []byte {
	graph := testONNXGraph(nodes, withInput, 1)
	model := appendTestVarint(nil, 1, 9)
	model = appendTestField(model, 7, protowire.BytesType, graph)
	model = appendTestField(model, 8, protowire.BytesType, appendTestVarint(nil, 2, 13))
	return model
}

func testONNXModelWithInitializer(dataType int32) []byte {
	initializer := appendTestVarint(nil, 1, 3)
	initializer = appendTestVarint(initializer, 2, uint64(dataType))
	initializer = appendTestField(initializer, 7, protowire.BytesType, appendTestVarint(nil, 1, 2))
	initializer = appendTestField(initializer, 8, protowire.BytesType, []byte("shape"))
	graph := testONNXGraph([]testONNXNode{{opType: "Identity", input: "X", output: "Y"}}, true, 1, initializer)
	model := appendTestVarint(nil, 1, 9)
	model = appendTestField(model, 7, protowire.BytesType, graph)
	model = appendTestField(model, 8, protowire.BytesType, appendTestVarint(nil, 2, 13))
	return model
}

func testONNXModelWithFloatInitializer() []byte {
	initializer := appendTestVarint(nil, 1, 1)
	initializer = appendTestVarint(initializer, 2, 1)
	initializer = appendTestField(initializer, 4, protowire.Fixed32Type, protowire.AppendFixed32(nil, math.Float32bits(0.5)))
	initializer = appendTestField(initializer, 8, protowire.BytesType, []byte("bias"))
	graph := testONNXGraph([]testONNXNode{{opType: "Identity", input: "X", output: "Y"}}, true, 1, initializer)
	model := appendTestVarint(nil, 1, 9)
	model = appendTestField(model, 7, protowire.BytesType, graph)
	model = appendTestField(model, 8, protowire.BytesType, appendTestVarint(nil, 2, 13))
	return model
}

func testONNXGraph(nodes []testONNXNode, withInput bool, elemType int32, initializers ...[]byte) []byte {
	var graph []byte
	for _, node := range nodes {
		var encoded []byte
		encoded = appendTestField(encoded, 1, protowire.BytesType, []byte(node.input))
		outputs := node.outputs
		if len(outputs) == 0 {
			outputs = []string{node.output}
		}
		for _, output := range outputs {
			encoded = appendTestField(encoded, 2, protowire.BytesType, []byte(output))
		}
		encoded = appendTestField(encoded, 4, protowire.BytesType, []byte(node.opType))
		if node.domain != "" {
			encoded = appendTestField(encoded, 7, protowire.BytesType, []byte(node.domain))
		}
		graph = appendTestField(graph, 1, protowire.BytesType, encoded)
	}
	for _, initializer := range initializers {
		graph = appendTestField(graph, 5, protowire.BytesType, initializer)
	}
	if withInput {
		graph = appendTestField(graph, 11, protowire.BytesType, testONNXValueInfo("X", elemType, []int64{-1, 3}))
	}
	graph = appendTestField(graph, 12, protowire.BytesType, testONNXValueInfo("Y", 1, []int64{-1, 3}))
	return graph
}

func testONNXModelWithMLImport(nodes []testONNXNode, version int64) []byte {
	graph := testONNXGraph(nodes, true, 1)
	model := appendTestVarint(nil, 1, 9)
	model = appendTestField(model, 7, protowire.BytesType, graph)
	model = appendTestField(model, 8, protowire.BytesType, appendTestVarint(nil, 2, 13))
	if version != 0 {
		opset := appendTestField(nil, 1, protowire.BytesType, []byte("ai.onnx.ml"))
		opset = appendTestVarint(opset, 2, uint64(version))
		model = appendTestField(model, 8, protowire.BytesType, opset)
	}
	return model
}

func testONNXValueInfo(name string, elemType int32, shape []int64) []byte {
	tensorType := appendTestVarint(nil, 1, uint64(elemType))
	var shapeMessage []byte
	for _, dimension := range shape {
		var dimensionMessage []byte
		if dimension < 0 {
			dimensionMessage = appendTestField(dimensionMessage, 2, protowire.BytesType, []byte("batch"))
		} else {
			dimensionMessage = appendTestVarint(dimensionMessage, 1, uint64(dimension))
		}
		shapeMessage = appendTestField(shapeMessage, 1, protowire.BytesType, dimensionMessage)
	}
	tensorType = appendTestField(tensorType, 2, protowire.BytesType, shapeMessage)
	typeMessage := appendTestField(nil, 1, protowire.BytesType, tensorType)
	valueInfo := appendTestField(nil, 1, protowire.BytesType, []byte(name))
	return appendTestField(valueInfo, 2, protowire.BytesType, typeMessage)
}

func appendTestVarint(out []byte, number int, value uint64) []byte {
	out = protowire.AppendTag(out, protowire.Number(number), protowire.VarintType)
	return protowire.AppendVarint(out, value)
}

func appendTestField(out []byte, number int, typ protowire.Type, value []byte) []byte {
	out = protowire.AppendTag(out, protowire.Number(number), typ)
	if typ == protowire.BytesType {
		return protowire.AppendBytes(out, value)
	}
	return append(out, value...)
}
