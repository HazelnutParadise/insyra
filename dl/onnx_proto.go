package dl

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

const maxONNXBytes = 256 << 20

type protoModel struct {
	graph     *protoGraph
	ops       []protoOpset
	irVersion int64
}

type protoOpset struct {
	domain  string
	version int64
}

type protoGraph struct {
	nodes        []protoNode
	name         string
	initializers []protoTensor
	inputs       []protoValueInfo
	outputs      []protoValueInfo
}

type protoNode struct {
	inputs     []string
	outputs    []string
	name       string
	opType     string
	domain     string
	attributes []protoAttribute
}

type protoAttribute struct {
	name       string
	typeID     int32
	floatValue float32
	hasFloat   bool
	intValue   int64
	hasInt     bool
	string     []byte
	tensor     *protoTensor
	floats     []float32
	ints       []int64
	strings    [][]byte
}

type protoTensor struct {
	dims      []int64
	dataType  int32
	floatData []float32
	int32Data []int32
	int64Data []int64
	name      string
	rawData   []byte
}

type protoValueInfo struct {
	name     string
	elemType int32
	shape    []int
	hasShape bool
}

func readONNX(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("onnx reader is nil")
	}
	data, err := io.ReadAll(io.LimitReader(r, maxONNXBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read onnx model: %w", err)
	}
	if len(data) > maxONNXBytes {
		return nil, fmt.Errorf("onnx model exceeds the %d-byte limit", maxONNXBytes)
	}
	return data, nil
}

func decodeModelProto(data []byte) (model protoModel, err error) {
	defer recoverDecode(&err, "model protobuf")
	err = decodeFields(data, "model", func(number int, typ protowire.Type, value []byte) error {
		switch number {
		case 1:
			parsed, parseErr := parseInt64(typ, value, "model ir_version")
			if parseErr != nil {
				return parseErr
			}
			model.irVersion = parsed
		case 2:
			if typ != protowire.BytesType {
				return wrongWireType("model producer_name", protowire.BytesType, typ)
			}
		case 7:
			if typ != protowire.BytesType {
				return wrongWireType("model graph", protowire.BytesType, typ)
			}
			graph, decodeErr := decodeGraphProto(value)
			if decodeErr != nil {
				return decodeErr
			}
			model.graph = &graph
		case 8:
			if typ != protowire.BytesType {
				return wrongWireType("model opset_import", protowire.BytesType, typ)
			}
			opset, decodeErr := decodeOpsetProto(value)
			if decodeErr != nil {
				return decodeErr
			}
			model.ops = append(model.ops, opset)
		}
		return nil
	})
	return model, err
}

func decodeOpsetProto(data []byte) (opset protoOpset, err error) {
	err = decodeFields(data, "opset", func(number int, typ protowire.Type, value []byte) error {
		switch number {
		case 1:
			if typ != protowire.BytesType {
				return wrongWireType("opset domain", protowire.BytesType, typ)
			}
			opset.domain = string(value)
		case 2:
			parsed, parseErr := parseInt64(typ, value, "opset version")
			if parseErr != nil {
				return parseErr
			}
			opset.version = parsed
		}
		return nil
	})
	return opset, err
}

func decodeGraphProto(data []byte) (graph protoGraph, err error) {
	err = decodeFields(data, "graph", func(number int, typ protowire.Type, value []byte) error {
		switch number {
		case 1:
			if typ != protowire.BytesType {
				return wrongWireType("graph node", protowire.BytesType, typ)
			}
			node, decodeErr := decodeNodeProto(value)
			if decodeErr != nil {
				return decodeErr
			}
			graph.nodes = append(graph.nodes, node)
		case 2:
			if typ != protowire.BytesType {
				return wrongWireType("graph name", protowire.BytesType, typ)
			}
			graph.name = string(value)
		case 5:
			if typ != protowire.BytesType {
				return wrongWireType("graph initializer", protowire.BytesType, typ)
			}
			initializer, decodeErr := decodeTensorProto(value)
			if decodeErr != nil {
				return decodeErr
			}
			graph.initializers = append(graph.initializers, initializer)
		case 11:
			if typ != protowire.BytesType {
				return wrongWireType("graph input", protowire.BytesType, typ)
			}
			input, decodeErr := decodeValueInfoProto(value)
			if decodeErr != nil {
				return decodeErr
			}
			graph.inputs = append(graph.inputs, input)
		case 12:
			if typ != protowire.BytesType {
				return wrongWireType("graph output", protowire.BytesType, typ)
			}
			output, decodeErr := decodeValueInfoProto(value)
			if decodeErr != nil {
				return decodeErr
			}
			graph.outputs = append(graph.outputs, output)
		}
		return nil
	})
	return graph, err
}

func decodeNodeProto(data []byte) (node protoNode, err error) {
	err = decodeFields(data, "node", func(number int, typ protowire.Type, value []byte) error {
		switch number {
		case 1:
			if typ != protowire.BytesType {
				return wrongWireType("node input", protowire.BytesType, typ)
			}
			node.inputs = append(node.inputs, string(value))
		case 2:
			if typ != protowire.BytesType {
				return wrongWireType("node output", protowire.BytesType, typ)
			}
			node.outputs = append(node.outputs, string(value))
		case 3:
			if typ != protowire.BytesType {
				return wrongWireType("node name", protowire.BytesType, typ)
			}
			node.name = string(value)
		case 4:
			if typ != protowire.BytesType {
				return wrongWireType("node op_type", protowire.BytesType, typ)
			}
			node.opType = string(value)
		case 5:
			if typ != protowire.BytesType {
				return wrongWireType("node attribute", protowire.BytesType, typ)
			}
			attribute, decodeErr := decodeAttributeProto(value)
			if decodeErr != nil {
				return decodeErr
			}
			node.attributes = append(node.attributes, attribute)
		case 7:
			if typ != protowire.BytesType {
				return wrongWireType("node domain", protowire.BytesType, typ)
			}
			node.domain = string(value)
		}
		return nil
	})
	return node, err
}

func decodeAttributeProto(data []byte) (attribute protoAttribute, err error) {
	err = decodeFields(data, "attribute", func(number int, typ protowire.Type, value []byte) error {
		switch number {
		case 1:
			if typ != protowire.BytesType {
				return wrongWireType("attribute name", protowire.BytesType, typ)
			}
			attribute.name = string(value)
		case 2:
			parsed, parseErr := parseFloat32(typ, value, "attribute float")
			if parseErr != nil {
				return parseErr
			}
			attribute.floatValue, attribute.hasFloat = parsed, true
		case 3:
			parsed, parseErr := parseInt64(typ, value, "attribute int")
			if parseErr != nil {
				return parseErr
			}
			attribute.intValue, attribute.hasInt = parsed, true
		case 4:
			if typ != protowire.BytesType {
				return wrongWireType("attribute string", protowire.BytesType, typ)
			}
			attribute.string = append([]byte(nil), value...)
		case 5:
			if typ != protowire.BytesType {
				return wrongWireType("attribute tensor", protowire.BytesType, typ)
			}
			tensor, decodeErr := decodeTensorProto(value)
			if decodeErr != nil {
				return decodeErr
			}
			attribute.tensor = &tensor
		case 7:
			parsed, parseErr := appendFloat32Values(attribute.floats, typ, value, "attribute floats")
			if parseErr != nil {
				return parseErr
			}
			attribute.floats = parsed
		case 8:
			parsed, parseErr := appendInt64Values(attribute.ints, typ, value, "attribute ints")
			if parseErr != nil {
				return parseErr
			}
			attribute.ints = parsed
		case 9:
			if typ != protowire.BytesType {
				return wrongWireType("attribute strings", protowire.BytesType, typ)
			}
			attribute.strings = append(attribute.strings, append([]byte(nil), value...))
		case 20:
			parsed, parseErr := parseInt32(typ, value, "attribute type")
			if parseErr != nil {
				return parseErr
			}
			attribute.typeID = parsed
		}
		return nil
	})
	return attribute, err
}

func decodeTensorProto(data []byte) (tensor protoTensor, err error) {
	err = decodeFields(data, "tensor", func(number int, typ protowire.Type, value []byte) error {
		switch number {
		case 1:
			parsed, parseErr := appendInt64Values(tensor.dims, typ, value, "tensor dims")
			if parseErr != nil {
				return parseErr
			}
			tensor.dims = parsed
		case 2:
			parsed, parseErr := parseInt32(typ, value, "tensor data_type")
			if parseErr != nil {
				return parseErr
			}
			tensor.dataType = parsed
		case 4:
			parsed, parseErr := appendFloat32Values(tensor.floatData, typ, value, "tensor float_data")
			if parseErr != nil {
				return parseErr
			}
			tensor.floatData = parsed
		case 5:
			parsed, parseErr := appendInt32Values(tensor.int32Data, typ, value, "tensor int32_data")
			if parseErr != nil {
				return parseErr
			}
			tensor.int32Data = parsed
		case 7:
			parsed, parseErr := appendInt64Values(tensor.int64Data, typ, value, "tensor int64_data")
			if parseErr != nil {
				return parseErr
			}
			tensor.int64Data = parsed
		case 8:
			if typ != protowire.BytesType {
				return wrongWireType("tensor name", protowire.BytesType, typ)
			}
			tensor.name = string(value)
		case 9:
			if typ != protowire.BytesType {
				return wrongWireType("tensor raw_data", protowire.BytesType, typ)
			}
			tensor.rawData = append([]byte(nil), value...)
		}
		return nil
	})
	return tensor, err
}

func decodeValueInfoProto(data []byte) (info protoValueInfo, err error) {
	err = decodeFields(data, "value_info", func(number int, typ protowire.Type, value []byte) error {
		switch number {
		case 1:
			if typ != protowire.BytesType {
				return wrongWireType("value_info name", protowire.BytesType, typ)
			}
			info.name = string(value)
		case 2:
			if typ != protowire.BytesType {
				return wrongWireType("value_info type", protowire.BytesType, typ)
			}
			elemType, shape, hasShape, decodeErr := decodeTypeProto(value)
			if decodeErr != nil {
				return decodeErr
			}
			info.elemType, info.shape, info.hasShape = elemType, shape, hasShape
		}
		return nil
	})
	return info, err
}

func decodeTypeProto(data []byte) (elemType int32, shape []int, hasShape bool, err error) {
	err = decodeFields(data, "type", func(number int, typ protowire.Type, value []byte) error {
		if number != 1 {
			return nil
		}
		if typ != protowire.BytesType {
			return wrongWireType("type tensor_type", protowire.BytesType, typ)
		}
		elemType, shape, hasShape, err = decodeTensorTypeProto(value)
		return err
	})
	return elemType, shape, hasShape, err
}

func decodeTensorTypeProto(data []byte) (elemType int32, shape []int, hasShape bool, err error) {
	err = decodeFields(data, "tensor_type", func(number int, typ protowire.Type, value []byte) error {
		switch number {
		case 1:
			parsed, parseErr := parseInt32(typ, value, "tensor_type elem_type")
			if parseErr != nil {
				return parseErr
			}
			elemType = parsed
		case 2:
			if typ != protowire.BytesType {
				return wrongWireType("tensor_type shape", protowire.BytesType, typ)
			}
			parsed, parseErr := decodeTensorShapeProto(value)
			if parseErr != nil {
				return parseErr
			}
			shape, hasShape = parsed, true
		}
		return nil
	})
	return elemType, shape, hasShape, err
}

func decodeTensorShapeProto(data []byte) (shape []int, err error) {
	err = decodeFields(data, "tensor_shape", func(number int, typ protowire.Type, value []byte) error {
		if number != 1 {
			return nil
		}
		if typ != protowire.BytesType {
			return wrongWireType("tensor_shape dim", protowire.BytesType, typ)
		}
		dimension, decodeErr := decodeDimensionProto(value)
		if decodeErr != nil {
			return decodeErr
		}
		shape = append(shape, dimension)
		return nil
	})
	return shape, err
}

func decodeDimensionProto(data []byte) (dimension int, err error) {
	var raw int64
	hasValue, hasParameter := false, false
	err = decodeFields(data, "dimension", func(number int, typ protowire.Type, value []byte) error {
		switch number {
		case 1:
			parsed, parseErr := parseInt64(typ, value, "dimension value")
			if parseErr != nil {
				return parseErr
			}
			raw, hasValue = parsed, true
		case 2:
			if typ != protowire.BytesType {
				return wrongWireType("dimension parameter", protowire.BytesType, typ)
			}
			hasParameter = true
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if hasValue && hasParameter {
		return 0, fmt.Errorf("dimension has both a value and a parameter")
	}
	if hasParameter || !hasValue {
		return -1, nil
	}
	if raw < 0 {
		return 0, fmt.Errorf("dimension value %d is negative", raw)
	}
	return intFromInt64(raw, "dimension value")
}

func decodeFields(data []byte, message string, visit func(number int, typ protowire.Type, value []byte) error) error {
	for len(data) > 0 {
		number, typ, tagSize := protowire.ConsumeTag(data)
		if tagSize < 0 {
			return fmt.Errorf("invalid %s protobuf tag: %d", message, tagSize)
		}
		if number == 0 {
			return fmt.Errorf("invalid %s protobuf field number 0", message)
		}
		if tagSize > len(data) {
			return fmt.Errorf("invalid %s protobuf tag size %d", message, tagSize)
		}
		field := data[tagSize:]
		valueSize := protowire.ConsumeFieldValue(number, typ, field)
		if valueSize < 0 || valueSize > len(field) {
			return fmt.Errorf("invalid %s protobuf field %d: %d", message, number, valueSize)
		}
		value := field[:valueSize]
		if typ == protowire.BytesType {
			var consumed int
			value, consumed = protowire.ConsumeBytes(field)
			if consumed < 0 || consumed > len(field) {
				return fmt.Errorf("invalid %s protobuf bytes field %d: %d", message, number, consumed)
			}
			valueSize = consumed
		}
		if err := visit(int(number), typ, value); err != nil {
			return err
		}
		data = field[valueSize:]
	}
	return nil
}

func parseVarint(typ protowire.Type, value []byte, field string) (uint64, error) {
	if typ != protowire.VarintType {
		return 0, wrongWireType(field, protowire.VarintType, typ)
	}
	parsed, size := protowire.ConsumeVarint(value)
	if size < 0 || size != len(value) {
		return 0, fmt.Errorf("invalid %s protobuf varint", field)
	}
	return parsed, nil
}

func parseInt64(typ protowire.Type, value []byte, field string) (int64, error) {
	parsed, err := parseVarint(typ, value, field)
	return int64(parsed), err
}

func parseInt32(typ protowire.Type, value []byte, field string) (int32, error) {
	parsed, err := parseVarint(typ, value, field)
	return int32(parsed), err
}

func parseFixed32(typ protowire.Type, value []byte, field string) (uint32, error) {
	if typ != protowire.Fixed32Type {
		return 0, wrongWireType(field, protowire.Fixed32Type, typ)
	}
	parsed, size := protowire.ConsumeFixed32(value)
	if size < 0 || size != len(value) {
		return 0, fmt.Errorf("invalid %s protobuf fixed32", field)
	}
	return parsed, nil
}

func parseFloat32(typ protowire.Type, value []byte, field string) (float32, error) {
	parsed, err := parseFixed32(typ, value, field)
	return math.Float32frombits(parsed), err
}

func appendInt64Values(values []int64, typ protowire.Type, value []byte, field string) ([]int64, error) {
	if typ == protowire.VarintType {
		parsed, err := parseInt64(typ, value, field)
		if err != nil {
			return nil, err
		}
		return append(values, parsed), nil
	}
	if typ != protowire.BytesType {
		return nil, wrongWireType(field, protowire.BytesType, typ)
	}
	for len(value) > 0 {
		parsed, size := protowire.ConsumeVarint(value)
		if size < 0 || size == 0 {
			return nil, fmt.Errorf("invalid packed %s protobuf varint", field)
		}
		values = append(values, int64(parsed))
		value = value[size:]
	}
	return values, nil
}

func appendInt32Values(values []int32, typ protowire.Type, value []byte, field string) ([]int32, error) {
	if typ == protowire.VarintType {
		parsed, err := parseInt32(typ, value, field)
		if err != nil {
			return nil, err
		}
		return append(values, parsed), nil
	}
	if typ != protowire.BytesType {
		return nil, wrongWireType(field, protowire.BytesType, typ)
	}
	for len(value) > 0 {
		parsed, size := protowire.ConsumeVarint(value)
		if size < 0 || size == 0 {
			return nil, fmt.Errorf("invalid packed %s protobuf varint", field)
		}
		values = append(values, int32(parsed))
		value = value[size:]
	}
	return values, nil
}

func appendFloat32Values(values []float32, typ protowire.Type, value []byte, field string) ([]float32, error) {
	if typ == protowire.Fixed32Type {
		parsed, err := parseFloat32(typ, value, field)
		if err != nil {
			return nil, err
		}
		return append(values, parsed), nil
	}
	if typ != protowire.BytesType {
		return nil, wrongWireType(field, protowire.BytesType, typ)
	}
	if len(value)%4 != 0 {
		return nil, fmt.Errorf("packed %s has %d bytes, not a multiple of 4", field, len(value))
	}
	for len(value) > 0 {
		values = append(values, math.Float32frombits(binary.LittleEndian.Uint32(value[:4])))
		value = value[4:]
	}
	return values, nil
}

func wrongWireType(field string, want, got protowire.Type) error {
	return fmt.Errorf("%s has wire type %v, want %v", field, want, got)
}

func recoverDecode(target *error, context string) {
	if recovered := recover(); recovered != nil {
		*target = fmt.Errorf("decode %s: %v", context, recovered)
	}
}

func intFromInt64(value int64, name string) (int, error) {
	if value < 0 || value > int64(maxInt()) {
		return 0, fmt.Errorf("%s %d does not fit in an int", name, value)
	}
	return int(value), nil
}

func tensorProtoToTensor(proto protoTensor) (*Tensor, error) {
	shape := make([]int, len(proto.dims))
	for index, dimension := range proto.dims {
		converted, err := intFromInt64(dimension, fmt.Sprintf("tensor %q dimension", proto.name))
		if err != nil {
			return nil, err
		}
		shape[index] = converted
	}
	_, _, count, err := makeLayout(shape)
	if err != nil {
		return nil, fmt.Errorf("tensor %q: %w", proto.name, err)
	}
	switch proto.dataType {
	case 1:
		values, dataErr := tensorFloatData(proto, count)
		if dataErr != nil {
			return nil, dataErr
		}
		return newFloat32Tensor(shape, values)
	case 6:
		values, dataErr := tensorInt32Data(proto, count)
		if dataErr != nil {
			return nil, dataErr
		}
		converted := make([]int64, len(values))
		for index, value := range values {
			converted[index] = int64(value)
		}
		return newInt64Tensor(shape, converted)
	case 7:
		values, dataErr := tensorInt64Data(proto, count)
		if dataErr != nil {
			return nil, dataErr
		}
		return newInt64Tensor(shape, values)
	default:
		return nil, fmt.Errorf("tensor %q has unsupported dtype %s", proto.name, onnxDTypeName(proto.dataType))
	}
}

func tensorFloatData(proto protoTensor, count int) ([]float32, error) {
	if proto.rawData != nil {
		if count > maxInt()/4 || len(proto.rawData) != count*4 {
			return nil, fmt.Errorf("tensor %q raw_data has %d bytes, want %d", proto.name, len(proto.rawData), count*4)
		}
		values := make([]float32, count)
		for index := range values {
			values[index] = math.Float32frombits(binary.LittleEndian.Uint32(proto.rawData[index*4:]))
		}
		return values, nil
	}
	if len(proto.floatData) != count {
		return nil, fmt.Errorf("tensor %q has %d float32 values, want %d", proto.name, len(proto.floatData), count)
	}
	return append([]float32(nil), proto.floatData...), nil
}

func tensorInt32Data(proto protoTensor, count int) ([]int32, error) {
	if proto.rawData != nil {
		if count > maxInt()/4 || len(proto.rawData) != count*4 {
			return nil, fmt.Errorf("tensor %q raw_data has %d bytes, want %d", proto.name, len(proto.rawData), count*4)
		}
		values := make([]int32, count)
		for index := range values {
			values[index] = int32(binary.LittleEndian.Uint32(proto.rawData[index*4:]))
		}
		return values, nil
	}
	if len(proto.int32Data) != count {
		return nil, fmt.Errorf("tensor %q has %d int32 values, want %d", proto.name, len(proto.int32Data), count)
	}
	return append([]int32(nil), proto.int32Data...), nil
}

func tensorInt64Data(proto protoTensor, count int) ([]int64, error) {
	if proto.rawData != nil {
		if count > maxInt()/8 || len(proto.rawData) != count*8 {
			return nil, fmt.Errorf("tensor %q raw_data has %d bytes, want %d", proto.name, len(proto.rawData), count*8)
		}
		values := make([]int64, count)
		for index := range values {
			values[index] = int64(binary.LittleEndian.Uint64(proto.rawData[index*8:]))
		}
		return values, nil
	}
	if len(proto.int64Data) != count {
		return nil, fmt.Errorf("tensor %q has %d int64 values, want %d", proto.name, len(proto.int64Data), count)
	}
	return append([]int64(nil), proto.int64Data...), nil
}
