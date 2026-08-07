package nn

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

// ExportONNX writes the inference graph represented by s as an ordinary ONNX
// ModelProto. It builds the complete model before touching w, so unsupported
// layers never leave a partial file behind.
func (s *Sequential) ExportONNX(w io.Writer) error {
	if s == nil {
		return fmt.Errorf("export sequential ONNX: sequential is nil")
	}
	if w == nil {
		return fmt.Errorf("export sequential ONNX: writer is nil")
	}
	model, err := buildSequentialONNX(s)
	if err != nil {
		return err
	}
	payload := model.marshal()
	for len(payload) > 0 {
		n, writeErr := w.Write(payload)
		if n < 0 || n > len(payload) {
			return fmt.Errorf("export sequential ONNX: invalid writer count %d", n)
		}
		if writeErr != nil {
			return fmt.Errorf("export sequential ONNX: %w", writeErr)
		}
		if n == 0 {
			return fmt.Errorf("export sequential ONNX: %w", io.ErrShortWrite)
		}
		payload = payload[n:]
	}
	return nil
}

type sequentialONNXModel struct {
	irVersion int64
	producer  string
	graph     sequentialONNXGraph
	opset     int64
}

type sequentialONNXGraph struct {
	nodes        []sequentialONNXNode
	name         string
	initializers []sequentialONNXTensor
	inputs       []sequentialONNXValueInfo
	outputs      []sequentialONNXValueInfo
}

type sequentialONNXNode struct {
	inputs, outputs []string
	name, opType    string
	attributes      []sequentialONNXAttribute
}

type sequentialONNXAttribute struct {
	name     string
	typeID   int32
	float    float32
	hasFloat bool
	integer  int64
	hasInt   bool
	string   []byte
	floats   []float32
	ints     []int64
}

type sequentialONNXTensor struct {
	dims     []int64
	dataType int32
	name     string
	rawData  []byte
}

type sequentialONNXValueInfo struct {
	name     string
	elemType int32
	shape    []int
}

func buildSequentialONNX(s *Sequential) (*sequentialONNXModel, error) {
	inputShape, err := sequentialONNXInputShape(s.layers)
	if err != nil {
		return nil, err
	}
	builder := &sequentialONNXModel{
		irVersion: 9,
		producer:  "insyra.nn",
		opset:     13,
		graph:     sequentialONNXGraph{name: "insyra_nn"},
	}
	builder.graph.inputs = append(builder.graph.inputs, sequentialONNXValueInfo{
		name: "input", elemType: 1, shape: append([]int(nil), inputShape...),
	})
	current := "input"
	shape := append([]int(nil), inputShape...)
	for index, layer := range s.layers {
		output := fmt.Sprintf("value_%d", index)
		var node *sequentialONNXNode
		var nextShape []int
		switch layer := layer.(type) {
		case *denseLayer:
			weight, transposeErr := transposeDenseWeight(layer, layer.weight.Value())
			if transposeErr != nil {
				return nil, fmt.Errorf("layer %d (%s): %w", index, sequentialLayerKind(layer), transposeErr)
			}
			weightName := fmt.Sprintf("layer_%d.weight", index)
			biasName := fmt.Sprintf("layer_%d.bias", index)
			builder.graph.initializers = append(builder.graph.initializers,
				sequentialONNXTensorFromFloat32(weightName, weight),
				sequentialONNXTensorFromFloat32(biasName, layer.bias.Value()))
			node = &sequentialONNXNode{
				inputs: []string{current, weightName, biasName}, outputs: []string{output},
				name: fmt.Sprintf("layer_%d_Dense", index), opType: "Gemm",
				attributes: []sequentialONNXAttribute{
					sequentialONNXAttrFloat("alpha", 1),
					sequentialONNXAttrFloat("beta", 1),
					sequentialONNXAttrInt("transB", 1),
				},
			}
			nextShape = replaceLastShape(shape, layer.out)
		case *activationLayer:
			opType := map[string]string{"ReLU": "Relu", "Sigmoid": "Sigmoid", "Tanh": "Tanh", "Gelu": "Gelu"}[layer.kind]
			if opType == "" {
				return nil, fmt.Errorf("layer %d (%s): ONNX export is unsupported", index, sequentialLayerKind(layer))
			}
			attributes := []sequentialONNXAttribute(nil)
			if opType == "Gelu" {
				attributes = []sequentialONNXAttribute{sequentialONNXAttrString("approximate", "none")}
			}
			node = &sequentialONNXNode{inputs: []string{current}, outputs: []string{output}, name: fmt.Sprintf("layer_%d_%s", index, layer.kind), opType: opType, attributes: attributes}
			nextShape = append([]int(nil), shape...)
		case *conv2DLayer:
			weightName := fmt.Sprintf("layer_%d.weight", index)
			inputs := []string{current, weightName}
			builder.graph.initializers = append(builder.graph.initializers, sequentialONNXTensorFromFloat32(weightName, layer.weight.Value()))
			if layer.bias != nil {
				biasName := fmt.Sprintf("layer_%d.bias", index)
				builder.graph.initializers = append(builder.graph.initializers, sequentialONNXTensorFromFloat32(biasName, layer.bias.Value()))
				inputs = append(inputs, biasName)
			}
			node = &sequentialONNXNode{inputs: inputs, outputs: []string{output}, name: fmt.Sprintf("layer_%d_Conv2D", index), opType: "Conv", attributes: convONNXAttributes(layer.opts)}
			nextShape = replaceChannelShape(shape, layer.out)
		case *batchNorm2DLayer:
			weightName := fmt.Sprintf("layer_%d.weight", index)
			biasName := fmt.Sprintf("layer_%d.bias", index)
			meanName := fmt.Sprintf("layer_%d.running_mean", index)
			varianceName := fmt.Sprintf("layer_%d.running_var", index)
			builder.graph.initializers = append(builder.graph.initializers,
				sequentialONNXTensorFromFloat32(weightName, layer.weight.Value()),
				sequentialONNXTensorFromFloat32(biasName, layer.bias.Value()),
				sequentialONNXTensorFromFloat32(meanName, layer.runningMean),
				sequentialONNXTensorFromFloat32(varianceName, layer.runningVariance))
			node = &sequentialONNXNode{
				inputs: []string{current, weightName, biasName, meanName, varianceName}, outputs: []string{output},
				name: fmt.Sprintf("layer_%d_BatchNorm2D", index), opType: "BatchNormalization",
				attributes: []sequentialONNXAttribute{sequentialONNXAttrFloat("epsilon", layer.epsilon)},
			}
			nextShape = append([]int(nil), shape...)
		case *pool2DLayer:
			opType := "AveragePool"
			if layer.max {
				opType = "MaxPool"
			}
			attributes := poolONNXAttributes(layer)
			node = &sequentialONNXNode{inputs: []string{current}, outputs: []string{output}, name: fmt.Sprintf("layer_%d_%s", index, layer.layerKind()), opType: opType, attributes: attributes}
			nextShape = append([]int(nil), shape...)
		case *globalAvgPoolLayer:
			node = &sequentialONNXNode{inputs: []string{current}, outputs: []string{output}, name: fmt.Sprintf("layer_%d_GlobalAvgPool", index), opType: "GlobalAveragePool"}
			nextShape = append([]int(nil), shape...)
			if len(nextShape) == 4 {
				nextShape[2], nextShape[3] = 1, 1
			}
		case *flattenLayer:
			node = &sequentialONNXNode{inputs: []string{current}, outputs: []string{output}, name: fmt.Sprintf("layer_%d_Flatten", index), opType: "Flatten", attributes: []sequentialONNXAttribute{sequentialONNXAttrInt("axis", 1)}}
			nextShape = flattenONNXShape(shape)
		case *layerNormLayer:
			weightName := fmt.Sprintf("layer_%d.weight", index)
			biasName := fmt.Sprintf("layer_%d.bias", index)
			builder.graph.initializers = append(builder.graph.initializers,
				sequentialONNXTensorFromFloat32(weightName, layer.weight.Value()),
				sequentialONNXTensorFromFloat32(biasName, layer.bias.Value()))
			node = &sequentialONNXNode{
				inputs: []string{current, weightName, biasName}, outputs: []string{output},
				name: fmt.Sprintf("layer_%d_LayerNorm", index), opType: "LayerNormalization",
				attributes: []sequentialONNXAttribute{
					sequentialONNXAttrInt("axis", int64(-len(layer.dims))),
					sequentialONNXAttrFloat("epsilon", layer.epsilon),
				},
			}
			nextShape = append([]int(nil), shape...)
		case *dropoutLayer:
			// Dropout is TrainingOnly and therefore an inference identity.
			continue
		case *funcLayer, *embeddingLayer:
			return nil, fmt.Errorf("layer %d (%s): ONNX export is unsupported", index, sequentialLayerKind(layer))
		case *multiHeadAttentionLayer, *residualLayer:
			return nil, fmt.Errorf("layer %d (%s): ONNX export is unsupported", index, sequentialLayerKind(layer))
		default:
			return nil, fmt.Errorf("layer %d (%s): ONNX export is unsupported", index, sequentialLayerKind(layer))
		}
		builder.graph.nodes = append(builder.graph.nodes, *node)
		current, shape = output, nextShape
	}
	if current != "output" {
		builder.graph.nodes = append(builder.graph.nodes, sequentialONNXNode{
			inputs: []string{current}, outputs: []string{"output"}, name: "output_identity", opType: "Identity",
		})
	}
	builder.graph.outputs = append(builder.graph.outputs, sequentialONNXValueInfo{name: "output", elemType: 1, shape: shape})
	return builder, nil
}

func sequentialONNXInputShape(layers []Layer) ([]int, error) {
	for index, layer := range layers {
		if _, dropout := layer.(*dropoutLayer); dropout {
			continue
		}
		switch layer := layer.(type) {
		case *denseLayer:
			return []int{-1, layer.in}, nil
		case *conv2DLayer:
			return []int{-1, layer.in, -1, -1}, nil
		case *multiHeadAttentionLayer:
			return []int{-1, -1, layer.embed}, nil
		default:
			return nil, fmt.Errorf("layer %d (%s): cannot infer ONNX input shape; the first inference layer must be Dense or Conv2D", index, sequentialLayerKind(layer))
		}
	}
	return nil, fmt.Errorf("sequential has no inference layers")
}

func replaceLastShape(shape []int, value int) []int {
	result := append([]int(nil), shape...)
	if len(result) == 0 {
		return []int{-1, value}
	}
	result[len(result)-1] = value
	return result
}

func replaceChannelShape(shape []int, channels int) []int {
	result := append([]int(nil), shape...)
	if len(result) > 1 {
		result[1] = channels
	}
	return result
}

func flattenONNXShape(shape []int) []int {
	if len(shape) <= 1 {
		return []int{-1, -1}
	}
	right := 1
	unknown := false
	for _, dimension := range shape[1:] {
		if dimension < 0 {
			unknown = true
			break
		}
		right *= dimension
	}
	if unknown {
		right = -1
	}
	return []int{shape[0], right}
}

func convONNXAttributes(options ConvOptions) []sequentialONNXAttribute {
	attributes := make([]sequentialONNXAttribute, 0, 5)
	if len(options.Pads) != 0 {
		attributes = append(attributes, sequentialONNXAttrInts("pads", intSlice64(options.Pads)))
	}
	if len(options.Strides) != 0 {
		attributes = append(attributes, sequentialONNXAttrInts("strides", intSlice64(options.Strides)))
	}
	if len(options.Dilations) != 0 {
		attributes = append(attributes, sequentialONNXAttrInts("dilations", intSlice64(options.Dilations)))
	}
	if options.AutoPad != "" {
		attributes = append(attributes, sequentialONNXAttrString("auto_pad", options.AutoPad))
	}
	if options.Group != 0 && options.Group != 1 {
		attributes = append(attributes, sequentialONNXAttrInt("group", int64(options.Group)))
	}
	return attributes
}

func poolONNXAttributes(layer *pool2DLayer) []sequentialONNXAttribute {
	attributes := []sequentialONNXAttribute{sequentialONNXAttrInts("kernel_shape", []int64{int64(layer.kernel), int64(layer.kernel)})}
	if len(layer.opts.Pads) != 0 {
		attributes = append(attributes, sequentialONNXAttrInts("pads", intSlice64(layer.opts.Pads)))
	}
	if len(layer.opts.Strides) != 0 {
		attributes = append(attributes, sequentialONNXAttrInts("strides", intSlice64(layer.opts.Strides)))
	}
	if layer.opts.AutoPad != "" {
		attributes = append(attributes, sequentialONNXAttrString("auto_pad", layer.opts.AutoPad))
	}
	if !layer.max && layer.opts.CountIncludePad {
		attributes = append(attributes, sequentialONNXAttrInt("count_include_pad", 1))
	}
	if layer.opts.CeilMode != 0 {
		attributes = append(attributes, sequentialONNXAttrInt("ceil_mode", int64(layer.opts.CeilMode)))
	}
	if layer.opts.StorageOrder != 0 {
		attributes = append(attributes, sequentialONNXAttrInt("storage_order", int64(layer.opts.StorageOrder)))
	}
	return attributes
}

func intSlice64(values []int) []int64 {
	result := make([]int64, len(values))
	for index, value := range values {
		result[index] = int64(value)
	}
	return result
}

func sequentialONNXTensorFromFloat32(name string, tensor *Tensor) sequentialONNXTensor {
	raw := make([]byte, len(tensor.data)*4)
	for index, value := range tensor.data {
		binary.LittleEndian.PutUint32(raw[index*4:], math.Float32bits(value))
	}
	dims := make([]int64, len(tensor.shape))
	for index, value := range tensor.shape {
		dims[index] = int64(value)
	}
	return sequentialONNXTensor{dims: dims, dataType: 1, name: name, rawData: raw}
}

func sequentialONNXAttrFloat(name string, value float32) sequentialONNXAttribute {
	return sequentialONNXAttribute{name: name, typeID: 1, float: value, hasFloat: true}
}

func sequentialONNXAttrInt(name string, value int64) sequentialONNXAttribute {
	return sequentialONNXAttribute{name: name, typeID: 2, integer: value, hasInt: true}
}

func sequentialONNXAttrString(name, value string) sequentialONNXAttribute {
	return sequentialONNXAttribute{name: name, typeID: 3, string: []byte(value)}
}

func sequentialONNXAttrInts(name string, values []int64) sequentialONNXAttribute {
	return sequentialONNXAttribute{name: name, typeID: 7, ints: append([]int64(nil), values...)}
}

func (m sequentialONNXModel) marshal() []byte {
	var out []byte
	out = nnONNXAppendInt64(out, 1, m.irVersion)
	out = nnONNXAppendString(out, 2, m.producer)
	out = nnONNXAppendMessage(out, 7, m.graph.marshal())
	opset := nnONNXAppendInt64(nil, 2, m.opset)
	out = nnONNXAppendMessage(out, 8, opset)
	return out
}

func (g sequentialONNXGraph) marshal() []byte {
	var out []byte
	for _, node := range g.nodes {
		out = nnONNXAppendMessage(out, 1, node.marshal())
	}
	out = nnONNXAppendString(out, 2, g.name)
	for _, initializer := range g.initializers {
		out = nnONNXAppendMessage(out, 5, initializer.marshal())
	}
	for _, input := range g.inputs {
		out = nnONNXAppendMessage(out, 11, input.marshal())
	}
	for _, output := range g.outputs {
		out = nnONNXAppendMessage(out, 12, output.marshal())
	}
	return out
}

func (n sequentialONNXNode) marshal() []byte {
	var out []byte
	for _, input := range n.inputs {
		out = nnONNXAppendString(out, 1, input)
	}
	for _, output := range n.outputs {
		out = nnONNXAppendString(out, 2, output)
	}
	out = nnONNXAppendString(out, 3, n.name)
	out = nnONNXAppendString(out, 4, n.opType)
	for _, attribute := range n.attributes {
		out = nnONNXAppendMessage(out, 5, attribute.marshal())
	}
	return out
}

func (a sequentialONNXAttribute) marshal() []byte {
	var out []byte
	out = nnONNXAppendString(out, 1, a.name)
	out = nnONNXAppendInt32(out, 20, a.typeID)
	if a.hasFloat {
		out = nnONNXAppendFixed32(out, 2, math.Float32bits(a.float))
	}
	if a.hasInt {
		out = nnONNXAppendInt64EvenZero(out, 3, a.integer)
	}
	if a.typeID == 3 {
		out = nnONNXAppendBytesEvenEmpty(out, 4, a.string)
	}
	if len(a.floats) != 0 {
		var packed []byte
		for _, value := range a.floats {
			packed = protowire.AppendFixed32(packed, math.Float32bits(value))
		}
		out = nnONNXAppendBytes(out, 7, packed)
	}
	if len(a.ints) != 0 {
		var packed []byte
		for _, value := range a.ints {
			packed = protowire.AppendVarint(packed, uint64(value))
		}
		out = nnONNXAppendBytes(out, 8, packed)
	}
	return out
}

func (t sequentialONNXTensor) marshal() []byte {
	var out []byte
	var dims []byte
	for _, dimension := range t.dims {
		dims = protowire.AppendVarint(dims, uint64(dimension))
	}
	if len(dims) != 0 {
		out = nnONNXAppendBytes(out, 1, dims)
	}
	out = nnONNXAppendInt32(out, 2, t.dataType)
	out = nnONNXAppendString(out, 8, t.name)
	out = nnONNXAppendBytes(out, 9, t.rawData)
	return out
}

func (v sequentialONNXValueInfo) marshal() []byte {
	var out []byte
	out = nnONNXAppendString(out, 1, v.name)
	var tensorType []byte
	tensorType = nnONNXAppendInt32(tensorType, 1, v.elemType)
	var shape []byte
	for _, dimension := range v.shape {
		var dimensionMessage []byte
		if dimension < 0 {
			dimensionMessage = nnONNXAppendString(dimensionMessage, 2, "dynamic")
		} else {
			dimensionMessage = nnONNXAppendInt64(dimensionMessage, 1, int64(dimension))
		}
		shape = nnONNXAppendMessage(shape, 1, dimensionMessage)
	}
	tensorType = nnONNXAppendMessage(tensorType, 2, shape)
	var typeMessage []byte
	typeMessage = nnONNXAppendMessage(typeMessage, 1, tensorType)
	out = nnONNXAppendMessage(out, 2, typeMessage)
	return out
}

func nnONNXAppendTag(out []byte, number int, typ protowire.Type) []byte {
	return protowire.AppendTag(out, protowire.Number(number), typ)
}

func nnONNXAppendBytes(out []byte, number int, value []byte) []byte {
	if len(value) == 0 {
		return out
	}
	return nnONNXAppendBytesEvenEmpty(out, number, value)
}

func nnONNXAppendBytesEvenEmpty(out []byte, number int, value []byte) []byte {
	out = nnONNXAppendTag(out, number, protowire.BytesType)
	return protowire.AppendBytes(out, value)
}

func nnONNXAppendMessage(out []byte, number int, value []byte) []byte {
	return nnONNXAppendBytes(out, number, value)
}

func nnONNXAppendString(out []byte, number int, value string) []byte {
	if value == "" {
		return out
	}
	out = nnONNXAppendTag(out, number, protowire.BytesType)
	return protowire.AppendString(out, value)
}

func nnONNXAppendInt32(out []byte, number int, value int32) []byte {
	if value == 0 {
		return out
	}
	out = nnONNXAppendTag(out, number, protowire.VarintType)
	return protowire.AppendVarint(out, uint64(value))
}

func nnONNXAppendInt64(out []byte, number int, value int64) []byte {
	if value == 0 {
		return out
	}
	return nnONNXAppendInt64EvenZero(out, number, value)
}

func nnONNXAppendInt64EvenZero(out []byte, number int, value int64) []byte {
	out = nnONNXAppendTag(out, number, protowire.VarintType)
	return protowire.AppendVarint(out, uint64(value))
}

func nnONNXAppendFixed32(out []byte, number int, value uint32) []byte {
	out = nnONNXAppendTag(out, number, protowire.Fixed32Type)
	return protowire.AppendFixed32(out, value)
}
