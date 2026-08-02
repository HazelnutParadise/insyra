package dl

import (
	"fmt"
)

// Run executes the graph in dependency order and returns named output tensors.
func (m *Model) Run(inputs map[string]*Tensor) (outputs map[string]*Tensor, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			outputs = nil
			err = fmt.Errorf("run onnx model: %v", recovered)
		}
	}()
	if m == nil {
		return nil, fmt.Errorf("model is nil")
	}
	values := make(map[string]*Tensor, len(m.initializers)+len(inputs))
	for name, initializer := range m.initializers {
		copy, copyErr := copyTensor(initializer)
		if copyErr != nil {
			return nil, fmt.Errorf("initializer %q: %w", name, copyErr)
		}
		values[name] = copy
	}

	declaredInputs := make(map[string]ValueInfo, len(m.inputSpecs))
	for _, spec := range m.inputSpecs {
		declaredInputs[spec.Name] = spec
	}
	for name, input := range inputs {
		spec, declared := declaredInputs[name]
		if !declared {
			return nil, fmt.Errorf("input %q is not declared by the model", name)
		}
		if err := validateModelInput(name, spec, input); err != nil {
			return nil, err
		}
		copy, copyErr := copyTensor(input)
		if copyErr != nil {
			return nil, fmt.Errorf("input %q: %w", name, copyErr)
		}
		values[name] = copy
	}
	for _, spec := range m.inputSpecs {
		if _, present := values[spec.Name]; !present {
			return nil, fmt.Errorf("input %q is missing", spec.Name)
		}
	}

	done := make([]bool, len(m.nodes))
	for completed := 0; completed < len(m.nodes); {
		progress := false
		for index, node := range m.nodes {
			if done[index] || !nodeInputsReady(node, values) {
				continue
			}
			produced, executeErr := executeNode(node, values)
			if executeErr != nil {
				return nil, fmt.Errorf("node %q (%s): %w", node.name, node.opType, executeErr)
			}
			for name, value := range produced {
				values[name] = value
			}
			done[index] = true
			completed++
			progress = true
		}
		if !progress {
			for index, node := range m.nodes {
				if done[index] {
					continue
				}
				for _, input := range node.inputs {
					if input != "" {
						if _, present := values[input]; !present {
							return nil, fmt.Errorf("node %q cannot execute because input %q is unavailable", node.name, input)
						}
					}
				}
				return nil, fmt.Errorf("node %q cannot execute because its dependencies form a cycle", node.name)
			}
		}
	}

	outputs = make(map[string]*Tensor, len(m.outputSpecs))
	for _, spec := range m.outputSpecs {
		value, present := values[spec.Name]
		if !present {
			return nil, fmt.Errorf("model output %q was not produced", spec.Name)
		}
		if err := validateModelValue(spec.Name, spec, value); err != nil {
			return nil, err
		}
		copy, copyErr := copyTensor(value)
		if copyErr != nil {
			return nil, fmt.Errorf("model output %q: %w", spec.Name, copyErr)
		}
		outputs[spec.Name] = copy
	}
	return outputs, nil
}

func validateModelInput(name string, spec ValueInfo, input *Tensor) error {
	if input == nil {
		return fmt.Errorf("input %q is nil", name)
	}
	if spec.DType != DTypeFloat32 {
		return fmt.Errorf("input %q declares unsupported dtype %s", name, dtypeName(spec.DType))
	}
	if input.dtype != spec.DType {
		return fmt.Errorf("input %q has dtype %s, want %s", name, dtypeName(input.dtype), dtypeName(spec.DType))
	}
	return validateModelValue(name, spec, input)
}

func validateModelValue(name string, spec ValueInfo, value *Tensor) error {
	if value == nil {
		return fmt.Errorf("value %q is nil", name)
	}
	if !spec.HasShape {
		return nil
	}
	if len(spec.Shape) != len(value.shape) {
		return fmt.Errorf("value %q has shape %v, want rank %d", name, value.shape, len(spec.Shape))
	}
	for index, expected := range spec.Shape {
		if expected >= 0 && expected != value.shape[index] {
			return fmt.Errorf("value %q has shape %v, want %v", name, value.shape, spec.Shape)
		}
	}
	return nil
}

func nodeInputsReady(node modelNode, values map[string]*Tensor) bool {
	for _, input := range node.inputs {
		if input == "" {
			continue
		}
		if _, present := values[input]; !present {
			return false
		}
	}
	return true
}

func executeNode(node modelNode, values map[string]*Tensor) (map[string]*Tensor, error) {
	if len(node.outputs) != 1 {
		return nil, fmt.Errorf("operator %s requires exactly one output, got %d", node.opType, len(node.outputs))
	}
	input := func(index int) (*Tensor, error) {
		if index >= len(node.inputs) || node.inputs[index] == "" {
			return nil, fmt.Errorf("input %d is missing", index)
		}
		value, present := values[node.inputs[index]]
		if !present {
			return nil, fmt.Errorf("input %q is unavailable", node.inputs[index])
		}
		return value, nil
	}
	attribute := func(name string) (protoAttribute, bool) {
		value, present := node.attributes[name]
		return value, present
	}
	outputName := node.outputs[0]
	var result *Tensor
	var err error
	switch node.opType {
	case "Gemm":
		a, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		b, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		var c *Tensor
		if len(node.inputs) > 2 && node.inputs[2] != "" {
			c, err = input(2)
			if err != nil {
				return nil, err
			}
		}
		opts := GemmOptions{Alpha: 1, Beta: 1}
		if value, present := attribute("alpha"); present {
			if !value.hasFloat {
				return nil, fmt.Errorf("attribute alpha is not a float")
			}
			opts.Alpha = value.floatValue
		}
		if value, present := attribute("beta"); present {
			if !value.hasFloat {
				return nil, fmt.Errorf("attribute beta is not a float")
			}
			opts.Beta = value.floatValue
		}
		if value, present := attribute("transA"); present {
			if !value.hasInt {
				return nil, fmt.Errorf("attribute transA is not an integer")
			}
			opts.TransA = value.intValue != 0
		}
		if value, present := attribute("transB"); present {
			if !value.hasInt {
				return nil, fmt.Errorf("attribute transB is not an integer")
			}
			opts.TransB = value.intValue != 0
		}
		result, err = Gemm(a, b, c, opts)
	case "MatMul":
		a, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		b, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		result, err = MatMul(a, b)
	case "Add", "Sub", "Mul":
		left, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		right, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		switch node.opType {
		case "Add":
			result, err = Add(left, right)
		case "Sub":
			result, err = Sub(left, right)
		case "Mul":
			result, err = Mul(left, right)
		}
	case "Relu", "Sigmoid", "Tanh":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		switch node.opType {
		case "Relu":
			result, err = Relu(value)
		case "Sigmoid":
			result, err = Sigmoid(value)
		case "Tanh":
			result, err = Tanh(value)
		}
	case "Softmax":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		axis := -1
		if attributeValue, present := attribute("axis"); present {
			if !attributeValue.hasInt || attributeValue.intValue < minIntValue() || attributeValue.intValue > int64(maxInt()) {
				return nil, fmt.Errorf("attribute axis is not a valid integer")
			}
			axis = int(attributeValue.intValue)
		}
		result, err = Softmax(value, axis)
	case "Identity":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		result, err = Identity(value)
	case "Reshape":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		shapeValue, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		shape, shapeErr := reshapeShape(shapeValue)
		if shapeErr != nil {
			return nil, shapeErr
		}
		allowZero := false
		if attributeValue, present := attribute("allowzero"); present {
			if !attributeValue.hasInt {
				return nil, fmt.Errorf("attribute allowzero is not an integer")
			}
			allowZero = attributeValue.intValue != 0
		}
		result, err = reshapeWithOptions(value, shape, allowZero)
	case "Flatten":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		axis := 1
		if attributeValue, present := attribute("axis"); present {
			if !attributeValue.hasInt || attributeValue.intValue < minIntValue() || attributeValue.intValue > int64(maxInt()) {
				return nil, fmt.Errorf("attribute axis is not a valid integer")
			}
			axis = int(attributeValue.intValue)
		}
		result, err = Flatten(value, axis)
	case "Transpose":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		if attributeValue, present := attribute("perm"); present {
			permutation, permutationErr := attributeInts(attributeValue, "perm")
			if permutationErr != nil {
				return nil, permutationErr
			}
			result, err = Transpose(value, permutation)
		} else {
			result, err = Transpose(value)
		}
	case "Cast":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		attributeValue, present := attribute("to")
		if !present || !attributeValue.hasInt || attributeValue.intValue < -1<<31 || attributeValue.intValue > 1<<31-1 {
			return nil, fmt.Errorf("attribute to is not a valid ONNX dtype")
		}
		result, err = Cast(value, onnxDType(int32(attributeValue.intValue)))
	case "Constant":
		attributeValue, present := attribute("value")
		if !present || attributeValue.tensor == nil {
			return nil, fmt.Errorf("constant has no tensor value")
		}
		value, valueErr := tensorProtoToTensor(*attributeValue.tensor)
		if valueErr != nil {
			return nil, valueErr
		}
		result, err = Constant(value)
	default:
		return nil, fmt.Errorf("unsupported operator %s", node.opType)
	}
	if err != nil {
		return nil, err
	}
	return map[string]*Tensor{outputName: result}, nil
}

func reshapeShape(input *Tensor) ([]int, error) {
	if input == nil {
		return nil, fmt.Errorf("reshape shape tensor is nil")
	}
	if input.dtype != DTypeInt64 {
		return nil, fmt.Errorf("reshape shape has unsupported dtype %s", dtypeName(input.dtype))
	}
	shape := make([]int, len(input.int64Data))
	for index, value := range input.int64Data {
		if value < -1 || value > int64(maxInt()) {
			return nil, fmt.Errorf("reshape dimension %d does not fit in an int", value)
		}
		shape[index] = int(value)
	}
	return shape, nil
}

func attributeInts(attribute protoAttribute, name string) ([]int, error) {
	values := make([]int, len(attribute.ints))
	for index, value := range attribute.ints {
		if value < minIntValue() || value > int64(maxInt()) {
			return nil, fmt.Errorf("attribute %s value %d does not fit in an int", name, value)
		}
		values[index] = int(value)
	}
	return values, nil
}

func minIntValue() int64 {
	return -int64(maxInt()) - 1
}
