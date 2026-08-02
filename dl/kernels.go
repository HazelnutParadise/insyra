package dl

import (
	"fmt"
	"math"
	"strconv"
)

// GemmOptions controls the optional attributes of Gemm. When no options are
// passed, ONNX defaults alpha=1 and beta=1 are used.
type GemmOptions struct {
	Alpha  float32
	Beta   float32
	TransA bool
	TransB bool
}

// Gemm computes alpha*A*B + beta*C. A and B must be two-dimensional and C,
// when present, follows numpy-style broadcasting to the output shape.
func Gemm(a, b, c *Tensor, options ...GemmOptions) (*Tensor, error) {
	if err := requireFloat32(a, "gemm input A"); err != nil {
		return nil, err
	}
	if err := requireFloat32(b, "gemm input B"); err != nil {
		return nil, err
	}
	if c != nil {
		if err := requireFloat32(c, "gemm input C"); err != nil {
			return nil, err
		}
	}
	if len(a.shape) != 2 || len(b.shape) != 2 {
		return nil, fmt.Errorf("gemm requires 2-D inputs, got shapes %v and %v", a.shape, b.shape)
	}
	if len(options) > 1 {
		return nil, fmt.Errorf("gemm accepts at most one options value")
	}
	opts := GemmOptions{Alpha: 1, Beta: 1}
	if len(options) == 1 {
		opts = options[0]
	}

	aRows, aCols := a.shape[0], a.shape[1]
	if opts.TransA {
		aRows, aCols = a.shape[1], a.shape[0]
	}
	bRows, bCols := b.shape[0], b.shape[1]
	if opts.TransB {
		bRows, bCols = b.shape[1], b.shape[0]
	}
	if aCols != bRows {
		return nil, fmt.Errorf("gemm shapes %v and %v are incompatible after transpose", a.shape, b.shape)
	}
	result, err := newFloat32Tensor([]int{aRows, bCols}, make([]float32, aRows*bCols))
	if err != nil {
		return nil, err
	}
	for row := 0; row < aRows; row++ {
		for column := 0; column < bCols; column++ {
			var sum float32
			for inner := 0; inner < aCols; inner++ {
				aIndex := row*a.shape[1] + inner
				if opts.TransA {
					aIndex = inner*a.shape[1] + row
				}
				bIndex := inner*b.shape[1] + column
				if opts.TransB {
					bIndex = column*b.shape[1] + inner
				}
				sum += a.data[aIndex] * b.data[bIndex]
			}
			result.data[row*bCols+column] = opts.Alpha * sum
		}
	}
	if c == nil {
		return result, nil
	}
	return tensorBroadcastBinary(result, c, "gemm", func(value, bias float32) float32 {
		return value + opts.Beta*bias
	})
}

// MatMul computes a matrix product with numpy-style broadcasting over leading
// batch dimensions. The two-dimensional case stays on the original tight
// loop because it is the common path for Gemm-style graphs.
func MatMul(a, b *Tensor) (*Tensor, error) {
	if err := requireFloat32(a, "matmul input A"); err != nil {
		return nil, err
	}
	if err := requireFloat32(b, "matmul input B"); err != nil {
		return nil, err
	}
	if len(a.shape) == 0 || len(b.shape) == 0 {
		return nil, fmt.Errorf("matmul requires inputs with rank at least 1, got shapes %v and %v", a.shape, b.shape)
	}
	if len(a.shape) == 2 && len(b.shape) == 2 {
		return matMul2D(a, b)
	}

	aRows, aCols := 1, a.shape[len(a.shape)-1]
	aRowStride, aColStride := 0, a.strides[len(a.shape)-1]
	aBatchShape := a.shape[:0]
	aBatchStrides := a.strides[:0]
	if len(a.shape) >= 2 {
		aRows = a.shape[len(a.shape)-2]
		aRowStride = a.strides[len(a.shape)-2]
		aBatchShape = a.shape[:len(a.shape)-2]
		aBatchStrides = a.strides[:len(a.shape)-2]
	}
	bRows, bCols := b.shape[len(b.shape)-1], 1
	bRowStride, bColStride := b.strides[len(b.shape)-1], 0
	bBatchShape := b.shape[:0]
	bBatchStrides := b.strides[:0]
	if len(b.shape) >= 2 {
		bRows = b.shape[len(b.shape)-2]
		bCols = b.shape[len(b.shape)-1]
		bColStride = b.strides[len(b.shape)-1]
		bRowStride = b.strides[len(b.shape)-2]
		bBatchShape = b.shape[:len(b.shape)-2]
		bBatchStrides = b.strides[:len(b.shape)-2]
	}
	if aCols != bRows {
		return nil, fmt.Errorf("matmul shapes %v and %v are incompatible", a.shape, b.shape)
	}
	batchShape, err := tensorBroadcastShape(aBatchShape, bBatchShape)
	if err != nil {
		return nil, fmt.Errorf("matmul batch shapes %v and %v are incompatible for input shapes %v and %v: %w", aBatchShape, bBatchShape, a.shape, b.shape, err)
	}
	logicalShape := append(append([]int(nil), batchShape...), aRows, bCols)
	result, err := newZeroFloat32Tensor(logicalShape)
	if err != nil {
		return nil, err
	}
	aBatchAligned := alignedShapeStrides(aBatchShape, aBatchStrides, len(batchShape))
	bBatchAligned := alignedShapeStrides(bBatchShape, bBatchStrides, len(batchShape))
	_, _, batchCount, err := makeLayout(batchShape)
	if err != nil {
		return nil, err
	}
	batchStrides := stridesForShape(batchShape)
	resultMatrixSize, err := checkedProduct(aRows, bCols, "matmul result")
	if err != nil {
		return nil, err
	}
	for batchIndex := 0; batchIndex < batchCount; batchIndex++ {
		aBase, bBase := 0, 0
		remaining := batchIndex
		for axis, stride := range batchStrides {
			coordinate := 0
			if stride != 0 {
				coordinate = remaining / stride
				remaining %= stride
			}
			aBase += coordinate * aBatchAligned[axis]
			bBase += coordinate * bBatchAligned[axis]
		}
		resultBase := batchIndex * resultMatrixSize
		for row := 0; row < aRows; row++ {
			for column := 0; column < bCols; column++ {
				var sum float32
				for inner := 0; inner < aCols; inner++ {
					sum += a.data[aBase+row*aRowStride+inner*aColStride] * b.data[bBase+inner*bRowStride+column*bColStride]
				}
				result.data[resultBase+row*bCols+column] = sum
			}
		}
	}
	if len(a.shape) == 1 {
		result.shape = append([]int(nil), batchShape...)
		if len(b.shape) >= 2 {
			result.shape = append(result.shape, bCols)
		}
		result.strides = stridesForShape(result.shape)
	}
	if len(b.shape) == 1 {
		result.shape = append([]int(nil), batchShape...)
		if len(a.shape) >= 2 {
			result.shape = append(result.shape, aRows)
		}
		result.strides = stridesForShape(result.shape)
	}
	if len(a.shape) == 1 && len(b.shape) == 1 {
		result.shape = nil
		result.strides = nil
	}
	return result, nil
}

func matMul2D(a, b *Tensor) (*Tensor, error) {
	if a.shape[1] != b.shape[0] {
		return nil, fmt.Errorf("matmul shapes %v and %v are incompatible", a.shape, b.shape)
	}
	result, err := newFloat32Tensor([]int{a.shape[0], b.shape[1]}, make([]float32, a.shape[0]*b.shape[1]))
	if err != nil {
		return nil, err
	}
	for row := 0; row < a.shape[0]; row++ {
		for column := 0; column < b.shape[1]; column++ {
			var sum float32
			for inner := 0; inner < a.shape[1]; inner++ {
				sum += a.data[row*a.shape[1]+inner] * b.data[inner*b.shape[1]+column]
			}
			result.data[row*b.shape[1]+column] = sum
		}
	}
	return result, nil
}

// Add performs float32 addition with numpy-style broadcasting.
func Add(a, b *Tensor) (*Tensor, error) {
	return tensorBroadcastBinary(a, b, "add", func(left, right float32) float32 { return left + right })
}

// Sub performs float32 subtraction with numpy-style broadcasting.
func Sub(a, b *Tensor) (*Tensor, error) {
	return tensorBroadcastBinary(a, b, "sub", func(left, right float32) float32 { return left - right })
}

// Mul performs float32 multiplication with numpy-style broadcasting.
func Mul(a, b *Tensor) (*Tensor, error) {
	return tensorBroadcastBinary(a, b, "mul", func(left, right float32) float32 { return left * right })
}

// Relu computes max(input, 0) element by element.
func Relu(input *Tensor) (*Tensor, error) {
	return unary("relu", input, func(value float32) float32 {
		if value < 0 {
			return 0
		}
		return value
	})
}

// Sigmoid computes 1/(1+exp(-input)) element by element.
func Sigmoid(input *Tensor) (*Tensor, error) {
	return unary("sigmoid", input, func(value float32) float32 {
		if value >= 0 {
			return float32(1 / (1 + math.Exp(-float64(value))))
		}
		expValue := math.Exp(float64(value))
		return float32(expValue / (1 + expValue))
	})
}

// Tanh computes the hyperbolic tangent element by element.
func Tanh(input *Tensor) (*Tensor, error) {
	return unary("tanh", input, func(value float32) float32 { return float32(math.Tanh(float64(value))) })
}

// Erf computes the Gauss error function element by element.
func Erf(input *Tensor) (*Tensor, error) {
	return unary("erf", input, func(value float32) float32 { return float32(math.Erf(float64(value))) })
}

// Sqrt computes the non-negative square root element by element.
func Sqrt(input *Tensor) (*Tensor, error) {
	return unary("sqrt", input, func(value float32) float32 { return float32(math.Sqrt(float64(value))) })
}

// Pow raises the left tensor to the right tensor element by element with
// numpy-style broadcasting.
func Pow(left, right *Tensor) (*Tensor, error) {
	return tensorBroadcastBinary(left, right, "pow", func(a, b float32) float32 {
		return float32(math.Pow(float64(a), float64(b)))
	})
}

// Gelu computes the Gaussian error linear unit. ONNX's default exact form is
// used unless approximate is "tanh".
func Gelu(input *Tensor, approximate ...string) (*Tensor, error) {
	if len(approximate) > 1 {
		return nil, fmt.Errorf("gelu accepts at most one approximate mode")
	}
	mode := "none"
	if len(approximate) == 1 {
		mode = approximate[0]
	}
	switch mode {
	case "none":
		return unary("gelu", input, func(value float32) float32 {
			x := float64(value)
			return float32(0.5 * x * (1 + math.Erf(x/math.Sqrt2)))
		})
	case "tanh":
		return unary("gelu", input, func(value float32) float32 {
			x := float64(value)
			return float32(0.5 * x * (1 + math.Tanh(math.Sqrt(2/math.Pi)*(x+0.044715*x*x*x))))
		})
	default:
		return nil, fmt.Errorf("gelu approximate mode %q is unsupported", mode)
	}
}

// LayerNormalization normalizes the suffix of input beginning at axis and
// applies scale and bias over that suffix.
func LayerNormalization(input, scale, bias *Tensor, axis int, epsilon float32) (*Tensor, error) {
	if err := requireFloat32(input, "layer normalization input"); err != nil {
		return nil, err
	}
	if err := requireFloat32(scale, "layer normalization scale"); err != nil {
		return nil, err
	}
	if err := requireFloat32(bias, "layer normalization bias"); err != nil {
		return nil, err
	}
	axis, err := normalizeAxis(axis, len(input.shape), "layer normalization")
	if err != nil {
		return nil, err
	}
	if epsilon < 0 || math.IsNaN(float64(epsilon)) {
		return nil, fmt.Errorf("layer normalization epsilon %g is invalid", epsilon)
	}
	normalizedShape := input.shape[axis:]
	if !sameShape(scale.shape, normalizedShape) || !sameShape(bias.shape, normalizedShape) {
		return nil, fmt.Errorf("layer normalization scale and bias shapes %v and %v must match normalized shape %v", scale.shape, bias.shape, normalizedShape)
	}
	result, err := copyTensor(input)
	if err != nil {
		return nil, err
	}
	groupSize := 1
	for _, dimension := range normalizedShape {
		groupSize *= dimension
	}
	groups := 1
	for _, dimension := range input.shape[:axis] {
		groups *= dimension
	}
	for group := 0; group < groups; group++ {
		base := group * groupSize
		var meanValue float64
		for index := 0; index < groupSize; index++ {
			meanValue += float64(input.data[base+index])
		}
		meanValue /= float64(groupSize)
		var variance float64
		for index := 0; index < groupSize; index++ {
			delta := float64(input.data[base+index]) - meanValue
			variance += delta * delta
		}
		variance /= float64(groupSize)
		denominator := math.Sqrt(variance + float64(epsilon))
		for index := 0; index < groupSize; index++ {
			value := (float64(input.data[base+index]) - meanValue) / denominator
			result.data[base+index] = float32(value*float64(scale.data[index]) + float64(bias.data[index]))
		}
	}
	return result, nil
}

// ReduceMean averages input over axes. A nil or empty axes list reduces all
// dimensions, matching ONNX when the axes input is omitted.
func ReduceMean(input *Tensor, axes []int, keepdims bool) (*Tensor, error) {
	if err := requireFloat32(input, "reduce mean input"); err != nil {
		return nil, err
	}
	rank := len(input.shape)
	axisSet := make(map[int]struct{})
	if len(axes) == 0 {
		for axis := 0; axis < rank; axis++ {
			axisSet[axis] = struct{}{}
		}
	} else {
		for _, rawAxis := range axes {
			axis, axisErr := normalizeAxis(rawAxis, rank, "reduce mean")
			if axisErr != nil {
				return nil, axisErr
			}
			if _, exists := axisSet[axis]; exists {
				return nil, fmt.Errorf("reduce mean axis %d is repeated", rawAxis)
			}
			axisSet[axis] = struct{}{}
		}
	}
	outShape := make([]int, 0, rank)
	for axis, dimension := range input.shape {
		if _, reduced := axisSet[axis]; reduced {
			if keepdims {
				outShape = append(outShape, 1)
			}
			continue
		}
		outShape = append(outShape, dimension)
	}
	result, err := newFloat32Tensor(outShape, make([]float32, elementCount(outShape)))
	if err != nil {
		return nil, err
	}
	sums := make([]float64, len(result.data))
	counts := make([]int, len(result.data))
	for inputIndex, value := range input.data {
		coordinates := linearCoordinates(inputIndex, input.shape, input.strides)
		outputCoordinates := make([]int, 0, len(outShape))
		for axis, coordinate := range coordinates {
			if _, reduced := axisSet[axis]; reduced {
				if keepdims {
					outputCoordinates = append(outputCoordinates, 0)
				}
				continue
			}
			outputCoordinates = append(outputCoordinates, coordinate)
		}
		outputIndex := 0
		for axis, coordinate := range outputCoordinates {
			outputIndex += coordinate * result.strides[axis]
		}
		sums[outputIndex] += float64(value)
		counts[outputIndex]++
	}
	for index, count := range counts {
		if count != 0 {
			result.data[index] = float32(sums[index] / float64(count))
		}
	}
	return result, nil
}

func unary(name string, input *Tensor, operation func(float32) float32) (*Tensor, error) {
	if err := requireFloat32(input, name+" input"); err != nil {
		return nil, err
	}
	result, err := copyTensor(input)
	if err != nil {
		return nil, err
	}
	for index, value := range result.data {
		result.data[index] = operation(value)
	}
	return result, nil
}

// Softmax computes a numerically stable softmax along axis. With no axis, the
// last axis is used, matching ONNX's default.
func Softmax(input *Tensor, axes ...int) (*Tensor, error) {
	if err := requireFloat32(input, "softmax input"); err != nil {
		return nil, err
	}
	if len(axes) > 1 {
		return nil, fmt.Errorf("softmax accepts at most one axis")
	}
	axis := -1
	if len(axes) == 1 {
		axis = axes[0]
	}
	axis, err := normalizeAxis(axis, len(input.shape), "softmax")
	if err != nil {
		return nil, err
	}
	result, err := copyTensor(input)
	if err != nil {
		return nil, err
	}
	axisSize, inner, outer := input.shape[axis], 1, 1
	for index := axis + 1; index < len(input.shape); index++ {
		inner *= input.shape[index]
	}
	for index := 0; index < axis; index++ {
		outer *= input.shape[index]
	}
	for outerIndex := 0; outerIndex < outer; outerIndex++ {
		for innerIndex := 0; innerIndex < inner; innerIndex++ {
			base := outerIndex*axisSize*inner + innerIndex
			maxValue := float32(math.Inf(-1))
			for axisIndex := 0; axisIndex < axisSize; axisIndex++ {
				value := input.data[base+axisIndex*inner]
				if value > maxValue {
					maxValue = value
				}
			}
			var sum float64
			for axisIndex := 0; axisIndex < axisSize; axisIndex++ {
				value := math.Exp(float64(input.data[base+axisIndex*inner] - maxValue))
				sum += value
				result.data[base+axisIndex*inner] = float32(value)
			}
			for axisIndex := 0; axisIndex < axisSize; axisIndex++ {
				result.data[base+axisIndex*inner] = float32(float64(result.data[base+axisIndex*inner]) / sum)
			}
		}
	}
	return result, nil
}

// Identity returns an independent tensor with the same dtype, shape, and data.
func Identity(input *Tensor) (*Tensor, error) {
	return copyTensor(input)
}

// Reshape changes the row-major shape without changing data order. A single -1
// dimension is inferred from the input element count.
func Reshape(input *Tensor, shape []int) (*Tensor, error) {
	return reshapeWithOptions(input, shape, false)
}

func reshapeWithOptions(input *Tensor, requested []int, allowZero bool) (*Tensor, error) {
	if err := requireFloat32(input, "reshape input"); err != nil {
		return nil, err
	}
	shape := append([]int(nil), requested...)
	unknown, known := -1, 1
	for index, dimension := range shape {
		if dimension < -1 {
			return nil, fmt.Errorf("reshape shape %v has invalid dimension %d at index %d", requested, dimension, index)
		}
		if dimension == -1 {
			if unknown != -1 {
				return nil, fmt.Errorf("reshape shape %v has more than one inferred dimension", requested)
			}
			unknown = index
			continue
		}
		resolved := dimension
		if dimension == 0 && !allowZero {
			if index >= len(input.shape) {
				return nil, fmt.Errorf("reshape shape %v copies missing input dimension %d", requested, index)
			}
			resolved = input.shape[index]
		}
		shape[index] = resolved
		if resolved != 0 && known > maxInt()/resolved {
			return nil, fmt.Errorf("reshape shape %v overflows element count", requested)
		}
		known *= resolved
	}
	if unknown != -1 {
		if known == 0 || len(input.data)%known != 0 {
			return nil, fmt.Errorf("reshape shape %v cannot infer a dimension from %d elements", requested, len(input.data))
		}
		shape[unknown] = len(input.data) / known
	}
	shapeCopy, strides, count, err := makeLayout(shape)
	if err != nil {
		return nil, err
	}
	if count != len(input.data) {
		return nil, fmt.Errorf("reshape shape %v has %d elements, want %d", requested, count, len(input.data))
	}
	result, err := copyTensor(input)
	if err != nil {
		return nil, err
	}
	result.shape, result.strides = shapeCopy, strides
	return result, nil
}

// Flatten collapses dimensions before axis into one dimension and all
// dimensions from axis onward into the second dimension.
func Flatten(input *Tensor, axes ...int) (*Tensor, error) {
	if err := requireFloat32(input, "flatten input"); err != nil {
		return nil, err
	}
	if len(axes) > 1 {
		return nil, fmt.Errorf("flatten accepts at most one axis")
	}
	axis := 1
	if len(axes) == 1 {
		axis = axes[0]
	}
	if axis < 0 {
		axis += len(input.shape)
	}
	if axis < 0 || axis > len(input.shape) {
		return nil, fmt.Errorf("flatten axis %d is out of range for shape %v", axis, input.shape)
	}
	left, right := 1, 1
	for index, dimension := range input.shape {
		if index < axis {
			left *= dimension
		} else {
			right *= dimension
		}
	}
	return reshapeWithOptions(input, []int{left, right}, true)
}

// Transpose permutes dimensions. With no permutation, dimensions are reversed.
func Transpose(input *Tensor, perms ...[]int) (*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("transpose input is nil")
	}
	if !supportedTensorDType(input.dtype) {
		return nil, unsupportedDTypeError(input.dtype)
	}
	if len(perms) > 1 {
		return nil, fmt.Errorf("transpose accepts at most one permutation")
	}
	perm := make([]int, len(input.shape))
	if len(perms) == 0 || len(perms[0]) == 0 {
		for index := range perm {
			perm[index] = len(perm) - 1 - index
		}
	} else {
		perm = append([]int(nil), perms[0]...)
	}
	if len(perm) != len(input.shape) {
		return nil, fmt.Errorf("transpose permutation %v has length %d for shape %v", perm, len(perm), input.shape)
	}
	seen := make([]bool, len(perm))
	outShape := make([]int, len(perm))
	for index, source := range perm {
		if source < 0 || source >= len(perm) || seen[source] {
			return nil, fmt.Errorf("transpose permutation %v is invalid for shape %v", perm, input.shape)
		}
		seen[source] = true
		outShape[index] = input.shape[source]
	}
	result, err := newTypedTensor(input.dtype, outShape)
	if err != nil {
		return nil, err
	}
	for outputIndex := 0; outputIndex < result.Len(); outputIndex++ {
		remaining, inputIndex := outputIndex, 0
		for outputDimension := len(outShape) - 1; outputDimension >= 0; outputDimension-- {
			coordinate := 0
			if outShape[outputDimension] != 0 {
				coordinate = remaining % outShape[outputDimension]
				remaining /= outShape[outputDimension]
			}
			inputIndex += coordinate * input.strides[perm[outputDimension]]
		}
		copyTensorElement(result, outputIndex, input, inputIndex)
	}
	return result, nil
}

// Cast converts among the scalar types used by the exporter: float32, int64,
// bool, and string. Numeric conversions follow ONNX's truncating integer
// conversion for the paths used by categorical preprocessing.
func Cast(input *Tensor, to DType) (*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("cast input is nil")
	}
	if !supportedTensorDType(input.dtype) {
		return nil, unsupportedDTypeError(input.dtype)
	}
	if !supportedTensorDType(to) {
		return nil, fmt.Errorf("cast to dtype %s is not implemented", dtypeName(to))
	}
	result, err := newTypedTensor(to, input.shape)
	if err != nil {
		return nil, err
	}
	for index := 0; index < input.Len(); index++ {
		if err := castElement(result, index, input, index); err != nil {
			return nil, fmt.Errorf("cast element %d: %w", index, err)
		}
	}
	return result, nil
}

func castElement(destination *Tensor, destinationIndex int, source *Tensor, sourceIndex int) error {
	switch destination.dtype {
	case DTypeFloat32:
		switch source.dtype {
		case DTypeFloat32:
			destination.data[destinationIndex] = source.data[sourceIndex]
		case DTypeInt64:
			destination.data[destinationIndex] = float32(source.int64Data[sourceIndex])
		case DTypeBool:
			if source.boolData[sourceIndex] {
				destination.data[destinationIndex] = 1
			}
		case DTypeString:
			value, err := strconv.ParseFloat(source.stringData[sourceIndex], 32)
			if err != nil {
				return fmt.Errorf("string %q is not numeric", source.stringData[sourceIndex])
			}
			destination.data[destinationIndex] = float32(value)
		}
	case DTypeInt64:
		switch source.dtype {
		case DTypeFloat32:
			if math.IsNaN(float64(source.data[sourceIndex])) || math.IsInf(float64(source.data[sourceIndex]), 0) {
				return fmt.Errorf("float %v cannot convert to int64", source.data[sourceIndex])
			}
			destination.int64Data[destinationIndex] = int64(source.data[sourceIndex])
		case DTypeInt64:
			destination.int64Data[destinationIndex] = source.int64Data[sourceIndex]
		case DTypeBool:
			if source.boolData[sourceIndex] {
				destination.int64Data[destinationIndex] = 1
			}
		case DTypeString:
			value, err := strconv.ParseInt(source.stringData[sourceIndex], 10, 64)
			if err != nil {
				return fmt.Errorf("string %q is not an integer", source.stringData[sourceIndex])
			}
			destination.int64Data[destinationIndex] = value
		}
	case DTypeBool:
		switch source.dtype {
		case DTypeFloat32:
			destination.boolData[destinationIndex] = source.data[sourceIndex] != 0
		case DTypeInt64:
			destination.boolData[destinationIndex] = source.int64Data[sourceIndex] != 0
		case DTypeBool:
			destination.boolData[destinationIndex] = source.boolData[sourceIndex]
		case DTypeString:
			switch source.stringData[sourceIndex] {
			case "1", "true", "True", "TRUE":
				destination.boolData[destinationIndex] = true
			case "0", "false", "False", "FALSE":
			default:
				return fmt.Errorf("string %q is not boolean", source.stringData[sourceIndex])
			}
		}
	case DTypeString:
		switch source.dtype {
		case DTypeFloat32:
			destination.stringData[destinationIndex] = strconv.FormatFloat(float64(source.data[sourceIndex]), 'g', -1, 32)
		case DTypeInt64:
			destination.stringData[destinationIndex] = strconv.FormatInt(source.int64Data[sourceIndex], 10)
		case DTypeBool:
			destination.stringData[destinationIndex] = strconv.FormatBool(source.boolData[sourceIndex])
		case DTypeString:
			destination.stringData[destinationIndex] = source.stringData[sourceIndex]
		}
	default:
		return unsupportedDTypeError(destination.dtype)
	}
	return nil
}

// Constant returns an independent copy of a tensor attribute.
func Constant(value *Tensor) (*Tensor, error) {
	return copyTensor(value)
}

func normalizeAxis(axis, rank int, operation string) (int, error) {
	if rank == 0 {
		return 0, fmt.Errorf("%s axis %d is invalid for a scalar", operation, axis)
	}
	if axis < 0 {
		axis += rank
	}
	if axis < 0 || axis >= rank {
		return 0, fmt.Errorf("%s axis %d is out of range for rank %d", operation, axis, rank)
	}
	return axis, nil
}

func elementCount(shape []int) int {
	count := 1
	for _, dimension := range shape {
		count *= dimension
	}
	return count
}

func sameShape(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func stridesForShape(shape []int) []int {
	strides := make([]int, len(shape))
	stride := 1
	for index := len(shape) - 1; index >= 0; index-- {
		strides[index] = stride
		stride *= shape[index]
	}
	return strides
}
