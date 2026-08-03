package nn

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"sync"

	insyra "github.com/HazelnutParadise/insyra"
)

// parallelMACThreshold keeps dispatch overhead out of small graphs. A quick
// best-of-five measurement on the 8-core M3 crossed over between 4K and 16K
// MatMul MACs and between 2K and 18K Conv MACs; 100K leaves a safety margin.
// Keep the threshold in multiply-accumulate units so it applies to both.
const parallelMACThreshold = 100_000

func parallelWorkerCountForMACs(factors ...int) int {
	work := 1
	for _, factor := range factors {
		if factor <= 0 {
			return 1
		}
		if work > parallelMACThreshold/factor {
			return runtime.NumCPU()
		}
		work *= factor
	}
	if work <= parallelMACThreshold {
		return 1
	}
	return runtime.NumCPU()
}

func parallelFor(total, workers int, work func(start, end int)) {
	if total <= 0 {
		return
	}
	if workers <= 1 {
		work(0, total)
		return
	}
	if workers > total {
		workers = total
	}

	base, remainder := total/workers, total%workers
	var wg sync.WaitGroup
	start := 0
	for worker := 0; worker < workers; worker++ {
		size := base
		if worker < remainder {
			size++
		}
		end := start + size
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			work(start, end)
		}(start, end)
		start = end
	}
	wg.Wait()
}

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

// ConvOptions controls the spatial attributes of a two-dimensional Conv.
// The input and output layout is NCHW and the weight layout is OIHW.
type ConvOptions struct {
	Pads      []int
	AutoPad   string
	Strides   []int
	Dilations []int
	Group     int
}

// PoolOptions controls the spatial attributes shared by MaxPool and
// AveragePool. CountIncludePad is used by AveragePool only. CeilMode and
// StorageOrder are exposed so direct kernel callers receive the same refusal
// as graph execution when an unsupported ONNX value is requested.
type PoolOptions struct {
	Pads            []int
	AutoPad         string
	Strides         []int
	CountIncludePad bool
	CeilMode        int
	StorageOrder    int
}

// MaxPoolOptions and AveragePoolOptions are descriptive aliases for callers
// that want to name the operator-specific option type at a call site.
type MaxPoolOptions = PoolOptions
type AveragePoolOptions = PoolOptions

// Conv computes a two-dimensional NCHW convolution. The accumulator stays in
// float64 until each output value is narrowed to the tensor's float32 dtype.
func Conv(input, weights, bias *Tensor, options ...ConvOptions) (*Tensor, error) {
	return convWithWorkers(input, weights, bias, 0, options...)
}

func convWithWorkers(input, weights, bias *Tensor, workers int, options ...ConvOptions) (*Tensor, error) {
	if err := requireFloat32(input, "conv input"); err != nil {
		return nil, err
	}
	if err := requireFloat32(weights, "conv weights"); err != nil {
		return nil, err
	}
	if bias != nil {
		if err := requireFloat32(bias, "conv bias"); err != nil {
			return nil, err
		}
	}
	if len(options) > 1 {
		return nil, fmt.Errorf("conv accepts at most one options value")
	}
	if len(input.shape) != 4 || len(weights.shape) != 4 {
		return nil, fmt.Errorf("conv requires input and weights to be 4-D, got shapes %v and %v", input.shape, weights.shape)
	}

	opts := ConvOptions{Group: 1}
	if len(options) == 1 {
		opts = options[0]
		if opts.Group == 0 {
			opts.Group = 1
		}
	}
	if opts.Group <= 0 {
		return nil, fmt.Errorf("conv group %d must be positive", opts.Group)
	}
	inputChannels := input.shape[1]
	outputChannels := weights.shape[0]
	if inputChannels == 0 || outputChannels == 0 {
		return nil, fmt.Errorf("conv input and output channels must be positive, got shapes %v and %v", input.shape, weights.shape)
	}
	if inputChannels%opts.Group != 0 {
		return nil, fmt.Errorf("conv input channels %d are not divisible by group %d for shapes %v and %v", inputChannels, opts.Group, input.shape, weights.shape)
	}
	if outputChannels%opts.Group != 0 {
		return nil, fmt.Errorf("conv output channels %d are not divisible by group %d for shapes %v and %v", outputChannels, opts.Group, input.shape, weights.shape)
	}
	if weights.shape[1] != inputChannels/opts.Group {
		return nil, fmt.Errorf("conv weight input channels %d do not match input channels per group %d for shapes %v and %v", weights.shape[1], inputChannels/opts.Group, input.shape, weights.shape)
	}
	if bias != nil && !sameShape(bias.shape, []int{outputChannels}) {
		return nil, fmt.Errorf("conv bias shape %v does not match output channels %d", bias.shape, outputChannels)
	}

	window, err := resolve2DWindow(input.shape[2], input.shape[3], weights.shape[2], weights.shape[3], opts.Pads, opts.AutoPad, opts.Strides, opts.Dilations, "conv")
	if err != nil {
		return nil, err
	}
	result, err := newZeroFloat32Tensor([]int{input.shape[0], outputChannels, window.outputH, window.outputW})
	if err != nil {
		return nil, err
	}
	inputHeight, inputWidth := input.shape[2], input.shape[3]
	inputChannelsPerGroup := inputChannels / opts.Group
	outputChannelsPerGroup := outputChannels / opts.Group
	if workers <= 0 {
		workers = parallelWorkerCountForMACs(
			input.shape[0], outputChannels, window.outputH, window.outputW,
			inputChannelsPerGroup, weights.shape[2], weights.shape[3],
		)
	}
	outputRows, err := checkedProduct(input.shape[0], outputChannels, "conv output rows")
	if err != nil {
		return nil, err
	}
	outputRows, err = checkedProduct(outputRows, window.outputH, "conv output rows")
	if err != nil {
		return nil, err
	}
	parallelFor(outputRows, workers, func(start, end int) {
		for outputIndex := start; outputIndex < end; outputIndex++ {
			batchAndChannel := outputIndex / window.outputH
			outputRow := outputIndex % window.outputH
			batch := batchAndChannel / outputChannels
			outputChannel := batchAndChannel % outputChannels
			group := outputChannel / outputChannelsPerGroup
			inputChannelStart := group * inputChannelsPerGroup
			for outputColumn := 0; outputColumn < window.outputW; outputColumn++ {
				var sum float64
				for inputChannel := 0; inputChannel < inputChannelsPerGroup; inputChannel++ {
					for kernelRow := 0; kernelRow < weights.shape[2]; kernelRow++ {
						inputRow := outputRow*window.strideH - window.padTop + kernelRow*window.dilationH
						if inputRow < 0 || inputRow >= inputHeight {
							continue
						}
						for kernelColumn := 0; kernelColumn < weights.shape[3]; kernelColumn++ {
							inputColumn := outputColumn*window.strideW - window.padLeft + kernelColumn*window.dilationW
							if inputColumn < 0 || inputColumn >= inputWidth {
								continue
							}
							inputIndex := ((batch*inputChannels+(inputChannelStart+inputChannel))*inputHeight+inputRow)*inputWidth + inputColumn
							weightIndex := ((outputChannel*weights.shape[1]+inputChannel)*weights.shape[2]+kernelRow)*weights.shape[3] + kernelColumn
							sum += float64(input.data[inputIndex]) * float64(weights.data[weightIndex])
						}
					}
				}
				if bias != nil {
					sum += float64(bias.data[outputChannel])
				}
				result.data[((batch*outputChannels+outputChannel)*window.outputH+outputRow)*window.outputW+outputColumn] = float32(sum)
			}
		}
	})
	return result, nil
}

// MaxPool computes a two-dimensional NCHW max pool. Values outside the input
// are treated as negative infinity and do not contribute a second output.
func MaxPool(input *Tensor, kernelShape []int, options ...PoolOptions) (*Tensor, error) {
	if err := requireFloat32(input, "max pool input"); err != nil {
		return nil, err
	}
	if len(options) > 1 {
		return nil, fmt.Errorf("max pool accepts at most one options value")
	}
	if len(input.shape) != 4 {
		return nil, fmt.Errorf("max pool requires a 4-D input, got shape %v", input.shape)
	}
	opts := PoolOptions{}
	if len(options) == 1 {
		opts = options[0]
	}
	if opts.CeilMode != 0 {
		return nil, fmt.Errorf("max pool ceil_mode value %d is unsupported; only 0 is supported", opts.CeilMode)
	}
	if opts.StorageOrder != 0 {
		return nil, fmt.Errorf("max pool storage_order value %d is unsupported; only 0 is supported", opts.StorageOrder)
	}
	window, err := resolvePoolWindow(input, kernelShape, opts, "max pool")
	if err != nil {
		return nil, err
	}
	result, err := newZeroFloat32Tensor([]int{input.shape[0], input.shape[1], window.outputH, window.outputW})
	if err != nil {
		return nil, err
	}
	for batch := 0; batch < input.shape[0]; batch++ {
		for channel := 0; channel < input.shape[1]; channel++ {
			for outputRow := 0; outputRow < window.outputH; outputRow++ {
				for outputColumn := 0; outputColumn < window.outputW; outputColumn++ {
					maximum := float32(math.Inf(-1))
					for kernelRow := 0; kernelRow < kernelShape[0]; kernelRow++ {
						inputRow := outputRow*window.strideH - window.padTop + kernelRow
						if inputRow < 0 || inputRow >= input.shape[2] {
							continue
						}
						for kernelColumn := 0; kernelColumn < kernelShape[1]; kernelColumn++ {
							inputColumn := outputColumn*window.strideW - window.padLeft + kernelColumn
							if inputColumn < 0 || inputColumn >= input.shape[3] {
								continue
							}
							inputIndex := ((batch*input.shape[1]+channel)*input.shape[2]+inputRow)*input.shape[3] + inputColumn
							if input.data[inputIndex] > maximum {
								maximum = input.data[inputIndex]
							}
						}
					}
					result.data[((batch*input.shape[1]+channel)*window.outputH+outputRow)*window.outputW+outputColumn] = maximum
				}
			}
		}
	}
	return result, nil
}

// AveragePool computes a two-dimensional NCHW average pool. When
// CountIncludePad is true the denominator includes every padded element.
func AveragePool(input *Tensor, kernelShape []int, options ...PoolOptions) (*Tensor, error) {
	if err := requireFloat32(input, "average pool input"); err != nil {
		return nil, err
	}
	if len(options) > 1 {
		return nil, fmt.Errorf("average pool accepts at most one options value")
	}
	if len(input.shape) != 4 {
		return nil, fmt.Errorf("average pool requires a 4-D input, got shape %v", input.shape)
	}
	opts := PoolOptions{}
	if len(options) == 1 {
		opts = options[0]
	}
	if opts.CeilMode != 0 {
		return nil, fmt.Errorf("average pool ceil_mode value %d is unsupported; only 0 is supported", opts.CeilMode)
	}
	if opts.StorageOrder != 0 {
		return nil, fmt.Errorf("average pool storage_order value %d is unsupported; only 0 is supported", opts.StorageOrder)
	}
	window, err := resolvePoolWindow(input, kernelShape, opts, "average pool")
	if err != nil {
		return nil, err
	}
	result, err := newZeroFloat32Tensor([]int{input.shape[0], input.shape[1], window.outputH, window.outputW})
	if err != nil {
		return nil, err
	}
	for batch := 0; batch < input.shape[0]; batch++ {
		for channel := 0; channel < input.shape[1]; channel++ {
			for outputRow := 0; outputRow < window.outputH; outputRow++ {
				for outputColumn := 0; outputColumn < window.outputW; outputColumn++ {
					var sum float64
					validCount := 0
					for kernelRow := 0; kernelRow < kernelShape[0]; kernelRow++ {
						inputRow := outputRow*window.strideH - window.padTop + kernelRow
						if inputRow < 0 || inputRow >= input.shape[2] {
							continue
						}
						for kernelColumn := 0; kernelColumn < kernelShape[1]; kernelColumn++ {
							inputColumn := outputColumn*window.strideW - window.padLeft + kernelColumn
							if inputColumn < 0 || inputColumn >= input.shape[3] {
								continue
							}
							inputIndex := ((batch*input.shape[1]+channel)*input.shape[2]+inputRow)*input.shape[3] + inputColumn
							sum += float64(input.data[inputIndex])
							validCount++
						}
					}
					denominator := validCount
					if opts.CountIncludePad {
						denominator = kernelShape[0] * kernelShape[1]
					}
					if denominator > 0 {
						result.data[((batch*input.shape[1]+channel)*window.outputH+outputRow)*window.outputW+outputColumn] = float32(sum / float64(denominator))
					}
				}
			}
		}
	}
	return result, nil
}

// GlobalAveragePool averages every spatial location of an NCHW input and
// retains singleton spatial dimensions in the output.
func GlobalAveragePool(input *Tensor) (*Tensor, error) {
	if err := requireFloat32(input, "global average pool input"); err != nil {
		return nil, err
	}
	if len(input.shape) != 4 {
		return nil, fmt.Errorf("global average pool requires a 4-D input, got shape %v", input.shape)
	}
	spatialCount := input.shape[2] * input.shape[3]
	if spatialCount == 0 {
		return nil, fmt.Errorf("global average pool cannot reduce an empty spatial shape %v", input.shape[2:])
	}
	result, err := newZeroFloat32Tensor([]int{input.shape[0], input.shape[1], 1, 1})
	if err != nil {
		return nil, err
	}
	for batch := 0; batch < input.shape[0]; batch++ {
		for channel := 0; channel < input.shape[1]; channel++ {
			base := (batch*input.shape[1] + channel) * spatialCount
			var sum float64
			for index := 0; index < spatialCount; index++ {
				sum += float64(input.data[base+index])
			}
			result.data[batch*input.shape[1]+channel] = float32(sum / float64(spatialCount))
		}
	}
	return result, nil
}

// BatchNormalization applies inference-mode channel normalization to an input
// whose channel axis is one. The five parameter tensors are one-dimensional.
func BatchNormalization(input, scale, bias, mean, variance *Tensor, epsilonValues ...float32) (*Tensor, error) {
	if err := requireFloat32(input, "batch normalization input"); err != nil {
		return nil, err
	}
	for name, tensor := range map[string]*Tensor{"scale": scale, "bias": bias, "mean": mean, "variance": variance} {
		if err := requireFloat32(tensor, "batch normalization "+name); err != nil {
			return nil, err
		}
	}
	if len(epsilonValues) > 1 {
		return nil, fmt.Errorf("batch normalization accepts at most one epsilon value")
	}
	epsilon := float32(1e-5)
	if len(epsilonValues) == 1 {
		epsilon = epsilonValues[0]
	}
	if epsilon < 0 || math.IsNaN(float64(epsilon)) || math.IsInf(float64(epsilon), 0) {
		return nil, fmt.Errorf("batch normalization epsilon %g is invalid", epsilon)
	}
	if len(input.shape) < 2 {
		return nil, fmt.Errorf("batch normalization requires input rank at least 2, got shape %v", input.shape)
	}
	channels := input.shape[1]
	wantShape := []int{channels}
	for name, tensor := range map[string]*Tensor{"scale": scale, "bias": bias, "mean": mean, "variance": variance} {
		if !sameShape(tensor.shape, wantShape) {
			return nil, fmt.Errorf("batch normalization %s shape %v does not match input channel shape %v", name, tensor.shape, wantShape)
		}
	}
	result, err := copyTensor(input)
	if err != nil {
		return nil, err
	}
	channelStride := input.strides[1]
	for index, value := range input.data {
		channel := (index / channelStride) % channels
		normalized := (float64(value) - float64(mean.data[channel])) / math.Sqrt(float64(variance.data[channel])+float64(epsilon))
		result.data[index] = float32(normalized*float64(scale.data[channel]) + float64(bias.data[channel]))
	}
	return result, nil
}

// Pad applies constant padding. Pads are ordered as all begins followed by
// all ends, matching ONNX. Negative values crop the corresponding edge.
func Pad(input *Tensor, pads []int, values ...float32) (*Tensor, error) {
	if err := requireFloat32(input, "pad input"); err != nil {
		return nil, err
	}
	if len(values) > 1 {
		return nil, fmt.Errorf("pad accepts at most one constant value")
	}
	if len(pads) != 2*len(input.shape) {
		return nil, fmt.Errorf("pad pads %v has length %d, want %d for input shape %v", pads, len(pads), 2*len(input.shape), input.shape)
	}
	outputShape := make([]int, len(input.shape))
	for axis, dimension := range input.shape {
		outputDimension := dimension + pads[axis] + pads[len(input.shape)+axis]
		if outputDimension < 0 {
			return nil, fmt.Errorf("pad pads %v produce negative dimension %d at axis %d for input shape %v", pads, outputDimension, axis, input.shape)
		}
		outputShape[axis] = outputDimension
	}
	value := float32(0)
	if len(values) == 1 {
		value = values[0]
	}
	result, err := newFloat32Tensor(outputShape, make([]float32, elementCount(outputShape)))
	if err != nil {
		return nil, err
	}
	for index := range result.data {
		result.data[index] = value
	}
	for outputIndex := range result.data {
		remaining := outputIndex
		inputIndex := 0
		valid := true
		for axis, stride := range result.strides {
			coordinate := 0
			if stride != 0 {
				coordinate = remaining / stride
				remaining %= stride
			}
			sourceCoordinate := coordinate - pads[axis]
			if sourceCoordinate < 0 || sourceCoordinate >= input.shape[axis] {
				valid = false
				break
			}
			inputIndex += sourceCoordinate * input.strides[axis]
		}
		if valid {
			result.data[outputIndex] = input.data[inputIndex]
		}
	}
	return result, nil
}

type resolved2DWindow struct {
	padTop, padLeft, padBottom, padRight int
	strideH, strideW                     int
	dilationH, dilationW                 int
	outputH, outputW                     int
}

func resolvePoolWindow(input *Tensor, kernelShape []int, options PoolOptions, operation string) (resolved2DWindow, error) {
	if len(kernelShape) != 2 {
		return resolved2DWindow{}, fmt.Errorf("%s kernel shape %v must contain two dimensions", operation, kernelShape)
	}
	return resolve2DWindow(input.shape[2], input.shape[3], kernelShape[0], kernelShape[1], options.Pads, options.AutoPad, options.Strides, nil, operation)
}

func resolve2DWindow(inputH, inputW, kernelH, kernelW int, pads []int, autoPad string, strides, dilations []int, operation string) (resolved2DWindow, error) {
	if inputH <= 0 || inputW <= 0 {
		return resolved2DWindow{}, fmt.Errorf("%s requires positive spatial input dimensions, got [%d %d]", operation, inputH, inputW)
	}
	if kernelH <= 0 || kernelW <= 0 {
		return resolved2DWindow{}, fmt.Errorf("%s kernel shape [%d %d] must be positive", operation, kernelH, kernelW)
	}
	strideH, strideW, err := pairOption(strides, 1, operation+" strides")
	if err != nil {
		return resolved2DWindow{}, err
	}
	dilationH, dilationW, err := pairOption(dilations, 1, operation+" dilations")
	if err != nil {
		return resolved2DWindow{}, err
	}
	if strideH <= 0 || strideW <= 0 {
		return resolved2DWindow{}, fmt.Errorf("%s strides [%d %d] must be positive", operation, strideH, strideW)
	}
	if dilationH <= 0 || dilationW <= 0 {
		return resolved2DWindow{}, fmt.Errorf("%s dilations [%d %d] must be positive", operation, dilationH, dilationW)
	}
	effectiveH, err := effectiveKernelSize(kernelH, dilationH, operation)
	if err != nil {
		return resolved2DWindow{}, err
	}
	effectiveW, err := effectiveKernelSize(kernelW, dilationW, operation)
	if err != nil {
		return resolved2DWindow{}, err
	}
	if len(pads) != 0 && len(pads) != 4 {
		return resolved2DWindow{}, fmt.Errorf("%s pads %v must contain four values", operation, pads)
	}
	mode := autoPad
	if mode == "" {
		mode = "NOTSET"
	}
	if mode != "NOTSET" && len(pads) != 0 {
		return resolved2DWindow{}, fmt.Errorf("%s cannot combine auto_pad %q with explicit pads %v", operation, autoPad, pads)
	}
	window := resolved2DWindow{strideH: strideH, strideW: strideW, dilationH: dilationH, dilationW: dilationW}
	switch mode {
	case "NOTSET":
		if len(pads) == 4 {
			window.padTop, window.padLeft, window.padBottom, window.padRight = pads[0], pads[1], pads[2], pads[3]
		}
		if window.padTop < 0 || window.padLeft < 0 || window.padBottom < 0 || window.padRight < 0 {
			return resolved2DWindow{}, fmt.Errorf("%s pads %v must be non-negative", operation, pads)
		}
	case "VALID":
	case "SAME_UPPER", "SAME_LOWER":
		window.outputH = (inputH + strideH - 1) / strideH
		window.outputW = (inputW + strideW - 1) / strideW
		totalH := (window.outputH-1)*strideH + effectiveH - inputH
		totalW := (window.outputW-1)*strideW + effectiveW - inputW
		if totalH < 0 {
			totalH = 0
		}
		if totalW < 0 {
			totalW = 0
		}
		if mode == "SAME_UPPER" {
			window.padTop, window.padBottom = totalH/2, totalH-totalH/2
			window.padLeft, window.padRight = totalW/2, totalW-totalW/2
		} else {
			window.padTop, window.padBottom = (totalH+1)/2, totalH-(totalH+1)/2
			window.padLeft, window.padRight = (totalW+1)/2, totalW-(totalW+1)/2
		}
	default:
		return resolved2DWindow{}, fmt.Errorf("%s auto_pad value %q is unsupported", operation, autoPad)
	}
	if window.outputH == 0 {
		window.outputH, err = windowOutputDimension(inputH, window.padTop, window.padBottom, effectiveH, strideH, operation)
		if err != nil {
			return resolved2DWindow{}, err
		}
	}
	if window.outputW == 0 {
		window.outputW, err = windowOutputDimension(inputW, window.padLeft, window.padRight, effectiveW, strideW, operation)
		if err != nil {
			return resolved2DWindow{}, err
		}
	}
	return window, nil
}

func pairOption(values []int, fallback int, name string) (int, int, error) {
	if len(values) == 0 {
		return fallback, fallback, nil
	}
	if len(values) != 2 {
		return 0, 0, fmt.Errorf("%s %v must contain two values", name, values)
	}
	return values[0], values[1], nil
}

func effectiveKernelSize(kernel, dilation int, operation string) (int, error) {
	if kernel-1 > (maxInt()-1)/dilation {
		return 0, fmt.Errorf("%s effective kernel size overflows for kernel %d and dilation %d", operation, kernel, dilation)
	}
	return (kernel-1)*dilation + 1, nil
}

func windowOutputDimension(input, padBefore, padAfter, effectiveKernel, stride int, operation string) (int, error) {
	numerator := input + padBefore + padAfter - effectiveKernel
	if numerator < 0 {
		return 0, fmt.Errorf("%s padding and kernel produce no output for input %d", operation, input)
	}
	return numerator/stride + 1, nil
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
	return matMulBatchedWithWorkers(a, b, 0)
}

func matMulBatchedWithWorkers(a, b *Tensor, workers int) (*Tensor, error) {
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
	aBatchBases := make([]int, batchCount)
	bBatchBases := make([]int, batchCount)
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
		aBatchBases[batchIndex] = aBase
		bBatchBases[batchIndex] = bBase
	}
	if workers <= 0 {
		workers = parallelWorkerCountForMACs(batchCount, aRows, bCols, aCols)
	}
	outputRows, err := checkedProduct(batchCount, aRows, "matmul output rows")
	if err != nil {
		return nil, err
	}
	parallelFor(outputRows, workers, func(start, end int) {
		for outputIndex := start; outputIndex < end; outputIndex++ {
			batchIndex := outputIndex / aRows
			row := outputIndex % aRows
			aBase, bBase := aBatchBases[batchIndex], bBatchBases[batchIndex]
			resultBase := batchIndex * resultMatrixSize
			for column := 0; column < bCols; column++ {
				var sum float32
				for inner := 0; inner < aCols; inner++ {
					sum += a.data[aBase+row*aRowStride+inner*aColStride] * b.data[bBase+inner*bRowStride+column*bColStride]
				}
				result.data[resultBase+row*bCols+column] = sum
			}
		}
	})
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
	if matMulMACsAtLeast(a.shape[0], b.shape[1], a.shape[1]) && insyra.Config.GetAccelerationEnabled() {
		if hook := registeredDeviceMatMul(); hook != nil {
			data, err := hook(a.data, a.shape[0], a.shape[1], b.data, b.shape[0], b.shape[1])
			if err == nil && len(data) == a.shape[0]*b.shape[1] {
				if result, resultErr := newFloat32Tensor([]int{a.shape[0], b.shape[1]}, data); resultErr == nil {
					return result, nil
				}
			}
		}
	}
	return matMul2DWithWorkers(a, b, 0)
}

func matMul2DWithWorkers(a, b *Tensor, workers int) (*Tensor, error) {
	if a.shape[1] != b.shape[0] {
		return nil, fmt.Errorf("matmul shapes %v and %v are incompatible", a.shape, b.shape)
	}
	result, err := newFloat32Tensor([]int{a.shape[0], b.shape[1]}, make([]float32, a.shape[0]*b.shape[1]))
	if err != nil {
		return nil, err
	}
	if workers <= 0 {
		workers = parallelWorkerCountForMACs(a.shape[0], b.shape[1], a.shape[1])
	}
	parallelFor(a.shape[0], workers, func(start, end int) {
		for row := start; row < end; row++ {
			for column := 0; column < b.shape[1]; column++ {
				var sum float32
				for inner := 0; inner < a.shape[1]; inner++ {
					sum += a.data[row*a.shape[1]+inner] * b.data[inner*b.shape[1]+column]
				}
				result.data[row*b.shape[1]+column] = sum
			}
		}
	})
	return result, nil
}

// Add performs addition with numpy-style broadcasting for float32 and int64 tensors.
func Add(a, b *Tensor) (*Tensor, error) {
	if a != nil && b != nil && a.dtype == DTypeInt64 && b.dtype == DTypeInt64 {
		return tensorBroadcastInt64Binary(a, b, "add", func(left, right int64) (int64, error) { return left + right, nil })
	}
	return tensorBroadcastBinary(a, b, "add", func(left, right float32) float32 { return left + right })
}

// Sub performs subtraction with numpy-style broadcasting for float32 and int64 tensors.
func Sub(a, b *Tensor) (*Tensor, error) {
	if a != nil && b != nil && a.dtype == DTypeInt64 && b.dtype == DTypeInt64 {
		return tensorBroadcastInt64Binary(a, b, "sub", func(left, right int64) (int64, error) { return left - right, nil })
	}
	return tensorBroadcastBinary(a, b, "sub", func(left, right float32) float32 { return left - right })
}

// Mul performs multiplication with numpy-style broadcasting for float32 and int64 tensors.
func Mul(a, b *Tensor) (*Tensor, error) {
	if a != nil && b != nil && a.dtype == DTypeInt64 && b.dtype == DTypeInt64 {
		return tensorBroadcastInt64Binary(a, b, "mul", func(left, right int64) (int64, error) { return left * right, nil })
	}
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
	if left != nil && right != nil && left.dtype == DTypeInt64 && right.dtype == DTypeInt64 {
		return tensorBroadcastInt64Binary(left, right, "pow", func(base, exponent int64) (int64, error) {
			if exponent < 0 {
				return 0, fmt.Errorf("negative integer exponent %d is unsupported", exponent)
			}
			result := int64(1)
			for index := int64(0); index < exponent; index++ {
				result *= base
			}
			return result, nil
		})
	}
	return tensorBroadcastBinary(left, right, "pow", func(a, b float32) float32 {
		return float32(math.Pow(float64(a), float64(b)))
	})
}

// Clip clamps each float32 element between optional scalar bounds.
func Clip(input, minimum, maximum *Tensor) (*Tensor, error) {
	if err := requireFloat32(input, "clip input"); err != nil {
		return nil, err
	}
	minValue, err := clipBound(minimum, "min", float32(math.Inf(-1)))
	if err != nil {
		return nil, err
	}
	maxValue, err := clipBound(maximum, "max", float32(math.Inf(1)))
	if err != nil {
		return nil, err
	}
	if minValue > maxValue {
		return nil, fmt.Errorf("clip min %g is greater than max %g", minValue, maxValue)
	}
	result, err := newFloat32Tensor(input.shape, make([]float32, input.Len()))
	if err != nil {
		return nil, err
	}
	for index, value := range input.data {
		if value < minValue {
			value = minValue
		}
		if value > maxValue {
			value = maxValue
		}
		result.data[index] = value
	}
	return result, nil
}

func clipBound(value *Tensor, name string, fallback float32) (float32, error) {
	if value == nil {
		return fallback, nil
	}
	if err := requireFloat32(value, "clip "+name); err != nil {
		return 0, err
	}
	if value.Len() != 1 {
		return 0, fmt.Errorf("clip %s input has shape %v, want a scalar", name, value.shape)
	}
	return value.data[0], nil
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
	if input == nil {
		return nil, fmt.Errorf("reshape input is nil")
	}
	if !supportedTensorDType(input.dtype) {
		return nil, unsupportedDTypeError(input.dtype)
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
		if known == 0 || input.Len()%known != 0 {
			return nil, fmt.Errorf("reshape shape %v cannot infer a dimension from %d elements", requested, input.Len())
		}
		shape[unknown] = input.Len() / known
	}
	shapeCopy, strides, count, err := makeLayout(shape)
	if err != nil {
		return nil, err
	}
	if count != input.Len() {
		return nil, fmt.Errorf("reshape shape %v has %d elements, want %d", requested, count, input.Len())
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
// conversion for the paths used by categorical preprocessing. A FLOAT16 or
// BFLOAT16 target rounds the f32 value to that storage format and widens it
// back immediately, because Tensor computation remains f32.
func Cast(input *Tensor, to DType) (*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("cast input is nil")
	}
	if !supportedTensorDType(input.dtype) {
		return nil, unsupportedDTypeError(input.dtype)
	}
	if to == DTypeFloat16 || to == DTypeBFloat16 {
		source := input
		if source.dtype != DTypeFloat32 {
			var err error
			source, err = Cast(input, DTypeFloat32)
			if err != nil {
				return nil, fmt.Errorf("cast to dtype %s: %w", dtypeName(to), err)
			}
		}
		values := make([]float32, source.Len())
		for index, value := range source.data {
			if to == DTypeFloat16 {
				values[index] = f16BitsToFloat32(float32ToF16Bits(value))
			} else {
				values[index] = bf16BitsToFloat32(float32ToBF16Bits(value))
			}
		}
		return newFloat32Tensor(input.shape, values)
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
