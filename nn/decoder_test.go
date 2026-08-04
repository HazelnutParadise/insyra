package nn

import (
	"encoding/binary"
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
		{opType: "ConvTranspose", input: "X", output: "C"},
		{opType: "Attention", input: "C", output: "Y"},
	}, true)

	_, err := LoadONNX(strings.NewReader(string(data)))
	if err == nil {
		t.Fatal("LoadONNX accepted unsupported operators")
	}
	message := err.Error()
	for _, operator := range []string{"ConvTranspose", "Attention"} {
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

func TestLoadONNXDecodesHalfInitializers(t *testing.T) {
	f16Raw := make([]byte, 8)
	for index, bits := range []uint16{0x0001, 0x7c00, 0xfc00, 0x7e01} {
		binary.LittleEndian.PutUint16(f16Raw[index*2:], bits)
	}
	data := testONNXModelWithHalfInitializers(
		testONNXInitializer{name: "f16_raw", dataType: 10, dims: []int64{4}, rawData: f16Raw},
		testONNXInitializer{name: "bf16_packed", dataType: 16, dims: []int64{3}, int32Data: []int32{0x3f80, 0x7f80, 0x8001}},
	)
	model, err := LoadONNX(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("LoadONNX: %v", err)
	}
	cases := []struct {
		name string
		want []float32
	}{
		{name: "f16_raw", want: []float32{f16BitsToFloat32(0x0001), f16BitsToFloat32(0x7c00), f16BitsToFloat32(0xfc00), f16BitsToFloat32(0x7e01)}},
		{name: "bf16_packed", want: []float32{bf16BitsToFloat32(0x3f80), bf16BitsToFloat32(0x7f80), bf16BitsToFloat32(0x8001)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := model.initializers[tc.name]
			if got == nil || got.DType() != DTypeFloat32 {
				t.Fatalf("initializer = %#v, want float32 tensor", got)
			}
			values := got.Data()
			if len(values) != len(tc.want) {
				t.Fatalf("values = %v, want %v", values, tc.want)
			}
			for index := range values {
				if math.IsNaN(float64(tc.want[index])) {
					if !math.IsNaN(float64(values[index])) {
						t.Fatalf("values[%d] = %v, want NaN", index, values[index])
					}
					continue
				}
				if math.Float32bits(values[index]) != math.Float32bits(tc.want[index]) {
					t.Fatalf("values[%d] bits = %#08x, want %#08x", index, math.Float32bits(values[index]), math.Float32bits(tc.want[index]))
				}
			}
		})
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

func TestLoadONNXReportsUnsupportedLoopBodyOperator(t *testing.T) {
	bodyNode := appendTestField(nil, 1, protowire.BytesType, []byte("iter"))
	bodyNode = appendTestField(bodyNode, 2, protowire.BytesType, []byte("body_out"))
	bodyNode = appendTestField(bodyNode, 4, protowire.BytesType, []byte("ConvTranspose"))
	body := appendTestField(nil, 1, protowire.BytesType, bodyNode)
	body = appendTestField(body, 2, protowire.BytesType, []byte("loop-body"))
	body = appendTestField(body, 11, protowire.BytesType, testONNXValueInfo("iter", 7, nil))
	body = appendTestField(body, 12, protowire.BytesType, testONNXValueInfo("body_out", 7, nil))

	attribute := appendTestField(nil, 1, protowire.BytesType, []byte("body"))
	attribute = appendTestField(attribute, 6, protowire.BytesType, body)
	attribute = appendTestVarint(attribute, 20, 5)
	node := appendTestField(nil, 1, protowire.BytesType, []byte("trip"))
	node = appendTestField(node, 2, protowire.BytesType, []byte("Y"))
	node = appendTestField(node, 4, protowire.BytesType, []byte("Loop"))
	node = appendTestField(node, 5, protowire.BytesType, attribute)
	graph := appendTestField(nil, 1, protowire.BytesType, node)
	graph = appendTestField(graph, 11, protowire.BytesType, testONNXValueInfo("trip", 7, nil))
	graph = appendTestField(graph, 12, protowire.BytesType, testONNXValueInfo("Y", 7, nil))
	modelBytes := appendTestVarint(nil, 1, 9)
	modelBytes = appendTestField(modelBytes, 7, protowire.BytesType, graph)
	modelBytes = appendTestField(modelBytes, 8, protowire.BytesType, appendTestVarint(nil, 2, 13))

	_, err := LoadONNX(strings.NewReader(string(modelBytes)))
	if err == nil || !strings.Contains(err.Error(), "loop-body") || !strings.Contains(err.Error(), "ConvTranspose") {
		t.Fatalf("LoadONNX error = %v, want loop-body and ConvTranspose", err)
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

type testONNXInitializer struct {
	name      string
	dataType  int32
	dims      []int64
	rawData   []byte
	int32Data []int32
}

func testONNXModelWithHalfInitializers(initializers ...testONNXInitializer) []byte {
	encodedInitializers := make([][]byte, 0, len(initializers))
	for _, initializer := range initializers {
		var encoded []byte
		for _, dimension := range initializer.dims {
			encoded = appendTestVarint(encoded, 1, uint64(dimension))
		}
		encoded = appendTestVarint(encoded, 2, uint64(initializer.dataType))
		for _, value := range initializer.int32Data {
			encoded = appendTestVarint(encoded, 5, uint64(uint32(value)))
		}
		if initializer.rawData != nil {
			encoded = appendTestField(encoded, 9, protowire.BytesType, initializer.rawData)
		}
		encoded = appendTestField(encoded, 8, protowire.BytesType, []byte(initializer.name))
		encodedInitializers = append(encodedInitializers, encoded)
	}
	graph := testONNXGraph([]testONNXNode{{opType: "Identity", input: "X", output: "Y"}}, true, 1, encodedInitializers...)
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
