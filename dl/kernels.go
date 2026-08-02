package dl

import (
	"fmt"
	"math"
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

// MatMul computes a two-dimensional matrix product.
func MatMul(a, b *Tensor) (*Tensor, error) {
	if err := requireFloat32(a, "matmul input A"); err != nil {
		return nil, err
	}
	if err := requireFloat32(b, "matmul input B"); err != nil {
		return nil, err
	}
	if len(a.shape) != 2 || len(b.shape) != 2 {
		return nil, fmt.Errorf("matmul requires 2-D inputs, got shapes %v and %v", a.shape, b.shape)
	}
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
	if err := requireFloat32(input, "transpose input"); err != nil {
		return nil, err
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
	result, err := newFloat32Tensor(outShape, make([]float32, elementCount(outShape)))
	if err != nil {
		return nil, err
	}
	for outputIndex := range result.data {
		remaining, inputIndex := outputIndex, 0
		for outputDimension := len(outShape) - 1; outputDimension >= 0; outputDimension-- {
			coordinate := 0
			if outShape[outputDimension] != 0 {
				coordinate = remaining % outShape[outputDimension]
				remaining /= outShape[outputDimension]
			}
			inputIndex += coordinate * input.strides[perm[outputDimension]]
		}
		result.data[outputIndex] = input.data[inputIndex]
	}
	return result, nil
}

// Cast accepts only float32-to-float32 because other float storage and
// conversion rules are not implemented.
func Cast(input *Tensor, to DType) (*Tensor, error) {
	if err := requireFloat32(input, "cast input"); err != nil {
		return nil, err
	}
	if to != DTypeFloat32 {
		return nil, fmt.Errorf("cast to dtype %s is not implemented", dtypeName(to))
	}
	return copyTensor(input)
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
