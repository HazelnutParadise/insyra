package nn

import (
	"fmt"
	"math"
)

// Div performs float32 division with numpy-style broadcasting.
func Div(a, b *Tensor) (*Tensor, error) {
	if a != nil && b != nil && a.dtype == DTypeInt64 && b.dtype == DTypeInt64 {
		return tensorBroadcastInt64Binary(a, b, "div", func(left, right int64) (int64, error) {
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		})
	}
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

// Squeeze removes dimensions of size one. When axes is empty, all singleton
// dimensions are removed, matching the ONNX optional-axes form.
func Squeeze(input *Tensor, axes []int) (*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("squeeze input is nil")
	}
	axisSet := make(map[int]struct{})
	if len(axes) == 0 {
		for axis, dimension := range input.shape {
			if dimension == 1 {
				axisSet[axis] = struct{}{}
			}
		}
	} else {
		for _, rawAxis := range axes {
			axis, err := normalizeAxis(rawAxis, len(input.shape), "squeeze")
			if err != nil {
				return nil, err
			}
			if _, exists := axisSet[axis]; exists {
				return nil, fmt.Errorf("squeeze axis %d is repeated", rawAxis)
			}
			if input.shape[axis] != 1 {
				return nil, fmt.Errorf("squeeze axis %d has dimension %d, want 1", rawAxis, input.shape[axis])
			}
			axisSet[axis] = struct{}{}
		}
	}
	shape := make([]int, 0, len(input.shape))
	for axis, dimension := range input.shape {
		if _, remove := axisSet[axis]; !remove {
			shape = append(shape, dimension)
		}
	}
	result, err := copyTensor(input)
	if err != nil {
		return nil, err
	}
	_, strides, _, err := makeLayout(shape)
	if err != nil {
		return nil, err
	}
	result.shape, result.strides = shape, strides
	return result, nil
}

// Expand broadcasts a tensor to targetShape without changing its dtype.
func Expand(input *Tensor, targetShape []int) (*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("expand input is nil")
	}
	if !supportedTensorDType(input.dtype) {
		return nil, unsupportedDTypeError(input.dtype)
	}
	shape, err := tensorBroadcastShape(input.shape, targetShape)
	if err != nil || !sameShape(shape, targetShape) {
		if err == nil {
			err = fmt.Errorf("target shape %v is not the broadcast result", targetShape)
		}
		return nil, fmt.Errorf("expand cannot broadcast shape %v to %v: %w", input.shape, targetShape, err)
	}
	result, err := newTypedTensor(input.dtype, targetShape)
	if err != nil {
		return nil, err
	}
	inputStrides := alignedBroadcastStrides(input, len(targetShape))
	for outputIndex := 0; outputIndex < result.Len(); outputIndex++ {
		inputIndex, _ := broadcastIndex(outputIndex, result.strides, inputStrides, inputStrides)
		copyTensorElement(result, outputIndex, input, inputIndex)
	}
	return result, nil
}

// Shape returns the selected portion of input's shape as an int64 tensor.
// With no bounds it returns every dimension. Bounds follow ONNX's start/end
// convention and may be negative.
func Shape(input *Tensor, bounds ...int) (*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("shape input is nil")
	}
	if len(bounds) > 2 {
		return nil, fmt.Errorf("shape accepts at most start and end bounds")
	}
	start, end := 0, len(input.shape)
	if len(bounds) >= 1 {
		start = bounds[0]
	}
	if len(bounds) == 2 {
		end = bounds[1]
	}
	start = normalizeShapeBound(start, len(input.shape), 0)
	end = normalizeShapeBound(end, len(input.shape), 0)
	if end < start {
		end = start
	}
	values := make([]int64, end-start)
	for index := range values {
		values[index] = int64(input.shape[start+index])
	}
	return newInt64Tensor([]int{len(values)}, values)
}

// ConstantOfShape fills a tensor with one scalar value. ONNX defaults the
// value to a float32 zero when the optional value attribute is absent.
func ConstantOfShape(shapeTensor, value *Tensor) (*Tensor, error) {
	if shapeTensor == nil {
		return nil, fmt.Errorf("constant of shape input is nil")
	}
	if shapeTensor.dtype != DTypeInt64 {
		return nil, fmt.Errorf("constant of shape input has dtype %s, want %s", dtypeName(shapeTensor.dtype), dtypeName(DTypeInt64))
	}
	shape := make([]int, len(shapeTensor.int64Data))
	for index, dimension := range shapeTensor.int64Data {
		if dimension < 0 || dimension > int64(maxInt()) {
			return nil, fmt.Errorf("constant of shape dimension %d is not a non-negative int", dimension)
		}
		shape[index] = int(dimension)
	}
	if value == nil {
		defaultValue, err := newFloat32Tensor(nil, []float32{0})
		if err != nil {
			return nil, err
		}
		value = defaultValue
	}
	if !supportedTensorDType(value.dtype) || value.Len() != 1 {
		return nil, fmt.Errorf("constant of shape value has dtype %s and shape %v, want one supported scalar", dtypeName(value.dtype), value.shape)
	}
	result, err := newTypedTensor(value.dtype, shape)
	if err != nil {
		return nil, err
	}
	for index := 0; index < result.Len(); index++ {
		copyTensorElement(result, index, value, 0)
	}
	return result, nil
}

func normalizeShapeBound(bound, rank, fallback int) int {
	if bound < 0 {
		bound += rank
	}
	if bound < 0 {
		return fallback
	}
	if bound > rank {
		return rank
	}
	return bound
}

// Slice applies ONNX starts, ends, axes, and steps to an N-dimensional tensor.
func Slice(input *Tensor, starts, ends, axes, steps []int64) (*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("slice input is nil")
	}
	if !supportedTensorDType(input.dtype) {
		return nil, unsupportedDTypeError(input.dtype)
	}
	if len(starts) != len(ends) {
		return nil, fmt.Errorf("slice starts length %d does not match ends length %d", len(starts), len(ends))
	}
	if len(axes) == 0 {
		axes = make([]int64, len(starts))
		for index := range axes {
			axes[index] = int64(index)
		}
	}
	if len(axes) != len(starts) {
		return nil, fmt.Errorf("slice axes length %d does not match starts length %d", len(axes), len(starts))
	}
	if len(steps) == 0 {
		steps = make([]int64, len(starts))
		for index := range steps {
			steps[index] = 1
		}
	}
	if len(steps) != len(starts) {
		return nil, fmt.Errorf("slice steps length %d does not match starts length %d", len(steps), len(starts))
	}
	indices := make([][]int, len(input.shape))
	for axis, dimension := range input.shape {
		indices[axis] = make([]int, dimension)
		for index := range indices[axis] {
			indices[axis][index] = index
		}
	}
	seen := make(map[int]struct{}, len(axes))
	for index, rawAxis := range axes {
		if rawAxis < int64(minIntValue()) || rawAxis > int64(maxInt()) {
			return nil, fmt.Errorf("slice axis %d does not fit in an int", rawAxis)
		}
		axis, err := normalizeAxis(int(rawAxis), len(input.shape), "slice")
		if err != nil {
			return nil, err
		}
		if _, exists := seen[axis]; exists {
			return nil, fmt.Errorf("slice axis %d is repeated", rawAxis)
		}
		seen[axis] = struct{}{}
		if steps[index] == 0 {
			return nil, fmt.Errorf("slice step for axis %d is zero", axis)
		}
		indices[axis] = sliceIndices(input.shape[axis], starts[index], ends[index], steps[index])
	}
	outShape := make([]int, len(input.shape))
	for axis := range input.shape {
		outShape[axis] = len(indices[axis])
	}
	result, err := newTypedTensor(input.dtype, outShape)
	if err != nil {
		return nil, err
	}
	for outputIndex := 0; outputIndex < result.Len(); outputIndex++ {
		coordinates := linearCoordinates(outputIndex, result.shape, result.strides)
		inputIndex := 0
		for axis, coordinate := range coordinates {
			inputIndex += indices[axis][coordinate] * input.strides[axis]
		}
		copyTensorElement(result, outputIndex, input, inputIndex)
	}
	return result, nil
}

func sliceIndices(length int, start, end, step int64) []int {
	if length == 0 {
		return nil
	}
	result := make([]int, 0)
	if step > 0 {
		start = normalizeSliceStart(start, length, false)
		end = normalizeSliceEnd(end, length, false)
		for index := start; index < end; index += step {
			result = append(result, int(index))
		}
		return result
	}
	start = normalizeSliceStart(start, length, true)
	end = normalizeSliceEnd(end, length, true)
	for index := start; index > end; index += step {
		result = append(result, int(index))
	}
	return result
}

func normalizeSliceStart(value int64, length int, reverse bool) int64 {
	limit := int64(length)
	if value < 0 && value >= -limit {
		value += limit
	}
	lower, upper := int64(0), limit
	if reverse {
		upper--
	}
	if upper < lower {
		return lower
	}
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}

func normalizeSliceEnd(value int64, length int, reverse bool) int64 {
	limit := int64(length)
	if value < 0 && value >= -limit {
		value += limit
	}
	lower, upper := int64(0), limit
	if reverse {
		lower = -1
		upper--
	}
	if upper < lower {
		return lower
	}
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}

// Split partitions input along axis. If splits is nil, outputCount must name
// the number of equal partitions.
func Split(input *Tensor, splits []int, axis int, outputCount ...int) ([]*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("split input is nil")
	}
	if !supportedTensorDType(input.dtype) {
		return nil, unsupportedDTypeError(input.dtype)
	}
	if len(outputCount) > 1 {
		return nil, fmt.Errorf("split accepts at most one output count")
	}
	axis, err := normalizeAxis(axis, len(input.shape), "split")
	if err != nil {
		return nil, err
	}
	if len(splits) == 0 {
		if len(outputCount) != 1 || outputCount[0] <= 0 {
			return nil, fmt.Errorf("split needs output count when split sizes are omitted")
		}
		if input.shape[axis]%outputCount[0] != 0 {
			return nil, fmt.Errorf("split axis %d with size %d is not divisible by %d outputs", axis, input.shape[axis], outputCount[0])
		}
		size := input.shape[axis] / outputCount[0]
		splits = make([]int, outputCount[0])
		for index := range splits {
			splits[index] = size
		}
	} else if len(outputCount) == 1 && outputCount[0] != len(splits) {
		return nil, fmt.Errorf("split has %d sizes but %d outputs", len(splits), outputCount[0])
	}
	total := 0
	for index, size := range splits {
		if size < 0 {
			return nil, fmt.Errorf("split size %d at index %d is negative", size, index)
		}
		if size > maxInt()-total {
			return nil, fmt.Errorf("split sizes overflow element count")
		}
		total += size
	}
	if total != input.shape[axis] {
		return nil, fmt.Errorf("split sizes %v sum to %d, want axis %d size %d", splits, total, axis, input.shape[axis])
	}
	outputs := make([]*Tensor, len(splits))
	offset := 0
	for outputIndex, size := range splits {
		shape := append([]int(nil), input.shape...)
		shape[axis] = size
		output, outputErr := newTypedTensor(input.dtype, shape)
		if outputErr != nil {
			return nil, outputErr
		}
		for index := 0; index < output.Len(); index++ {
			coordinates := linearCoordinates(index, output.shape, output.strides)
			coordinates[axis] += offset
			inputIndex := 0
			for dimension, coordinate := range coordinates {
				inputIndex += coordinate * input.strides[dimension]
			}
			copyTensorElement(output, index, input, inputIndex)
		}
		outputs[outputIndex] = output
		offset += size
	}
	return outputs, nil
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

// Equal compares matching tensors with numpy-style broadcasting.
func Equal(left, right *Tensor) (*Tensor, error) {
	return compareBinary(left, right, "equal", func(a, b comparisonValue) bool { return a.equal(b) })
}

// Greater compares matching numeric tensors with numpy-style broadcasting.
func Greater(left, right *Tensor) (*Tensor, error) {
	if left == nil || right == nil {
		return nil, fmt.Errorf("greater operands must not be nil")
	}
	if left.dtype != DTypeFloat32 && left.dtype != DTypeInt64 {
		return nil, fmt.Errorf("greater supports matching float32 or int64 operands, got %s and %s", dtypeName(left.dtype), dtypeName(right.dtype))
	}
	return compareBinary(left, right, "greater", func(a, b comparisonValue) bool { return a.greater(b) })
}

type comparisonValue struct {
	float  float32
	int64  int64
	bool   bool
	string string
	dtype  DType
}

func (value comparisonValue) equal(other comparisonValue) bool {
	switch value.dtype {
	case DTypeFloat32:
		return value.float == other.float
	case DTypeInt64:
		return value.int64 == other.int64
	case DTypeBool:
		return value.bool == other.bool
	case DTypeString:
		return value.string == other.string
	default:
		return false
	}
}

func (value comparisonValue) greater(other comparisonValue) bool {
	switch value.dtype {
	case DTypeFloat32:
		return value.float > other.float
	case DTypeInt64:
		return value.int64 > other.int64
	default:
		return false
	}
}

func compareBinary(left, right *Tensor, operation string, compare func(comparisonValue, comparisonValue) bool) (*Tensor, error) {
	if left == nil || right == nil {
		return nil, fmt.Errorf("%s operands must not be nil", operation)
	}
	if left.dtype != right.dtype || !supportedTensorDType(left.dtype) {
		return nil, fmt.Errorf("%s supports matching implemented dtypes, got %s and %s", operation, dtypeName(left.dtype), dtypeName(right.dtype))
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
		result.boolData[outputIndex] = compare(tensorComparisonValue(left, leftIndex), tensorComparisonValue(right, rightIndex))
	}
	return result, nil
}

func tensorComparisonValue(tensor *Tensor, index int) comparisonValue {
	value := comparisonValue{dtype: tensor.dtype}
	switch tensor.dtype {
	case DTypeFloat32:
		value.float = tensor.data[index]
	case DTypeInt64:
		value.int64 = tensor.int64Data[index]
	case DTypeBool:
		value.bool = tensor.boolData[index]
	case DTypeString:
		value.string = tensor.stringData[index]
	}
	return value
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
