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
	if !supportedTensorDType(spec.DType) {
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
	if !supportedTensorDType(spec.DType) {
		return fmt.Errorf("value %q declares unsupported dtype %s", name, dtypeName(spec.DType))
	}
	if value.dtype != spec.DType {
		return fmt.Errorf("value %q has dtype %s, want %s", name, dtypeName(value.dtype), dtypeName(spec.DType))
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
	if len(node.outputs) == 0 {
		return nil, fmt.Errorf("operator %s has no outputs", node.opType)
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
	controlInput := func(index int, controlName string) (*Tensor, error) {
		if index >= len(node.inputs) || node.inputs[index] == "" {
			return nil, fmt.Errorf("node %q %s %s input is missing", node.name, operatorDisplayName(node.domain, node.opType), controlName)
		}
		name := node.inputs[index]
		value, present := values[name]
		if !present {
			return nil, fmt.Errorf("node %q %s %s input %q is unavailable", node.name, operatorDisplayName(node.domain, node.opType), controlName, name)
		}
		return value, nil
	}
	attribute := func(name string) (protoAttribute, bool) {
		value, present := node.attributes[name]
		return value, present
	}
	outputName := node.outputs[0]
	var result *Tensor
	produced := make(map[string]*Tensor, len(node.outputs))
	var err error
	switch operatorKey(node.domain, node.opType) {
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
	case "Conv":
		if arityErr := nodeInputArity(node, 2, 3); arityErr != nil {
			return nil, arityErr
		}
		inputValue, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		weights, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		var bias *Tensor
		if len(node.inputs) > 2 && node.inputs[2] != "" {
			bias, inputErr = input(2)
			if inputErr != nil {
				return nil, inputErr
			}
		}
		if len(node.inputs) > 3 {
			for index := 3; index < len(node.inputs); index++ {
				if node.inputs[index] != "" {
					return nil, fmt.Errorf("node %q Conv input %d %q is unsupported; Conv accepts input, weights, and optional bias", node.name, index, node.inputs[index])
				}
			}
		}
		options, optionsErr := convOptionsFromNode(node)
		if optionsErr != nil {
			return nil, optionsErr
		}
		result, err = Conv(inputValue, weights, bias, options)
		if err != nil {
			if bias != nil {
				return nil, fmt.Errorf("node %q Conv inputs %q shape %v, %q shape %v, %q shape %v: %w", node.name, nodeInputName(node, 0), inputValue.shape, nodeInputName(node, 1), weights.shape, nodeInputName(node, 2), bias.shape, err)
			}
			return nil, fmt.Errorf("node %q Conv inputs %q shape %v, %q shape %v: %w", node.name, nodeInputName(node, 0), inputValue.shape, nodeInputName(node, 1), weights.shape, err)
		}
	case "Add", "Sub", "Mul", "Div", "Pow":
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
		case "Div":
			result, err = Div(left, right)
		case "Pow":
			result, err = Pow(left, right)
		}
	case "Relu", "Sigmoid", "Tanh", "Gelu", "Erf", "Sqrt":
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
		case "Gelu":
			approximate := "none"
			if attributeValue, present := attribute("approximate"); present {
				approximate = string(attributeValue.string)
			}
			result, err = Gelu(value, approximate)
		case "Erf":
			result, err = Erf(value)
		case "Sqrt":
			result, err = Sqrt(value)
		}
	case "Clip":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		var minimum, maximum *Tensor
		if len(node.inputs) > 1 && node.inputs[1] != "" {
			minimum, err = controlInput(1, "min")
			if err != nil {
				return nil, err
			}
		}
		if len(node.inputs) > 2 && node.inputs[2] != "" {
			maximum, err = controlInput(2, "max")
			if err != nil {
				return nil, err
			}
		}
		result, err = Clip(value, minimum, maximum)
	case "LayerNormalization":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		scale, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		var bias *Tensor
		if len(node.inputs) > 2 && node.inputs[2] != "" {
			bias, inputErr = input(2)
			if inputErr != nil {
				return nil, inputErr
			}
		} else {
			bias, inputErr = newZeroFloat32Tensor(scale.shape)
			if inputErr != nil {
				return nil, inputErr
			}
		}
		axis, axisErr := nodeAxis(attribute, "axis", -1)
		if axisErr != nil {
			return nil, axisErr
		}
		epsilon := float32(1e-5)
		if attributeValue, present := attribute("epsilon"); present {
			if !attributeValue.hasFloat {
				return nil, fmt.Errorf("attribute epsilon is not a float")
			}
			epsilon = attributeValue.floatValue
		}
		requestedStatistics := 0
		for _, output := range node.outputs[1:] {
			if output != "" {
				requestedStatistics++
			}
		}
		if requestedStatistics > 0 {
			return nil, fmt.Errorf("node %q LayerNormalization Mean and InvStdDev outputs are unsupported", node.name)
		}
		result, err = LayerNormalization(value, scale, bias, axis, epsilon)
	case "MaxPool", "AveragePool":
		if arityErr := nodeInputArity(node, 1, 1); arityErr != nil {
			return nil, arityErr
		}
		if len(node.outputs) != 1 {
			if node.opType == "MaxPool" && len(node.outputs) == 2 {
				return nil, fmt.Errorf("node %q MaxPool second Indices output is unsupported", node.name)
			}
			return nil, fmt.Errorf("node %q %s requires exactly one output", node.name, node.opType)
		}
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		kernelAttribute, present := attribute("kernel_shape")
		if !present {
			return nil, fmt.Errorf("node %q %s has no kernel_shape", node.name, node.opType)
		}
		kernelShape, kernelErr := attributeInts(kernelAttribute, "kernel_shape")
		if kernelErr != nil {
			return nil, fmt.Errorf("node %q %s: %w", node.name, node.opType, kernelErr)
		}
		options, optionsErr := poolOptionsFromNode(node)
		if optionsErr != nil {
			return nil, optionsErr
		}
		if node.opType == "MaxPool" {
			result, err = MaxPool(value, kernelShape, options)
		} else {
			result, err = AveragePool(value, kernelShape, options)
		}
	case "GlobalAveragePool":
		if arityErr := nodeInputArity(node, 1, 1); arityErr != nil {
			return nil, arityErr
		}
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		result, err = GlobalAveragePool(value)
	case "BatchNormalization":
		if len(node.outputs) != 1 {
			return nil, fmt.Errorf("node %q BatchNormalization training-mode extra outputs are unsupported; inference requires one output", node.name)
		}
		if len(node.inputs) > 5 {
			for index := 5; index < len(node.inputs); index++ {
				if node.inputs[index] != "" {
					return nil, fmt.Errorf("node %q BatchNormalization training-mode input %q is unsupported; inference requires five inputs", node.name, node.inputs[index])
				}
			}
		}
		if len(node.inputs) != 5 {
			return nil, fmt.Errorf("node %q BatchNormalization requires five inference inputs, got %d", node.name, len(node.inputs))
		}
		batchInput, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		scale, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		bias, inputErr := input(2)
		if inputErr != nil {
			return nil, inputErr
		}
		mean, inputErr := input(3)
		if inputErr != nil {
			return nil, inputErr
		}
		variance, inputErr := input(4)
		if inputErr != nil {
			return nil, inputErr
		}
		epsilon, epsilonErr := nodeFloatAttribute(node, "epsilon", 1e-5)
		if epsilonErr != nil {
			return nil, epsilonErr
		}
		result, err = BatchNormalization(batchInput, scale, bias, mean, variance, epsilon)
		if err != nil {
			return nil, fmt.Errorf("node %q BatchNormalization inputs %q, %q, %q, %q, %q: %w", node.name, nodeInputName(node, 0), nodeInputName(node, 1), nodeInputName(node, 2), nodeInputName(node, 3), nodeInputName(node, 4), err)
		}
	case "Pad":
		if arityErr := nodeInputArity(node, 1, 3); arityErr != nil {
			return nil, arityErr
		}
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		mode, modeErr := nodeStringAttribute(node, "mode", "constant")
		if modeErr != nil {
			return nil, modeErr
		}
		if mode != "constant" {
			return nil, fmt.Errorf("node %q Pad mode %q is unsupported; only constant mode is supported", node.name, mode)
		}
		var pads []int
		if len(node.inputs) > 1 && node.inputs[1] != "" {
			padsValue, padsErr := controlInput(1, "pads")
			if padsErr != nil {
				return nil, padsErr
			}
			pads, err = tensorAxes(padsValue, "pad pads")
		} else if padsAttribute, present := attribute("pads"); present {
			pads, err = attributeInts(padsAttribute, "pads")
		} else {
			return nil, fmt.Errorf("node %q Pad has no pads attribute or initializer input", node.name)
		}
		if err != nil {
			return nil, err
		}
		constantValue := float32(0)
		if len(node.inputs) > 2 && node.inputs[2] != "" {
			valueInput, valueErr := controlInput(2, "constant value")
			if valueErr != nil {
				return nil, valueErr
			}
			if valueInput.dtype != DTypeFloat32 || len(valueInput.data) != 1 {
				return nil, fmt.Errorf("node %q Pad constant value input %q has dtype %s and shape %v; want one float32 value", node.name, nodeInputName(node, 2), dtypeName(valueInput.dtype), valueInput.shape)
			}
			constantValue = valueInput.data[0]
		} else if valueAttribute, present := attribute("value"); present {
			if !valueAttribute.hasFloat {
				return nil, fmt.Errorf("node %q Pad attribute value is not a float", node.name)
			}
			constantValue = valueAttribute.floatValue
		}
		if len(node.inputs) > 3 {
			for index := 3; index < len(node.inputs); index++ {
				if node.inputs[index] != "" {
					return nil, fmt.Errorf("node %q Pad input %d %q is unsupported", node.name, index, node.inputs[index])
				}
			}
		}
		result, err = Pad(value, pads, constantValue)
	case "ReduceMean":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		var axes []int
		if len(node.inputs) > 1 && node.inputs[1] != "" {
			axesValue, axesErr := controlInput(1, "axes")
			if axesErr != nil {
				return nil, axesErr
			}
			axes, err = tensorAxes(axesValue, "reduce mean axes")
		} else if axisAttribute, present := attribute("axes"); present {
			axes, err = attributeInts(axisAttribute, "axes")
		}
		if err != nil {
			return nil, err
		}
		keepdims := true
		if keepdimsAttribute, present := attribute("keepdims"); present {
			if !keepdimsAttribute.hasInt {
				return nil, fmt.Errorf("attribute keepdims is not an integer")
			}
			keepdims = keepdimsAttribute.intValue != 0
		}
		result, err = ReduceMean(value, axes, keepdims)
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
	case "Concat":
		axis, axisErr := nodeAxis(attribute, "axis", 0)
		if axisErr != nil {
			return nil, axisErr
		}
		inputs := make([]*Tensor, len(node.inputs))
		for index := range inputs {
			inputs[index], err = input(index)
			if err != nil {
				return nil, err
			}
		}
		result, err = Concat(inputs, axis)
	case "Unsqueeze":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		var axes []int
		if len(node.inputs) > 1 && node.inputs[1] != "" {
			axesValue, axesErr := input(1)
			if axesErr != nil {
				return nil, axesErr
			}
			axes, err = tensorAxes(axesValue, "unsqueeze axes")
		} else if axisAttribute, present := attribute("axes"); present {
			axes, err = attributeInts(axisAttribute, "axes")
		} else {
			return nil, fmt.Errorf("unsqueeze has no axes")
		}
		if err != nil {
			return nil, err
		}
		result, err = Unsqueeze(value, axes)
	case "Squeeze":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		var axes []int
		if len(node.inputs) > 1 && node.inputs[1] != "" {
			axesValue, axesErr := controlInput(1, "axes")
			if axesErr != nil {
				return nil, axesErr
			}
			axes, err = tensorAxes(axesValue, "squeeze axes")
		} else if axesAttribute, present := attribute("axes"); present {
			axes, err = attributeInts(axesAttribute, "axes")
		}
		if err != nil {
			return nil, err
		}
		result, err = Squeeze(value, axes)
	case "Expand":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		shapeValue, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		shape, shapeErr := tensorShapeValues(shapeValue, "expand shape")
		if shapeErr != nil {
			return nil, shapeErr
		}
		result, err = Expand(value, shape)
	case "Shape":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		bounds := make([]int, 0, 2)
		if startValue, present := attribute("start"); present {
			if !startValue.hasInt || startValue.intValue < minIntValue() || startValue.intValue > int64(maxInt()) {
				return nil, fmt.Errorf("attribute start is not a valid integer")
			}
			bounds = append(bounds, int(startValue.intValue))
		}
		if endValue, present := attribute("end"); present {
			if !endValue.hasInt || endValue.intValue < minIntValue() || endValue.intValue > int64(maxInt()) {
				return nil, fmt.Errorf("attribute end is not a valid integer")
			}
			if len(bounds) == 0 {
				bounds = append(bounds, 0)
			}
			bounds = append(bounds, int(endValue.intValue))
		}
		result, err = Shape(value, bounds...)
	case "Slice":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		var starts, ends []int64
		if len(node.inputs) > 1 && node.inputs[1] != "" {
			startsValue, startsErr := controlInput(1, "starts")
			if startsErr != nil {
				return nil, startsErr
			}
			starts, startsErr = tensorInt64Values(startsValue, "slice starts")
			if startsErr != nil {
				return nil, startsErr
			}
		} else if startsAttribute, present := attribute("starts"); present {
			starts = attributeInt64Values(startsAttribute, "starts")
		} else {
			return nil, fmt.Errorf("slice has no starts")
		}
		if len(node.inputs) > 2 && node.inputs[2] != "" {
			endsValue, endsErr := controlInput(2, "ends")
			if endsErr != nil {
				return nil, endsErr
			}
			ends, endsErr = tensorInt64Values(endsValue, "slice ends")
			if endsErr != nil {
				return nil, endsErr
			}
		} else if endsAttribute, present := attribute("ends"); present {
			ends = attributeInt64Values(endsAttribute, "ends")
		} else {
			return nil, fmt.Errorf("slice has no ends")
		}
		var axes, steps []int64
		if len(node.inputs) > 3 && node.inputs[3] != "" {
			axesValue, axesErr := controlInput(3, "axes")
			if axesErr != nil {
				return nil, axesErr
			}
			axes, err = tensorInt64Values(axesValue, "slice axes")
		} else if axesAttribute, present := attribute("axes"); present {
			axes = attributeInt64Values(axesAttribute, "axes")
		}
		if len(node.inputs) > 4 && node.inputs[4] != "" {
			stepsValue, stepsErr := controlInput(4, "steps")
			if stepsErr != nil {
				return nil, stepsErr
			}
			steps, err = tensorInt64Values(stepsValue, "slice steps")
		} else if stepsAttribute, present := attribute("steps"); present {
			steps = attributeInt64Values(stepsAttribute, "steps")
		}
		if err != nil {
			return nil, err
		}
		result, err = Slice(value, starts, ends, axes, steps)
	case "Split":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		var splits []int
		if len(node.inputs) > 1 && node.inputs[1] != "" {
			splitsValue, splitsErr := controlInput(1, "split")
			if splitsErr != nil {
				return nil, splitsErr
			}
			splits, err = tensorShapeValues(splitsValue, "split sizes")
		} else if splitAttribute, present := attribute("split"); present {
			splits, err = attributeInts(splitAttribute, "split")
		}
		if err != nil {
			return nil, err
		}
		axis, axisErr := nodeAxis(attribute, "axis", 0)
		if axisErr != nil {
			return nil, axisErr
		}
		var splitOutputs []*Tensor
		splitOutputs, err = Split(value, splits, axis, len(node.outputs))
		if err == nil {
			for index, splitOutput := range splitOutputs {
				produced[node.outputs[index]] = splitOutput
			}
		}
	case "Gather":
		data, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		indices, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		axis, axisErr := nodeAxis(attribute, "axis", 0)
		if axisErr != nil {
			return nil, axisErr
		}
		result, err = Gather(data, indices, axis)
	case "GreaterOrEqual":
		left, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		right, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		result, err = GreaterOrEqual(left, right)
	case "Equal", "Greater":
		left, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		right, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		if node.opType == "Equal" {
			result, err = Equal(left, right)
		} else {
			result, err = Greater(left, right)
		}
	case "Where":
		condition, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		left, inputErr := input(1)
		if inputErr != nil {
			return nil, inputErr
		}
		right, inputErr := input(2)
		if inputErr != nil {
			return nil, inputErr
		}
		result, err = Where(condition, left, right)
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
	case "ConstantOfShape":
		shapeValue, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		var value *Tensor
		if attributeValue, present := attribute("value"); present {
			if attributeValue.tensor == nil {
				return nil, fmt.Errorf("constant of shape value attribute has no tensor")
			}
			value, err = tensorProtoToTensor(*attributeValue.tensor)
			if err != nil {
				return nil, err
			}
		}
		result, err = ConstantOfShape(shapeValue, value)
	case "ai.onnx.ml:LinearRegressor":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		result, err = linearRegressor(value, node.attributes)
	case "ai.onnx.ml:LinearClassifier":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		var label, probabilities *Tensor
		label, probabilities, err = linearClassifier(value, node.attributes)
		if err == nil {
			if len(node.outputs) != 2 {
				err = fmt.Errorf("linear classifier has %d outputs, want 2", len(node.outputs))
			} else {
				produced[node.outputs[0]] = label
				produced[node.outputs[1]] = probabilities
			}
		}
	case "ai.onnx.ml:TreeEnsembleRegressor":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		result, err = treeEnsembleRegressor(value, node.attributes)
	case "ai.onnx.ml:TreeEnsembleClassifier":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		var label, probabilities *Tensor
		label, probabilities, err = treeEnsembleClassifier(value, node.attributes)
		if err == nil {
			if len(node.outputs) != 2 {
				err = fmt.Errorf("tree ensemble classifier has %d outputs, want 2", len(node.outputs))
			} else {
				produced[node.outputs[0]] = label
				produced[node.outputs[1]] = probabilities
			}
		}
	case "ai.onnx.ml:Scaler":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		result, err = scaler(value, node.attributes)
	case "ai.onnx.ml:OneHotEncoder":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		result, err = oneHotEncoder(value, node.attributes)
	case "ai.onnx.ml:LabelEncoder":
		value, inputErr := input(0)
		if inputErr != nil {
			return nil, inputErr
		}
		result, err = labelEncoder(value, node.attributes)
	default:
		return nil, fmt.Errorf("unsupported operator %s", operatorDisplayName(node.domain, node.opType))
	}
	if err != nil {
		return nil, err
	}
	if result != nil {
		produced[outputName] = result
	}
	if len(produced) == 0 {
		return nil, fmt.Errorf("operator %s produced no outputs", operatorDisplayName(node.domain, node.opType))
	}
	return produced, nil
}

func nodeAxis(attribute func(string) (protoAttribute, bool), name string, fallback int) (int, error) {
	value, present := attribute(name)
	if !present {
		return fallback, nil
	}
	if !value.hasInt || value.intValue < minIntValue() || value.intValue > int64(maxInt()) {
		return 0, fmt.Errorf("attribute %s is not a valid integer", name)
	}
	return int(value.intValue), nil
}

func nodeInputName(node modelNode, index int) string {
	if index < 0 || index >= len(node.inputs) || node.inputs[index] == "" {
		return fmt.Sprintf("input[%d]", index)
	}
	return node.inputs[index]
}

func nodeInputArity(node modelNode, minimum, maximum int) error {
	if len(node.inputs) < minimum || len(node.inputs) > maximum {
		return fmt.Errorf("node %q %s has %d inputs, want %d to %d", node.name, operatorDisplayName(node.domain, node.opType), len(node.inputs), minimum, maximum)
	}
	return nil
}

func nodeStringAttribute(node modelNode, name, fallback string) (string, error) {
	value, present := node.attributes[name]
	if !present {
		return fallback, nil
	}
	if len(value.string) == 0 {
		return "", fmt.Errorf("node %q attribute %s is not a string", node.name, name)
	}
	return string(value.string), nil
}

func nodeFloatAttribute(node modelNode, name string, fallback float32) (float32, error) {
	value, present := node.attributes[name]
	if !present {
		return fallback, nil
	}
	if !value.hasFloat {
		return 0, fmt.Errorf("node %q attribute %s is not a float", node.name, name)
	}
	return value.floatValue, nil
}

func convOptionsFromNode(node modelNode) (ConvOptions, error) {
	options := ConvOptions{Group: 1}
	if value, present := node.attributes["pads"]; present {
		pads, err := attributeInts(value, "pads")
		if err != nil {
			return ConvOptions{}, fmt.Errorf("node %q Conv: %w", node.name, err)
		}
		options.Pads = pads
	}
	if value, present := node.attributes["strides"]; present {
		strides, err := attributeInts(value, "strides")
		if err != nil {
			return ConvOptions{}, fmt.Errorf("node %q Conv: %w", node.name, err)
		}
		options.Strides = strides
	}
	if value, present := node.attributes["dilations"]; present {
		dilations, err := attributeInts(value, "dilations")
		if err != nil {
			return ConvOptions{}, fmt.Errorf("node %q Conv: %w", node.name, err)
		}
		options.Dilations = dilations
	}
	if value, present := node.attributes["auto_pad"]; present {
		if len(value.string) == 0 {
			return ConvOptions{}, fmt.Errorf("node %q Conv attribute auto_pad is not a string", node.name)
		}
		options.AutoPad = string(value.string)
	}
	if value, present := node.attributes["group"]; present {
		if !value.hasInt || value.intValue < 1 || value.intValue > int64(maxInt()) {
			return ConvOptions{}, fmt.Errorf("node %q Conv attribute group must be a positive integer", node.name)
		}
		options.Group = int(value.intValue)
	}
	return options, nil
}

func poolOptionsFromNode(node modelNode) (PoolOptions, error) {
	options := PoolOptions{}
	if value, present := node.attributes["pads"]; present {
		pads, err := attributeInts(value, "pads")
		if err != nil {
			return PoolOptions{}, fmt.Errorf("node %q %s: %w", node.name, node.opType, err)
		}
		options.Pads = pads
	}
	if value, present := node.attributes["strides"]; present {
		strides, err := attributeInts(value, "strides")
		if err != nil {
			return PoolOptions{}, fmt.Errorf("node %q %s: %w", node.name, node.opType, err)
		}
		options.Strides = strides
	}
	if value, present := node.attributes["auto_pad"]; present {
		if len(value.string) == 0 {
			return PoolOptions{}, fmt.Errorf("node %q %s attribute auto_pad is not a string", node.name, node.opType)
		}
		options.AutoPad = string(value.string)
	}
	if value, present := node.attributes["count_include_pad"]; present {
		if !value.hasInt || (value.intValue != 0 && value.intValue != 1) {
			return PoolOptions{}, fmt.Errorf("node %q AveragePool count_include_pad value is unsupported; want 0 or 1", node.name)
		}
		options.CountIncludePad = value.intValue != 0
	}
	if value, present := node.attributes["ceil_mode"]; present {
		if !value.hasInt {
			return PoolOptions{}, fmt.Errorf("node %q %s attribute ceil_mode is not an integer", node.name, node.opType)
		}
		options.CeilMode = int(value.intValue)
		if options.CeilMode != 0 {
			return PoolOptions{}, fmt.Errorf("node %q %s ceil_mode value %d is unsupported; only 0 is supported", node.name, node.opType, options.CeilMode)
		}
	}
	if value, present := node.attributes["storage_order"]; present {
		if !value.hasInt {
			return PoolOptions{}, fmt.Errorf("node %q MaxPool attribute storage_order is not an integer", node.name)
		}
		options.StorageOrder = int(value.intValue)
		if options.StorageOrder != 0 {
			return PoolOptions{}, fmt.Errorf("node %q MaxPool storage_order value %d is unsupported; only 0 is supported", node.name, options.StorageOrder)
		}
	}
	return options, nil
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

func tensorInt64Values(input *Tensor, name string) ([]int64, error) {
	if input == nil {
		return nil, fmt.Errorf("%s tensor is nil", name)
	}
	if input.dtype != DTypeInt64 {
		return nil, fmt.Errorf("%s has dtype %s, want %s", name, dtypeName(input.dtype), dtypeName(DTypeInt64))
	}
	return append([]int64(nil), input.int64Data...), nil
}

func tensorAxes(input *Tensor, name string) ([]int, error) {
	values, err := tensorInt64Values(input, name)
	if err != nil {
		return nil, err
	}
	result := make([]int, len(values))
	for index, value := range values {
		if value < minIntValue() || value > int64(maxInt()) {
			return nil, fmt.Errorf("%s value %d does not fit in an int", name, value)
		}
		result[index] = int(value)
	}
	return result, nil
}

func tensorShapeValues(input *Tensor, name string) ([]int, error) {
	values, err := tensorInt64Values(input, name)
	if err != nil {
		return nil, err
	}
	result := make([]int, len(values))
	for index, value := range values {
		if value < 0 || value > int64(maxInt()) {
			return nil, fmt.Errorf("%s value %d is not a non-negative int", name, value)
		}
		result[index] = int(value)
	}
	return result, nil
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

func attributeInt64Values(attribute protoAttribute, _ string) []int64 {
	return append([]int64(nil), attribute.ints...)
}

func minIntValue() int64 {
	return -int64(maxInt()) - 1
}
