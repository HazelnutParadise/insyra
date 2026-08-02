// Package dl provides pure-Go inference for a focused subset of ONNX models.
package dl

import (
	"fmt"
)

// DType identifies the element type carried by a Tensor. Float32 is the
// arithmetic dtype used by the inference kernels; int64, bool, and string are
// also stored so ONNX categorical preprocessing and classifier outputs retain
// their declared types.
type DType string

const (
	DTypeUnknown  DType = "unknown"
	DTypeFloat32  DType = "float32"
	DTypeFloat16  DType = "float16"
	DTypeFloat64  DType = "float64"
	DTypeInt8     DType = "int8"
	DTypeUInt8    DType = "uint8"
	DTypeInt16    DType = "int16"
	DTypeUInt16   DType = "uint16"
	DTypeInt32    DType = "int32"
	DTypeUInt32   DType = "uint32"
	DTypeInt64    DType = "int64"
	DTypeUInt64   DType = "uint64"
	DTypeBool     DType = "bool"
	DTypeString   DType = "string"
	DTypeFloat8   DType = "float8"
	DTypeBFloat16 DType = "bfloat16"
)

// Short aliases make kernel call sites read like the ONNX dtype names while
// the DType-prefixed constants remain unambiguous in documentation.
const (
	Float32 = DTypeFloat32
	Float16 = DTypeFloat16
	Float64 = DTypeFloat64
)

// DataType is kept as an alias for callers that use ONNX's terminology.
type DataType = DType

// Tensor is a row-major, N-dimensional tensor. Its data slice is private so a
// caller cannot change a model initializer while another goroutine is running
// the same Model. Data returns a copy for the same reason.
type Tensor struct {
	dtype      DType
	shape      []int
	strides    []int
	data       []float32
	int64Data  []int64
	boolData   []bool
	stringData []string
}

// NewTensor constructs a float32 tensor with the supplied row-major data.
// A nil or empty shape denotes a scalar and therefore requires one value.
func NewTensor(shape []int, data []float32) (*Tensor, error) {
	return newFloat32Tensor(shape, data)
}

// NewFloat32Tensor is an explicit spelling of NewTensor for code that builds
// tensors alongside values of other, future dtypes.
func NewFloat32Tensor(shape []int, data []float32) (*Tensor, error) {
	return newFloat32Tensor(shape, data)
}

// NewInt64Tensor constructs an int64 tensor.
func NewInt64Tensor(shape []int, data []int64) (*Tensor, error) {
	return newInt64Tensor(shape, data)
}

// NewStringTensor constructs a string tensor.
func NewStringTensor(shape []int, data []string) (*Tensor, error) {
	return newStringTensor(shape, data)
}

// NewBoolTensor constructs a bool tensor.
func NewBoolTensor(shape []int, data []bool) (*Tensor, error) {
	return newBoolTensor(shape, data)
}

// NewTensorWithDType constructs a tensor when the dtype is known at a call
// site. The data argument is float32, so this constructor is intentionally
// limited to float32 tensors.
func NewTensorWithDType(dtype DType, shape []int, data []float32) (*Tensor, error) {
	if dtype != DTypeFloat32 {
		return nil, unsupportedDTypeError(dtype)
	}
	return newFloat32Tensor(shape, data)
}

// DType reports the tensor's explicit element type.
func (t *Tensor) DType() DType {
	if t == nil {
		return DTypeUnknown
	}
	return t.dtype
}

// Shape returns a copy of the tensor's dimensions.
func (t *Tensor) Shape() []int {
	if t == nil {
		return nil
	}
	return append([]int(nil), t.shape...)
}

// Strides returns row-major element strides. For example, a [2, 3, 4] tensor
// has strides [12, 4, 1].
func (t *Tensor) Strides() []int {
	if t == nil {
		return nil
	}
	return append([]int(nil), t.strides...)
}

// Data returns a copy of float32 data. It returns nil for a non-float32
// control tensor used internally by an ONNX shape operation.
func (t *Tensor) Data() []float32 {
	if t == nil || t.dtype != DTypeFloat32 {
		return nil
	}
	return append([]float32(nil), t.data...)
}

// Float32Data is the error-reporting form of Data.
func (t *Tensor) Float32Data() ([]float32, error) {
	if t == nil {
		return nil, fmt.Errorf("tensor is nil")
	}
	if t.dtype != DTypeFloat32 {
		return nil, unsupportedDTypeError(t.dtype)
	}
	return append([]float32(nil), t.data...), nil
}

// Int64Data returns a copy of int64 data.
func (t *Tensor) Int64Data() ([]int64, error) {
	if t == nil {
		return nil, fmt.Errorf("tensor is nil")
	}
	if t.dtype != DTypeInt64 {
		return nil, unsupportedDTypeError(t.dtype)
	}
	return append([]int64(nil), t.int64Data...), nil
}

// StringData returns a copy of string data.
func (t *Tensor) StringData() ([]string, error) {
	if t == nil {
		return nil, fmt.Errorf("tensor is nil")
	}
	if t.dtype != DTypeString {
		return nil, unsupportedDTypeError(t.dtype)
	}
	return append([]string(nil), t.stringData...), nil
}

// BoolData returns a copy of bool data.
func (t *Tensor) BoolData() ([]bool, error) {
	if t == nil {
		return nil, fmt.Errorf("tensor is nil")
	}
	if t.dtype != DTypeBool {
		return nil, unsupportedDTypeError(t.dtype)
	}
	return append([]bool(nil), t.boolData...), nil
}

// Len returns the number of elements in the tensor.
func (t *Tensor) Len() int {
	if t == nil {
		return 0
	}
	switch t.dtype {
	case DTypeFloat32:
		return len(t.data)
	case DTypeInt64:
		return len(t.int64Data)
	case DTypeBool:
		return len(t.boolData)
	case DTypeString:
		return len(t.stringData)
	default:
		return 0
	}
}

func newFloat32Tensor(shape []int, data []float32) (*Tensor, error) {
	shapeCopy, strides, count, err := makeLayout(shape)
	if err != nil {
		return nil, err
	}
	if len(data) != count {
		return nil, fmt.Errorf("tensor data has %d elements, want %d for shape %v", len(data), count, shapeCopy)
	}
	return &Tensor{
		dtype:   DTypeFloat32,
		shape:   shapeCopy,
		strides: strides,
		data:    append([]float32(nil), data...),
	}, nil
}

func newInt64Tensor(shape []int, data []int64) (*Tensor, error) {
	shapeCopy, strides, count, err := makeLayout(shape)
	if err != nil {
		return nil, err
	}
	if len(data) != count {
		return nil, fmt.Errorf("tensor data has %d elements, want %d for shape %v", len(data), count, shapeCopy)
	}
	return &Tensor{
		dtype:     DTypeInt64,
		shape:     shapeCopy,
		strides:   strides,
		int64Data: append([]int64(nil), data...),
	}, nil
}

func newStringTensor(shape []int, data []string) (*Tensor, error) {
	shapeCopy, strides, count, err := makeLayout(shape)
	if err != nil {
		return nil, err
	}
	if len(data) != count {
		return nil, fmt.Errorf("tensor data has %d elements, want %d for shape %v", len(data), count, shapeCopy)
	}
	return &Tensor{
		dtype:      DTypeString,
		shape:      shapeCopy,
		strides:    strides,
		stringData: append([]string(nil), data...),
	}, nil
}

func newBoolTensor(shape []int, data []bool) (*Tensor, error) {
	shapeCopy, strides, count, err := makeLayout(shape)
	if err != nil {
		return nil, err
	}
	if len(data) != count {
		return nil, fmt.Errorf("tensor data has %d elements, want %d for shape %v", len(data), count, shapeCopy)
	}
	return &Tensor{
		dtype:    DTypeBool,
		shape:    shapeCopy,
		strides:  strides,
		boolData: append([]bool(nil), data...),
	}, nil
}

func copyTensor(t *Tensor) (*Tensor, error) {
	if t == nil {
		return nil, fmt.Errorf("tensor is nil")
	}
	result := &Tensor{
		dtype:      t.dtype,
		shape:      append([]int(nil), t.shape...),
		strides:    append([]int(nil), t.strides...),
		data:       append([]float32(nil), t.data...),
		int64Data:  append([]int64(nil), t.int64Data...),
		boolData:   append([]bool(nil), t.boolData...),
		stringData: append([]string(nil), t.stringData...),
	}
	return result, nil
}

func makeLayout(shape []int) ([]int, []int, int, error) {
	shapeCopy := append([]int(nil), shape...)
	count := 1
	for index, dimension := range shapeCopy {
		if dimension < 0 {
			return nil, nil, 0, fmt.Errorf("shape %v has negative dimension at index %d", shapeCopy, index)
		}
		if dimension != 0 && count > maxInt()/dimension {
			return nil, nil, 0, fmt.Errorf("shape %v overflows element count", shapeCopy)
		}
		count *= dimension
	}
	strides := make([]int, len(shapeCopy))
	stride := 1
	for index := len(shapeCopy) - 1; index >= 0; index-- {
		strides[index] = stride
		if shapeCopy[index] != 0 && stride > maxInt()/shapeCopy[index] {
			return nil, nil, 0, fmt.Errorf("shape %v overflows strides", shapeCopy)
		}
		stride *= shapeCopy[index]
	}
	return shapeCopy, strides, count, nil
}

func maxInt() int {
	return int(^uint(0) >> 1)
}

func unsupportedDTypeError(dtype DType) error {
	return fmt.Errorf("dtype %s is not implemented", dtypeName(dtype))
}

func dtypeName(dtype DType) string {
	if dtype == "" {
		return string(DTypeUnknown)
	}
	return string(dtype)
}

func requireFloat32(t *Tensor, operand string) error {
	if t == nil {
		return fmt.Errorf("%s tensor is nil", operand)
	}
	if t.dtype != DTypeFloat32 {
		return fmt.Errorf("%s has unsupported dtype %s", operand, dtypeName(t.dtype))
	}
	return nil
}

func tensorBroadcastBinary(left, right *Tensor, operation string, fn func(float32, float32) float32) (*Tensor, error) {
	if err := requireFloat32(left, "left operand"); err != nil {
		return nil, err
	}
	if err := requireFloat32(right, "right operand"); err != nil {
		return nil, err
	}
	if fn == nil {
		return nil, fmt.Errorf("%s operation is nil", operation)
	}

	shape, err := tensorBroadcastShape(left.shape, right.shape)
	if err != nil {
		return nil, err
	}
	shape, strides, count, err := makeLayout(shape)
	if err != nil {
		return nil, err
	}
	result := &Tensor{
		dtype:   DTypeFloat32,
		shape:   shape,
		strides: strides,
		data:    make([]float32, count),
	}

	leftStrides := alignedBroadcastStrides(left, len(shape))
	rightStrides := alignedBroadcastStrides(right, len(shape))
	for outputIndex := 0; outputIndex < count; outputIndex++ {
		remaining := outputIndex
		leftIndex := 0
		rightIndex := 0
		for axis, stride := range strides {
			coordinate := 0
			if stride != 0 {
				coordinate = remaining / stride
				remaining %= stride
			}
			leftIndex += coordinate * leftStrides[axis]
			rightIndex += coordinate * rightStrides[axis]
		}
		result.data[outputIndex] = fn(left.data[leftIndex], right.data[rightIndex])
	}
	return result, nil
}

func tensorBroadcastShape(left, right []int) ([]int, error) {
	rank := len(left)
	if len(right) > rank {
		rank = len(right)
	}
	shape := make([]int, rank)
	for axis := 0; axis < rank; axis++ {
		leftDimension := alignedDimension(left, rank, axis)
		rightDimension := alignedDimension(right, rank, axis)
		switch {
		case leftDimension == rightDimension:
			shape[axis] = leftDimension
		case leftDimension == 1:
			shape[axis] = rightDimension
		case rightDimension == 1:
			shape[axis] = leftDimension
		default:
			return nil, fmt.Errorf("cannot broadcast shapes %v and %v", left, right)
		}
	}
	return shape, nil
}

func alignedDimension(shape []int, rank, axis int) int {
	shapeAxis := axis - (rank - len(shape))
	if shapeAxis < 0 {
		return 1
	}
	return shape[shapeAxis]
}

func alignedBroadcastStrides(t *Tensor, outputRank int) []int {
	strides := make([]int, outputRank)
	axisOffset := outputRank - len(t.shape)
	for axis := 0; axis < outputRank; axis++ {
		inputAxis := axis - axisOffset
		if inputAxis < 0 {
			strides[axis] = 0
			continue
		}
		if t.shape[inputAxis] == 1 {
			strides[axis] = 0
			continue
		}
		strides[axis] = t.strides[inputAxis]
	}
	return strides
}
