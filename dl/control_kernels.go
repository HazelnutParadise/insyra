package dl

import (
	"fmt"
	"math"
)

// Div performs float32 division with numpy-style broadcasting.
func Div(a, b *Tensor) (*Tensor, error) {
	return tensorBroadcastBinary(a, b, "div", func(left, right float32) float32 { return left / right })
}

// Unsqueeze inserts dimensions of size one at the supplied output axes.
func Unsqueeze(input *Tensor, axes []int) (*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("unsqueeze input is nil")
	}
	if len(axes) == 0 {
		return nil, fmt.Errorf("unsqueeze requires at least one axis")
	}
	outputRank := len(input.shape) + len(axes)
	normalized := make([]int, len(axes))
	seen := make(map[int]struct{}, len(axes))
	for index, axis := range axes {
		if axis < 0 {
			axis += outputRank
		}
		if axis < 0 || axis >= outputRank {
			return nil, fmt.Errorf("unsqueeze axis %d is out of range for output rank %d", axes[index], outputRank)
		}
		if _, exists := seen[axis]; exists {
			return nil, fmt.Errorf("unsqueeze axis %d is repeated", axis)
		}
		seen[axis] = struct{}{}
		normalized[index] = axis
	}
	axisSet := seen
	shape := make([]int, outputRank)
	inputAxis := 0
	for axis := range shape {
		if _, inserted := axisSet[axis]; inserted {
			shape[axis] = 1
			continue
		}
		shape[axis] = input.shape[inputAxis]
		inputAxis++
	}
	result, err := newTypedTensor(input.dtype, shape)
	if err != nil {
		return nil, err
	}
	for outputIndex := 0; outputIndex < result.Len(); outputIndex++ {
		remaining := outputIndex
		inputIndex := 0
		for axis, stride := range result.strides {
			coordinate := 0
			if stride != 0 {
				coordinate = remaining / stride
				remaining %= stride
			}
			if _, inserted := axisSet[axis]; !inserted {
				inputIndex += coordinate * input.strides[inputAxisForOutputAxis(axis, axisSet)]
			}
		}
		copyTensorElement(result, outputIndex, input, inputIndex)
	}
	_ = normalized
	return result, nil
}

func inputAxisForOutputAxis(outputAxis int, inserted map[int]struct{}) int {
	inputAxis := 0
	for axis := 0; axis < outputAxis; axis++ {
		if _, isInserted := inserted[axis]; !isInserted {
			inputAxis++
		}
	}
	return inputAxis
}

// Concat concatenates tensors along one axis. The participating tensors may
// carry float32, int64, bool, or string values, but must share rank and dtype.
func Concat(inputs []*Tensor, axis int) (*Tensor, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("concat requires at least one input")
	}
	first := inputs[0]
	if first == nil {
		return nil, fmt.Errorf("concat input 0 is nil")
	}
	rank := len(first.shape)
	if rank == 0 {
		return nil, fmt.Errorf("concat does not accept scalar inputs")
	}
	axis, err := normalizeAxis(axis, rank, "concat")
	if err != nil {
		return nil, err
	}
	shape := append([]int(nil), first.shape...)
	for index, input := range inputs {
		if input == nil {
			return nil, fmt.Errorf("concat input %d is nil", index)
		}
		if input.dtype != first.dtype {
			return nil, fmt.Errorf("concat input %d has dtype %s, want %s", index, dtypeName(input.dtype), dtypeName(first.dtype))
		}
		if len(input.shape) != rank {
			return nil, fmt.Errorf("concat input %d has rank %d, want %d", index, len(input.shape), rank)
		}
		for dimension := range input.shape {
			if dimension != axis && input.shape[dimension] != first.shape[dimension] {
				return nil, fmt.Errorf("concat input %d shape %v is incompatible with %v", index, input.shape, first.shape)
			}
		}
		if index > 0 {
			shape[axis] += input.shape[axis]
		}
	}
	result, err := newTypedTensor(first.dtype, shape)
	if err != nil {
		return nil, err
	}
	for outputIndex := 0; outputIndex < result.Len(); outputIndex++ {
		remaining := outputIndex
		axisCoordinate := 0
		coordinates := make([]int, rank)
		for dimension, stride := range result.strides {
			if stride != 0 {
				coordinates[dimension] = remaining / stride
				remaining %= stride
			}
		}
		for _, input := range inputs {
			if axisCoordinate+input.shape[axis] <= coordinates[axis] {
				axisCoordinate += input.shape[axis]
				continue
			}
			coordinates[axis] -= axisCoordinate
			inputIndex := 0
			for dimension, coordinate := range coordinates {
				inputIndex += coordinate * input.strides[dimension]
			}
			copyTensorElement(result, outputIndex, input, inputIndex)
			coordinates[axis] += axisCoordinate
			break
		}
	}
	return result, nil
}

// Gather selects values along axis using int64 indices. It implements the
// scalar-index form emitted by the ml pipeline exporter as well as general
// ONNX index shapes for small standalone graphs.
func Gather(data, indices *Tensor, axis int) (*Tensor, error) {
	if data == nil || indices == nil {
		return nil, fmt.Errorf("gather inputs must not be nil")
	}
	if indices.dtype != DTypeInt64 {
		return nil, fmt.Errorf("gather indices have dtype %s, want %s", dtypeName(indices.dtype), dtypeName(DTypeInt64))
	}
	if len(data.shape) == 0 {
		return nil, fmt.Errorf("gather data must not be scalar")
	}
	axis, err := normalizeAxis(axis, len(data.shape), "gather")
	if err != nil {
		return nil, err
	}
	outShape := make([]int, 0, len(data.shape)-1+len(indices.shape))
	outShape = append(outShape, data.shape[:axis]...)
	outShape = append(outShape, indices.shape...)
	outShape = append(outShape, data.shape[axis+1:]...)
	result, err := newTypedTensor(data.dtype, outShape)
	if err != nil {
		return nil, err
	}
	prefix := axis
	suffix := len(data.shape) - axis - 1
	for outputIndex := 0; outputIndex < result.Len(); outputIndex++ {
		coordinates := linearCoordinates(outputIndex, result.shape, result.strides)
		dataCoordinates := make([]int, len(data.shape))
		copy(dataCoordinates[:prefix], coordinates[:prefix])
		indexCoordinates := coordinates[prefix : prefix+len(indices.shape)]
		indexLinear := 0
		for dimension, coordinate := range indexCoordinates {
			indexLinear += coordinate * indices.strides[dimension]
		}
		selected := indices.int64Data[indexLinear]
		if selected < 0 {
			selected += int64(data.shape[axis])
		}
		if selected < 0 || selected >= int64(data.shape[axis]) {
			return nil, fmt.Errorf("gather index %d is out of range for axis %d with size %d", indices.int64Data[indexLinear], axis, data.shape[axis])
		}
		dataCoordinates[axis] = int(selected)
		copy(dataCoordinates[axis+1:], coordinates[prefix+len(indices.shape):prefix+len(indices.shape)+suffix])
		dataLinear := 0
		for dimension, coordinate := range dataCoordinates {
			dataLinear += coordinate * data.strides[dimension]
		}
		copyTensorElement(result, outputIndex, data, dataLinear)
	}
	return result, nil
}

// GreaterOrEqual compares float32 or int64 tensors with broadcasting.
func GreaterOrEqual(left, right *Tensor) (*Tensor, error) {
	if left == nil || right == nil {
		return nil, fmt.Errorf("greater-or-equal operands must not be nil")
	}
	if left.dtype != right.dtype || (left.dtype != DTypeFloat32 && left.dtype != DTypeInt64) {
		return nil, fmt.Errorf("greater-or-equal supports matching float32 or int64 operands, got %s and %s", dtypeName(left.dtype), dtypeName(right.dtype))
	}
	shape, err := tensorBroadcastShape(left.shape, right.shape)
	if err != nil {
		return nil, err
	}
	shape, strides, count, err := makeLayout(shape)
	if err != nil {
		return nil, err
	}
	result := &Tensor{dtype: DTypeBool, shape: shape, strides: strides, boolData: make([]bool, count)}
	leftStrides := alignedBroadcastStrides(left, len(shape))
	rightStrides := alignedBroadcastStrides(right, len(shape))
	for outputIndex := 0; outputIndex < count; outputIndex++ {
		leftIndex, rightIndex := broadcastIndex(outputIndex, strides, leftStrides, rightStrides)
		if left.dtype == DTypeFloat32 {
			result.boolData[outputIndex] = left.data[leftIndex] >= right.data[rightIndex]
		} else {
			result.boolData[outputIndex] = left.int64Data[leftIndex] >= right.int64Data[rightIndex]
		}
	}
	return result, nil
}

// Where selects values from left and right using a broadcast bool condition.
func Where(condition, left, right *Tensor) (*Tensor, error) {
	if condition == nil || left == nil || right == nil {
		return nil, fmt.Errorf("where inputs must not be nil")
	}
	if condition.dtype != DTypeBool {
		return nil, fmt.Errorf("where condition has dtype %s, want %s", dtypeName(condition.dtype), dtypeName(DTypeBool))
	}
	if left.dtype != right.dtype {
		return nil, fmt.Errorf("where branches have dtypes %s and %s", dtypeName(left.dtype), dtypeName(right.dtype))
	}
	shape, err := tensorBroadcastShape(left.shape, right.shape)
	if err != nil {
		return nil, err
	}
	shape, err = tensorBroadcastShape(shape, condition.shape)
	if err != nil {
		return nil, err
	}
	result, err := newTypedTensor(left.dtype, shape)
	if err != nil {
		return nil, err
	}
	leftStrides := alignedBroadcastStrides(left, len(shape))
	rightStrides := alignedBroadcastStrides(right, len(shape))
	conditionStrides := alignedBroadcastStrides(condition, len(shape))
	for outputIndex := 0; outputIndex < result.Len(); outputIndex++ {
		leftIndex, rightIndex := broadcastIndex(outputIndex, result.strides, leftStrides, rightStrides)
		conditionIndex, _ := broadcastIndex(outputIndex, result.strides, conditionStrides, conditionStrides)
		if condition.boolData[conditionIndex] {
			copyTensorElement(result, outputIndex, left, leftIndex)
		} else {
			copyTensorElement(result, outputIndex, right, rightIndex)
		}
	}
	return result, nil
}

func newTypedTensor(dtype DType, shape []int) (*Tensor, error) {
	_, _, count, err := makeLayout(shape)
	if err != nil {
		return nil, err
	}
	switch dtype {
	case DTypeFloat32:
		return newFloat32Tensor(shape, make([]float32, count))
	case DTypeInt64:
		return newInt64Tensor(shape, make([]int64, count))
	case DTypeBool:
		return newBoolTensor(shape, make([]bool, count))
	case DTypeString:
		return newStringTensor(shape, make([]string, count))
	default:
		return nil, unsupportedDTypeError(dtype)
	}
}

func linearCoordinates(index int, shape, strides []int) []int {
	coordinates := make([]int, len(shape))
	remaining := index
	for dimension, stride := range strides {
		if stride != 0 {
			coordinates[dimension] = remaining / stride
			remaining %= stride
		}
	}
	return coordinates
}

func broadcastIndex(outputIndex int, outputStrides, leftStrides, rightStrides []int) (int, int) {
	remaining := outputIndex
	leftIndex, rightIndex := 0, 0
	for axis, stride := range outputStrides {
		coordinate := 0
		if stride != 0 {
			coordinate = remaining / stride
			remaining %= stride
		}
		leftIndex += coordinate * leftStrides[axis]
		rightIndex += coordinate * rightStrides[axis]
	}
	return leftIndex, rightIndex
}

func copyTensorElement(destination *Tensor, destinationIndex int, source *Tensor, sourceIndex int) {
	switch destination.dtype {
	case DTypeFloat32:
		destination.data[destinationIndex] = source.data[sourceIndex]
	case DTypeInt64:
		destination.int64Data[destinationIndex] = source.int64Data[sourceIndex]
	case DTypeBool:
		destination.boolData[destinationIndex] = source.boolData[sourceIndex]
	case DTypeString:
		destination.stringData[destinationIndex] = source.stringData[sourceIndex]
	}
}

func sigmoidScalar(value float32) float32 {
	if value >= 0 {
		return float32(1 / (1 + math.Exp(-float64(value))))
	}
	expValue := math.Exp(float64(value))
	return float32(expValue / (1 + expValue))
}
