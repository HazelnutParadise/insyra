package nn

import (
	"fmt"
	"io"
	"strings"
)

// ValueInfo describes a named model input or output. A negative shape
// dimension is dynamic and accepts any non-negative runtime dimension.
type ValueInfo struct {
	Name     string
	DType    DType
	Shape    []int
	HasShape bool
}

type modelNode struct {
	name       string
	domain     string
	opType     string
	inputs     []string
	outputs    []string
	attributes map[string]protoAttribute
}

type modelGraph struct {
	name         string
	inputSpecs   []ValueInfo
	outputSpecs  []ValueInfo
	nodes        []modelNode
	initializers map[string]*Tensor
}

// Model is a validated ONNX graph. Initializers are materialised during load.
type Model struct {
	inputSpecs   []ValueInfo
	outputSpecs  []ValueInfo
	nodes        []modelNode
	initializers map[string]*Tensor
	opsetVersion int64
}

var supportedOperators = map[string]struct{}{
	"Gemm": {}, "MatMul": {}, "Add": {}, "Sub": {}, "Mul": {}, "Div": {},
	"Relu": {}, "Sigmoid": {}, "Tanh": {}, "Gelu": {}, "Erf": {}, "Sqrt": {}, "Floor": {}, "Pow": {},
	"LeakyRelu": {}, "Exp": {}, "Ceil": {}, "Round": {}, "LayerNormalization": {}, "ReduceMean": {}, "ReduceMin": {}, "Softmax": {}, "Identity": {},
	"Conv": {}, "MaxPool": {}, "AveragePool": {}, "GlobalAveragePool": {},
	"BatchNormalization": {}, "Pad": {},
	"InstanceNormalization": {}, "Resize": {}, "Upsample": {},
	"Clip": {}, "ConstantOfShape": {},
	"Reshape": {}, "Flatten": {}, "Transpose": {}, "Cast": {}, "Constant": {},
	"Concat": {}, "Unsqueeze": {}, "Squeeze": {}, "Expand": {}, "Shape": {}, "Slice": {}, "Split": {},
	"Gather": {}, "GreaterOrEqual": {}, "Equal": {}, "Greater": {}, "Where": {}, "Tile": {}, "NonMaxSuppression": {}, "Loop": {},
	"ai.onnx.ml:LinearRegressor":        {},
	"ai.onnx.ml:LinearClassifier":       {},
	"ai.onnx.ml:TreeEnsembleRegressor":  {},
	"ai.onnx.ml:TreeEnsembleClassifier": {},
	"ai.onnx.ml:Scaler":                 {},
	"ai.onnx.ml:OneHotEncoder":          {},
	"ai.onnx.ml:LabelEncoder":           {},
}

// LoadONNX reads and validates an ONNX ModelProto. The reader is untrusted
// input: malformed bytes and invalid graph data return errors, never panics.
func LoadONNX(r io.Reader) (model *Model, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			model = nil
			err = fmt.Errorf("load onnx model: %v", recovered)
		}
	}()
	data, err := readONNX(r)
	if err != nil {
		return nil, err
	}
	decoded, err := decodeModelProto(data)
	if err != nil {
		return nil, fmt.Errorf("decode onnx model: %w", err)
	}
	return buildModel(decoded)
}

// Inputs returns the model's declared runtime inputs. Initializers that also
// appear in graph.input are omitted because callers do not provide them.
func (m *Model) Inputs() []ValueInfo {
	if m == nil {
		return nil
	}
	return cloneValueInfos(m.inputSpecs)
}

// Outputs returns the model's declared graph outputs.
func (m *Model) Outputs() []ValueInfo {
	if m == nil {
		return nil
	}
	return cloneValueInfos(m.outputSpecs)
}

// OpsetVersion reports the default-domain ONNX opset declared by the model.
func (m *Model) OpsetVersion() int64 {
	if m == nil {
		return 0
	}
	return m.opsetVersion
}

func buildModel(decoded protoModel) (*Model, error) {
	if decoded.graph == nil {
		return nil, fmt.Errorf("onnx model has no graph")
	}
	var defaultOpset int64
	var mlOpset int64
	var mlOpsetDeclared bool
	for _, opset := range decoded.ops {
		switch opset.domain {
		case "", "ai.onnx":
			if opset.version <= 0 {
				return nil, fmt.Errorf("onnx model has invalid default opset version %d", opset.version)
			}
			if defaultOpset != 0 && defaultOpset != opset.version {
				return nil, fmt.Errorf("onnx model declares conflicting default opset versions %d and %d", defaultOpset, opset.version)
			}
			defaultOpset = opset.version
		case "ai.onnx.ml":
			if opset.version <= 0 {
				return nil, fmt.Errorf("onnx model has invalid ai.onnx.ml opset version %d", opset.version)
			}
			if mlOpsetDeclared && mlOpset != opset.version {
				return nil, fmt.Errorf("onnx model declares conflicting ai.onnx.ml opset versions %d and %d", mlOpset, opset.version)
			}
			mlOpset, mlOpsetDeclared = opset.version, true
		}
	}
	if defaultOpset == 0 {
		return nil, fmt.Errorf("onnx model has no default opset")
	}
	usesMLDomain := false
	for _, node := range decoded.graph.nodes {
		if node.domain == "ai.onnx.ml" {
			usesMLDomain = true
			break
		}
	}
	if usesMLDomain && !mlOpsetDeclared {
		return nil, fmt.Errorf("onnx model uses ai.onnx.ml operators without an ai.onnx.ml opset import")
	}
	if mlOpsetDeclared && mlOpset != 3 {
		return nil, fmt.Errorf("unsupported ai.onnx.ml opset version %d", mlOpset)
	}
	unsupported := make([]string, 0)
	seenUnsupported := make(map[string]struct{})
	for _, node := range decoded.graph.nodes {
		operatorName := operatorDisplayName(node.domain, node.opType)
		if _, supported := supportedOperators[operatorKey(node.domain, node.opType)]; !supported {
			if _, exists := seenUnsupported[operatorName]; !exists {
				unsupported = append(unsupported, operatorName)
				seenUnsupported[operatorName] = struct{}{}
			}
		}
	}
	if len(unsupported) > 0 {
		return nil, fmt.Errorf("unsupported operators: %s", strings.Join(unsupported, ", "))
	}

	graph, err := buildGraph(*decoded.graph, decoded.graph.name)
	if err != nil {
		return nil, err
	}
	return &Model{
		inputSpecs:   graph.inputSpecs,
		outputSpecs:  graph.outputSpecs,
		nodes:        graph.nodes,
		initializers: graph.initializers,
		opsetVersion: defaultOpset,
	}, nil
}

func buildGraph(decoded protoGraph, graphLabel string) (modelGraph, error) {
	if graphLabel == "" {
		graphLabel = "<unnamed>"
	}
	graph := modelGraph{name: graphLabel, initializers: make(map[string]*Tensor, len(decoded.initializers))}
	for _, initializer := range decoded.initializers {
		if initializer.name == "" {
			return modelGraph{}, fmt.Errorf("subgraph %q has an unnamed initializer", graphLabel)
		}
		if _, exists := graph.initializers[initializer.name]; exists {
			return modelGraph{}, fmt.Errorf("subgraph %q initializer %q is declared more than once", graphLabel, initializer.name)
		}
		value, err := tensorProtoToTensor(initializer)
		if err != nil {
			return modelGraph{}, fmt.Errorf("subgraph %q: %w", graphLabel, err)
		}
		graph.initializers[initializer.name] = value
	}

	inputNames := make(map[string]struct{}, len(decoded.inputs))
	for index, input := range decoded.inputs {
		if input.name == "" {
			return modelGraph{}, fmt.Errorf("subgraph %q input %d has no name", graphLabel, index)
		}
		if _, duplicate := inputNames[input.name]; duplicate {
			return modelGraph{}, fmt.Errorf("subgraph %q input %q is declared more than once", graphLabel, input.name)
		}
		inputNames[input.name] = struct{}{}
		if _, isInitializer := graph.initializers[input.name]; isInitializer {
			continue
		}
		if input.elemType == 0 {
			return modelGraph{}, fmt.Errorf("subgraph %q input %q has no dtype", graphLabel, input.name)
		}
		if !supportedTensorDType(onnxDType(input.elemType)) {
			return modelGraph{}, fmt.Errorf("subgraph %q input %q has unsupported dtype %s", graphLabel, input.name, onnxDTypeName(input.elemType))
		}
		if input.hasShape {
			if err := validateDeclaredShape(input.shape, fmt.Sprintf("subgraph %q input %q", graphLabel, input.name)); err != nil {
				return modelGraph{}, err
			}
		}
		graph.inputSpecs = append(graph.inputSpecs, valueInfoFromProto(input))
	}
	for index, output := range decoded.outputs {
		if output.name == "" {
			return modelGraph{}, fmt.Errorf("subgraph %q output %d has no name", graphLabel, index)
		}
		for _, existing := range graph.outputSpecs {
			if existing.Name == output.name {
				return modelGraph{}, fmt.Errorf("subgraph %q output %q is declared more than once", graphLabel, output.name)
			}
		}
		if output.elemType == 0 {
			return modelGraph{}, fmt.Errorf("subgraph %q output %q has no dtype", graphLabel, output.name)
		}
		if !supportedTensorDType(onnxDType(output.elemType)) {
			return modelGraph{}, fmt.Errorf("subgraph %q output %q has unsupported dtype %s", graphLabel, output.name, onnxDTypeName(output.elemType))
		}
		if output.hasShape {
			if err := validateDeclaredShape(output.shape, fmt.Sprintf("subgraph %q output %q", graphLabel, output.name)); err != nil {
				return modelGraph{}, err
			}
		}
		graph.outputSpecs = append(graph.outputSpecs, valueInfoFromProto(output))
	}
	if len(graph.outputSpecs) == 0 {
		return modelGraph{}, fmt.Errorf("subgraph %q has no outputs", graphLabel)
	}

	declaredValues := make(map[string]struct{}, len(graph.initializers)+len(inputNames))
	for name := range graph.initializers {
		declaredValues[name] = struct{}{}
	}
	for name := range inputNames {
		declaredValues[name] = struct{}{}
	}
	for index, node := range decoded.nodes {
		if len(node.outputs) == 0 {
			return modelGraph{}, fmt.Errorf("subgraph %q node %d (%s) has no outputs", graphLabel, index, node.opType)
		}
		name := node.name
		if name == "" {
			name = fmt.Sprintf("node_%d", index)
		}
		if _, supported := supportedOperators[operatorKey(node.domain, node.opType)]; !supported {
			return modelGraph{}, fmt.Errorf("unsupported operators in subgraph %q: %s", graphLabel, operatorDisplayName(node.domain, node.opType))
		}
		attributes := make(map[string]protoAttribute, len(node.attributes))
		for _, attribute := range node.attributes {
			if attribute.name == "" {
				return modelGraph{}, fmt.Errorf("subgraph %q node %q has an unnamed attribute", graphLabel, name)
			}
			if _, exists := attributes[attribute.name]; exists {
				return modelGraph{}, fmt.Errorf("subgraph %q node %q declares attribute %q more than once", graphLabel, name, attribute.name)
			}
			if attribute.graph != nil {
				nested, nestedErr := buildGraph(*attribute.graph, stringOrDefault(attribute.graph.name, attribute.name))
				if nestedErr != nil {
					return modelGraph{}, fmt.Errorf("subgraph %q node %q attribute %q: %w", graphLabel, name, attribute.name, nestedErr)
				}
				attribute.graph = nil
				attribute.modelGraph = &nested
			}
			if len(attribute.graphs) > 0 {
				attribute.modelGraphs = make([]*modelGraph, len(attribute.graphs))
				for graphIndex, nestedProto := range attribute.graphs {
					nested, nestedErr := buildGraph(nestedProto, stringOrDefault(nestedProto.name, fmt.Sprintf("%s[%d]", attribute.name, graphIndex)))
					if nestedErr != nil {
						return modelGraph{}, fmt.Errorf("subgraph %q node %q attribute %q[%d]: %w", graphLabel, name, attribute.name, graphIndex, nestedErr)
					}
					attribute.modelGraphs[graphIndex] = &nested
				}
			}
			attributes[attribute.name] = attribute
		}
		if err := validateNodeOutputArity(node, name); err != nil {
			return modelGraph{}, err
		}
		if node.opType == "Loop" {
			body, present := attributes["body"]
			if !present || body.modelGraph == nil {
				return modelGraph{}, fmt.Errorf("subgraph %q node %q Loop has no GRAPH body", graphLabel, name)
			}
			if len(node.outputs) != len(body.modelGraph.outputSpecs)-1 {
				return modelGraph{}, fmt.Errorf("subgraph %q node %q Loop has %d outputs, want %d", graphLabel, name, len(node.outputs), len(body.modelGraph.outputSpecs)-1)
			}
		}
		for outputIndex, output := range node.outputs {
			if output == "" {
				if operatorKey(node.domain, node.opType) == "LayerNormalization" && outputIndex > 0 {
					continue
				}
				return modelGraph{}, fmt.Errorf("subgraph %q node %q output %d has no name", graphLabel, name, outputIndex)
			}
			if _, exists := declaredValues[output]; exists {
				return modelGraph{}, fmt.Errorf("subgraph %q node %q output %q is already declared", graphLabel, name, output)
			}
			declaredValues[output] = struct{}{}
		}
		graph.nodes = append(graph.nodes, modelNode{name: name, domain: node.domain, opType: node.opType, inputs: append([]string(nil), node.inputs...), outputs: append([]string(nil), node.outputs...), attributes: attributes})
	}
	return graph, nil
}

func stringOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func validateNodeOutputArity(node protoNode, name string) error {
	want := 1
	switch operatorKey(node.domain, node.opType) {
	case "ai.onnx.ml:LinearClassifier", "ai.onnx.ml:TreeEnsembleClassifier":
		want = 2
	case "LayerNormalization":
		if len(node.outputs) < 1 || len(node.outputs) > 3 {
			return fmt.Errorf("node %q (%s) has %d outputs, want 1 to 3", name, operatorDisplayName(node.domain, node.opType), len(node.outputs))
		}
		return nil
	case "MaxPool":
		if len(node.outputs) == 2 {
			return fmt.Errorf("node %q MaxPool second Indices output is unsupported", name)
		}
	case "BatchNormalization":
		if len(node.outputs) != 1 {
			return fmt.Errorf("node %q BatchNormalization training-mode extra outputs are unsupported; inference requires one output", name)
		}
	case "Split":
		if len(node.outputs) == 0 {
			return fmt.Errorf("node %q (%s) has no outputs", name, operatorDisplayName(node.domain, node.opType))
		}
		return nil
	case "Loop":
		if len(node.outputs) == 0 {
			return fmt.Errorf("node %q (%s) has no outputs", name, operatorDisplayName(node.domain, node.opType))
		}
		return nil
	}
	if len(node.outputs) != want {
		return fmt.Errorf("node %q (%s) has %d outputs, want %d", name, operatorDisplayName(node.domain, node.opType), len(node.outputs), want)
	}
	return nil
}

func operatorKey(domain, opType string) string {
	if domain == "" || domain == "ai.onnx" {
		return opType
	}
	return domain + ":" + opType
}

func operatorDisplayName(domain, opType string) string {
	if opType == "" {
		return "<empty>"
	}
	return operatorKey(domain, opType)
}

func supportedTensorDType(dtype DType) bool {
	switch dtype {
	case DTypeFloat32, DTypeInt64, DTypeString, DTypeBool:
		return true
	default:
		return false
	}
}

func valueInfoFromProto(info protoValueInfo) ValueInfo {
	return ValueInfo{
		Name:     info.name,
		DType:    onnxDType(info.elemType),
		Shape:    append([]int(nil), info.shape...),
		HasShape: info.hasShape,
	}
}

func cloneValueInfos(values []ValueInfo) []ValueInfo {
	result := make([]ValueInfo, len(values))
	for index, value := range values {
		result[index] = ValueInfo{
			Name:     value.Name,
			DType:    value.DType,
			Shape:    append([]int(nil), value.Shape...),
			HasShape: value.HasShape,
		}
	}
	return result
}

func validateDeclaredShape(shape []int, name string) error {
	for index, dimension := range shape {
		if dimension < -1 {
			return fmt.Errorf("%s has invalid dimension %d at index %d", name, dimension, index)
		}
	}
	return nil
}

func onnxDType(dataType int32) DType {
	switch dataType {
	case 1:
		return DTypeFloat32
	case 2:
		return DTypeUInt8
	case 3:
		return DTypeInt8
	case 4:
		return DTypeUInt16
	case 5:
		return DTypeInt16
	case 6:
		// Tensor stores ONNX INT32 controls and outputs in the existing int64
		// representation; the decoder already widens INT32 initializers here.
		return DTypeInt64
	case 7:
		return DTypeInt64
	case 8:
		return DTypeString
	case 9:
		return DTypeBool
	case 10:
		return DTypeFloat16
	case 11:
		return DTypeFloat64
	case 12:
		return DTypeUInt32
	case 13:
		return DTypeUInt64
	case 16:
		return DTypeBFloat16
	case 14, 15, 17, 18, 19, 20:
		return DTypeFloat8
	default:
		return DType(fmt.Sprintf("onnx(%d)", dataType))
	}
}

func onnxDTypeName(dataType int32) string {
	return dtypeName(onnxDType(dataType))
}
