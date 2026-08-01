package ml

// This file contains the small, wire-compatible ONNX protobuf surface needed
// by the exporters below. The fields and numbers are taken from ONNX's
// onnx.proto schema. Keeping the serializer here avoids a protoc or cgo
// dependency while still producing ordinary ModelProto files that standard
// ONNX runtimes can read.

import (
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

type onnxModelProto struct {
	IRVersion    int64
	ProducerName string
	Graph        *onnxGraphProto
	OpsetImport  []onnxOperatorSetIdProto
}

type onnxOperatorSetIdProto struct {
	Domain  string
	Version int64
}

type onnxGraphProto struct {
	Nodes        []onnxNodeProto
	Name         string
	Initializers []onnxTensorProto
	Inputs       []onnxValueInfoProto
	Outputs      []onnxValueInfoProto
}

type onnxNodeProto struct {
	Inputs     []string
	Outputs    []string
	Name       string
	OpType     string
	Attributes []onnxAttributeProto
	Domain     string
}

type onnxAttributeProto struct {
	Name     string
	Type     int32
	Float    float32
	HasFloat bool
	Int      int64
	HasInt   bool
	String   []byte
	Tensor   *onnxTensorProto
	Floats   []float32
	Ints     []int64
	Strings  [][]byte
}

type onnxTensorProto struct {
	Dims      []int64
	DataType  int32
	FloatData []float32
	Int32Data []int32
	Int64Data []int64
	Name      string
	RawData   []byte
}

type onnxValueInfoProto struct {
	Name     string
	ElemType int32
	Shape    []int64
}

func (m onnxModelProto) marshal() []byte {
	var out []byte
	out = onnxAppendInt64(out, 1, m.IRVersion)
	out = onnxAppendString(out, 2, m.ProducerName)
	if m.Graph != nil {
		out = onnxAppendMessage(out, 7, m.Graph.marshal())
	}
	for _, opset := range m.OpsetImport {
		out = onnxAppendMessage(out, 8, opset.marshal())
	}
	return out
}

func (m onnxOperatorSetIdProto) marshal() []byte {
	var out []byte
	out = onnxAppendString(out, 1, m.Domain)
	out = onnxAppendInt64(out, 2, m.Version)
	return out
}

func (g onnxGraphProto) marshal() []byte {
	var out []byte
	for _, node := range g.Nodes {
		out = onnxAppendMessage(out, 1, node.marshal())
	}
	out = onnxAppendString(out, 2, g.Name)
	for _, initializer := range g.Initializers {
		out = onnxAppendMessage(out, 5, initializer.marshal())
	}
	for _, input := range g.Inputs {
		out = onnxAppendMessage(out, 11, input.marshal())
	}
	for _, output := range g.Outputs {
		out = onnxAppendMessage(out, 12, output.marshal())
	}
	return out
}

func (n onnxNodeProto) marshal() []byte {
	var out []byte
	for _, input := range n.Inputs {
		out = onnxAppendString(out, 1, input)
	}
	for _, output := range n.Outputs {
		out = onnxAppendString(out, 2, output)
	}
	out = onnxAppendString(out, 3, n.Name)
	out = onnxAppendString(out, 4, n.OpType)
	for _, attribute := range n.Attributes {
		out = onnxAppendMessage(out, 5, attribute.marshal())
	}
	out = onnxAppendString(out, 7, n.Domain)
	return out
}

func (a onnxAttributeProto) marshal() []byte {
	var out []byte
	out = onnxAppendString(out, 1, a.Name)
	if a.Type != 0 {
		out = onnxAppendInt32(out, 20, a.Type)
	}
	if a.HasFloat || a.Float != 0 {
		out = onnxAppendFixed32(out, 2, math.Float32bits(a.Float))
	}
	if a.HasInt || a.Int != 0 {
		out = onnxAppendInt64EvenZero(out, 3, a.Int)
	}
	// The s field may only be present on a STRING attribute. Writing it
	// unconditionally — even empty — made every INT and FLOAT attribute carry a
	// stray string data field, which onnxruntime rejects as "type field and
	// data field mismatch" and refuses to load the whole model. The empty-even
	// form stays, because an empty string VALUE on a string attribute is legal
	// and must still be encoded.
	if a.Type == 3 {
		out = onnxAppendBytesEvenEmpty(out, 4, a.String)
	}
	if a.Tensor != nil {
		out = onnxAppendMessage(out, 5, a.Tensor.marshal())
	}
	out = onnxAppendPackedFloat32(out, 7, a.Floats)
	out = onnxAppendPackedInt64(out, 8, a.Ints)
	for _, value := range a.Strings {
		out = onnxAppendBytesEvenEmpty(out, 9, value)
	}
	return out
}

func (t onnxTensorProto) marshal() []byte {
	var out []byte
	out = onnxAppendPackedInt64(out, 1, t.Dims)
	out = onnxAppendInt32(out, 2, t.DataType)
	out = onnxAppendPackedFloat32(out, 4, t.FloatData)
	out = onnxAppendPackedInt32(out, 5, t.Int32Data)
	out = onnxAppendPackedInt64(out, 7, t.Int64Data)
	out = onnxAppendString(out, 8, t.Name)
	out = onnxAppendBytes(out, 9, t.RawData)
	return out
}

func (v onnxValueInfoProto) marshal() []byte {
	var out []byte
	out = onnxAppendString(out, 1, v.Name)
	out = onnxAppendMessage(out, 2, onnxTypeProto{ElemType: v.ElemType, Shape: v.Shape}.marshal())
	return out
}

type onnxTypeProto struct {
	ElemType int32
	Shape    []int64
}

func (t onnxTypeProto) marshal() []byte {
	var out []byte
	tensorType := onnxTensorTypeProto{ElemType: t.ElemType, Shape: t.Shape}.marshal()
	out = onnxAppendMessage(out, 1, tensorType)
	return out
}

type onnxTensorTypeProto struct {
	ElemType int32
	Shape    []int64
}

func (t onnxTensorTypeProto) marshal() []byte {
	var out []byte
	out = onnxAppendInt32(out, 1, t.ElemType)
	var shape []byte
	for _, dim := range t.Shape {
		var dimension []byte
		dimension = onnxAppendInt64(dimension, 1, dim)
		shape = onnxAppendMessage(shape, 1, dimension)
	}
	out = onnxAppendMessage(out, 2, shape)
	return out
}

func onnxAppendTag(out []byte, number int, typ protowire.Type) []byte {
	return protowire.AppendTag(out, protowire.Number(number), typ)
}

func onnxAppendBytes(out []byte, number int, value []byte) []byte {
	if len(value) == 0 {
		return out
	}
	return onnxAppendBytesEvenEmpty(out, number, value)
}

func onnxAppendBytesEvenEmpty(out []byte, number int, value []byte) []byte {
	out = onnxAppendTag(out, number, protowire.BytesType)
	return protowire.AppendBytes(out, value)
}

func onnxAppendMessage(out []byte, number int, value []byte) []byte {
	return onnxAppendBytes(out, number, value)
}

func onnxAppendString(out []byte, number int, value string) []byte {
	if value == "" {
		return out
	}
	out = onnxAppendTag(out, number, protowire.BytesType)
	return protowire.AppendString(out, value)
}

func onnxAppendInt32(out []byte, number int, value int32) []byte {
	if value == 0 {
		return out
	}
	out = onnxAppendTag(out, number, protowire.VarintType)
	return protowire.AppendVarint(out, uint64(value))
}

func onnxAppendInt64(out []byte, number int, value int64) []byte {
	if value == 0 {
		return out
	}
	return onnxAppendInt64EvenZero(out, number, value)
}

func onnxAppendInt64EvenZero(out []byte, number int, value int64) []byte {
	out = onnxAppendTag(out, number, protowire.VarintType)
	return protowire.AppendVarint(out, uint64(value))
}

func onnxAppendFixed32(out []byte, number int, value uint32) []byte {
	out = onnxAppendTag(out, number, protowire.Fixed32Type)
	return protowire.AppendFixed32(out, value)
}

func onnxAppendPackedInt32(out []byte, number int, values []int32) []byte {
	if len(values) == 0 {
		return out
	}
	var payload []byte
	for _, value := range values {
		payload = protowire.AppendVarint(payload, uint64(value))
	}
	return onnxAppendBytes(out, number, payload)
}

func onnxAppendPackedInt64(out []byte, number int, values []int64) []byte {
	if len(values) == 0 {
		return out
	}
	var payload []byte
	for _, value := range values {
		payload = protowire.AppendVarint(payload, uint64(value))
	}
	return onnxAppendBytes(out, number, payload)
}

func onnxAppendPackedFloat32(out []byte, number int, values []float32) []byte {
	if len(values) == 0 {
		return out
	}
	var payload []byte
	for _, value := range values {
		payload = protowire.AppendFixed32(payload, math.Float32bits(value))
	}
	return onnxAppendBytes(out, number, payload)
}
