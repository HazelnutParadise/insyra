package nn

import (
	"fmt"
	"math"
)

// Conv runs the existing 2-D NCHW convolution and records input, weight, and
// optional bias VJPs. The backward path mirrors Conv's direct index arithmetic
// so padding, stride, dilation, and groups have exactly the same semantics.
func (t *Tape) Conv(input, weights, bias *Tensor, options ...ConvOptions) (*Tensor, error) {
	output, err := Conv(input, weights, bias, options...)
	if err != nil {
		return nil, err
	}
	opts, err := resolvedConvOptions(options)
	if err != nil {
		return nil, err
	}
	inputs := []*Tensor{input, weights}
	if bias != nil {
		inputs = append(inputs, bias)
	}
	t.record("Conv", inputs, output, func(upstream *Tensor) ([]*Tensor, error) {
		return convVJP(input, weights, bias, opts, upstream)
	})
	return output, nil
}

// MaxPool runs the existing 2-D NCHW max pool and records a VJP that follows
// the forward's first-maximum tie rule.
func (t *Tape) MaxPool(input *Tensor, kernelShape []int, options ...PoolOptions) (*Tensor, error) {
	output, err := MaxPool(input, kernelShape, options...)
	if err != nil {
		return nil, err
	}
	opts, err := resolvedPoolOptions(options)
	if err != nil {
		return nil, err
	}
	t.record("MaxPool", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		gradient, err := maxPoolVJP(input, kernelShape, opts, upstream)
		return []*Tensor{gradient}, err
	})
	return output, nil
}

// AveragePool runs the existing 2-D NCHW average pool and records a VJP that
// uses the same per-output denominator as the forward.
func (t *Tape) AveragePool(input *Tensor, kernelShape []int, options ...PoolOptions) (*Tensor, error) {
	output, err := AveragePool(input, kernelShape, options...)
	if err != nil {
		return nil, err
	}
	opts, err := resolvedPoolOptions(options)
	if err != nil {
		return nil, err
	}
	t.record("AveragePool", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		gradient, err := averagePoolVJP(input, kernelShape, opts, upstream)
		return []*Tensor{gradient}, err
	})
	return output, nil
}

// GlobalAveragePool records the inverse spatial broadcast of the global mean.
func (t *Tape) GlobalAveragePool(input *Tensor) (*Tensor, error) {
	output, err := GlobalAveragePool(input)
	if err != nil {
		return nil, err
	}
	t.record("GlobalAveragePool", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		gradient, err := globalAveragePoolVJP(input, upstream)
		return []*Tensor{gradient}, err
	})
	return output, nil
}

// BatchNormalization records the inference-mode affine VJP. Mean and
// variance are constants loaded from the model, so no training-statistics
// gradient is exposed by this wrapper.
func (t *Tape) BatchNormalization(input, scale, bias, mean, variance *Tensor, epsilonValues ...float32) (*Tensor, error) {
	output, err := BatchNormalization(input, scale, bias, mean, variance, epsilonValues...)
	if err != nil {
		return nil, err
	}
	epsilon := float32(1e-5)
	if len(epsilonValues) == 1 {
		epsilon = epsilonValues[0]
	}
	t.record("BatchNormalization", []*Tensor{input, scale, bias}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return batchNormalizationVJP(input, scale, bias, mean, variance, epsilon, upstream)
	})
	return output, nil
}

// BatchNormalizationTraining applies BatchNorm's training semantics. The
// normalization uses the biased batch variance, while the running variance
// update uses the unbiased estimator, matching torch.nn.BatchNorm2d.
// options are [momentum, epsilon], with torch's defaults when omitted.
func (t *Tape) BatchNormalizationTraining(input, scale, bias, runningMean, runningVariance *Tensor, options ...float32) (*Tensor, error) {
	if t == nil {
		return nil, fmt.Errorf("training batch normalization tape is nil")
	}
	momentum, epsilon, err := batchNormalizationTrainingOptions(options)
	if err != nil {
		return nil, err
	}
	if err := requireFloat32(input, "training batch normalization input"); err != nil {
		return nil, err
	}
	for name, tensor := range map[string]*Tensor{
		"scale": scale, "bias": bias, "running mean": runningMean, "running variance": runningVariance,
	} {
		if err := requireFloat32(tensor, "training batch normalization "+name); err != nil {
			return nil, err
		}
	}
	if len(input.shape) < 2 {
		return nil, fmt.Errorf("training batch normalization requires input rank at least 2, got shape %v", input.shape)
	}
	channels := input.shape[1]
	channelShape := []int{channels}
	for name, tensor := range map[string]*Tensor{
		"scale": scale, "bias": bias, "running mean": runningMean, "running variance": runningVariance,
	} {
		if !sameShape(tensor.shape, channelShape) {
			return nil, fmt.Errorf("training batch normalization %s shape %v does not match channel shape %v", name, tensor.shape, channelShape)
		}
	}
	if momentum < 0 || momentum > 1 || math.IsNaN(float64(momentum)) || math.IsInf(float64(momentum), 0) {
		return nil, fmt.Errorf("training batch normalization momentum %g is invalid", momentum)
	}
	if epsilon < 0 || math.IsNaN(float64(epsilon)) || math.IsInf(float64(epsilon), 0) {
		return nil, fmt.Errorf("training batch normalization epsilon %g is invalid", epsilon)
	}
	countPerChannel := len(input.data) / channels
	if countPerChannel <= 1 {
		return nil, fmt.Errorf("training batch normalization needs more than one value per channel, got %d", countPerChannel)
	}
	mean := make([]float64, channels)
	for index, value := range input.data {
		mean[(index/input.strides[1])%channels] += float64(value)
	}
	for channel := range mean {
		mean[channel] /= float64(countPerChannel)
	}
	variance := make([]float64, channels)
	for index, value := range input.data {
		channel := (index / input.strides[1]) % channels
		delta := float64(value) - mean[channel]
		variance[channel] += delta * delta
	}
	for channel := range variance {
		variance[channel] /= float64(countPerChannel)
		unbiased := variance[channel] * float64(countPerChannel) / float64(countPerChannel-1)
		runningMean.data[channel] = (1-momentum)*runningMean.data[channel] + momentum*float32(mean[channel])
		runningVariance.data[channel] = (1-momentum)*runningVariance.data[channel] + momentum*float32(unbiased)
	}
	output, err := newZeroFloat32Tensor(input.shape)
	if err != nil {
		return nil, err
	}
	for index, value := range input.data {
		channel := (index / input.strides[1]) % channels
		normalized := (float64(value) - mean[channel]) / math.Sqrt(variance[channel]+float64(epsilon))
		output.data[index] = float32(normalized*float64(scale.data[channel]) + float64(bias.data[channel]))
	}
	t.record("BatchNormalizationTraining", []*Tensor{input, scale, bias}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return batchNormalizationTrainingVJP(input, scale, bias, mean, variance, epsilon, upstream)
	})
	return output, nil
}

// BatchNormTraining is a concise alias for BatchNormalizationTraining.
func (t *Tape) BatchNormTraining(input, scale, bias, runningMean, runningVariance *Tensor, options ...float32) (*Tensor, error) {
	return t.BatchNormalizationTraining(input, scale, bias, runningMean, runningVariance, options...)
}

func batchNormalizationTrainingOptions(options []float32) (momentum, epsilon float32, err error) {
	if len(options) > 2 {
		return 0, 0, fmt.Errorf("training batch normalization accepts at most momentum and epsilon")
	}
	momentum, epsilon = 0.1, 1e-5
	if len(options) >= 1 {
		momentum = options[0]
	}
	if len(options) == 2 {
		epsilon = options[1]
	}
	return momentum, epsilon, nil
}

func resolvedConvOptions(options []ConvOptions) (ConvOptions, error) {
	if len(options) > 1 {
		return ConvOptions{}, fmt.Errorf("autodiff conv accepts at most one options value")
	}
	opts := ConvOptions{Group: 1}
	if len(options) == 1 {
		opts = options[0]
		if opts.Group == 0 {
			opts.Group = 1
		}
	}
	return opts, nil
}

func resolvedPoolOptions(options []PoolOptions) (PoolOptions, error) {
	if len(options) > 1 {
		return PoolOptions{}, fmt.Errorf("autodiff pool accepts at most one options value")
	}
	if len(options) == 1 {
		return options[0], nil
	}
	return PoolOptions{}, nil
}

func convVJP(input, weights, bias *Tensor, opts ConvOptions, upstream *Tensor) ([]*Tensor, error) {
	if err := requireFloat32(upstream, "conv upstream"); err != nil {
		return nil, err
	}
	if len(input.shape) != 4 || len(weights.shape) != 4 {
		return nil, fmt.Errorf("conv VJP requires 4-D input and weights, got shapes %v and %v", input.shape, weights.shape)
	}
	window, err := resolve2DWindow(input.shape[2], input.shape[3], weights.shape[2], weights.shape[3], opts.Pads, opts.AutoPad, opts.Strides, opts.Dilations, "conv VJP")
	if err != nil {
		return nil, err
	}
	wantShape := []int{input.shape[0], weights.shape[0], window.outputH, window.outputW}
	if !sameShape(upstream.shape, wantShape) {
		return nil, fmt.Errorf("conv upstream shape %v does not match output shape %v", upstream.shape, wantShape)
	}

	inputChannels := input.shape[1]
	outputChannels := weights.shape[0]
	inputChannelsPerGroup := inputChannels / opts.Group
	outputChannelsPerGroup := outputChannels / opts.Group
	dInputValues := make([]float64, len(input.data))
	dWeightValues := make([]float64, len(weights.data))
	dBiasValues := make([]float64, outputChannels)
	for batch := 0; batch < input.shape[0]; batch++ {
		for outputChannel := 0; outputChannel < outputChannels; outputChannel++ {
			group := outputChannel / outputChannelsPerGroup
			inputChannelStart := group * inputChannelsPerGroup
			for outputRow := 0; outputRow < window.outputH; outputRow++ {
				for outputColumn := 0; outputColumn < window.outputW; outputColumn++ {
					upstreamIndex := ((batch*outputChannels+outputChannel)*window.outputH+outputRow)*window.outputW + outputColumn
					upstreamValue := float64(upstream.data[upstreamIndex])
					dBiasValues[outputChannel] += upstreamValue
					for inputChannel := 0; inputChannel < inputChannelsPerGroup; inputChannel++ {
						for kernelRow := 0; kernelRow < weights.shape[2]; kernelRow++ {
							inputRow := outputRow*window.strideH - window.padTop + kernelRow*window.dilationH
							if inputRow < 0 || inputRow >= input.shape[2] {
								continue
							}
							for kernelColumn := 0; kernelColumn < weights.shape[3]; kernelColumn++ {
								inputColumn := outputColumn*window.strideW - window.padLeft + kernelColumn*window.dilationW
								if inputColumn < 0 || inputColumn >= input.shape[3] {
									continue
								}
								inputIndex := ((batch*inputChannels+(inputChannelStart+inputChannel))*input.shape[2]+inputRow)*input.shape[3] + inputColumn
								weightIndex := ((outputChannel*weights.shape[1]+inputChannel)*weights.shape[2]+kernelRow)*weights.shape[3] + kernelColumn
								dInputValues[inputIndex] += upstreamValue * float64(weights.data[weightIndex])
								dWeightValues[weightIndex] += float64(input.data[inputIndex]) * upstreamValue
							}
						}
					}
				}
			}
		}
	}
	dInput, err := float64Gradient(input.shape, dInputValues)
	if err != nil {
		return nil, err
	}
	dWeights, err := float64Gradient(weights.shape, dWeightValues)
	if err != nil {
		return nil, err
	}
	gradients := []*Tensor{dInput, dWeights}
	if bias != nil {
		dBias, err := float64Gradient(bias.shape, dBiasValues)
		if err != nil {
			return nil, err
		}
		gradients = append(gradients, dBias)
	}
	return gradients, nil
}

func maxPoolVJP(input *Tensor, kernelShape []int, opts PoolOptions, upstream *Tensor) (*Tensor, error) {
	if err := requireFloat32(upstream, "max pool upstream"); err != nil {
		return nil, err
	}
	window, err := resolvePoolWindow(input, kernelShape, opts, "max pool VJP")
	if err != nil {
		return nil, err
	}
	wantShape := []int{input.shape[0], input.shape[1], window.outputH, window.outputW}
	if !sameShape(upstream.shape, wantShape) {
		return nil, fmt.Errorf("max pool upstream shape %v does not match output shape %v", upstream.shape, wantShape)
	}
	gradient, err := newZeroFloat32Tensor(input.shape)
	if err != nil {
		return nil, err
	}
	values := make([]float64, len(input.data))
	for batch := 0; batch < input.shape[0]; batch++ {
		for channel := 0; channel < input.shape[1]; channel++ {
			for outputRow := 0; outputRow < window.outputH; outputRow++ {
				for outputColumn := 0; outputColumn < window.outputW; outputColumn++ {
					maximum := float32(math.Inf(-1))
					maximumIndex := -1
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
								maximumIndex = inputIndex
							}
						}
					}
					if maximumIndex >= 0 {
						upstreamIndex := ((batch*input.shape[1]+channel)*window.outputH+outputRow)*window.outputW + outputColumn
						values[maximumIndex] += float64(upstream.data[upstreamIndex])
					}
				}
			}
		}
	}
	for index, value := range values {
		gradient.data[index] = float32(value)
	}
	return gradient, nil
}

func averagePoolVJP(input *Tensor, kernelShape []int, opts PoolOptions, upstream *Tensor) (*Tensor, error) {
	if err := requireFloat32(upstream, "average pool upstream"); err != nil {
		return nil, err
	}
	window, err := resolvePoolWindow(input, kernelShape, opts, "average pool VJP")
	if err != nil {
		return nil, err
	}
	wantShape := []int{input.shape[0], input.shape[1], window.outputH, window.outputW}
	if !sameShape(upstream.shape, wantShape) {
		return nil, fmt.Errorf("average pool upstream shape %v does not match output shape %v", upstream.shape, wantShape)
	}
	gradient, err := newZeroFloat32Tensor(input.shape)
	if err != nil {
		return nil, err
	}
	values := make([]float64, len(input.data))
	for batch := 0; batch < input.shape[0]; batch++ {
		for channel := 0; channel < input.shape[1]; channel++ {
			for outputRow := 0; outputRow < window.outputH; outputRow++ {
				for outputColumn := 0; outputColumn < window.outputW; outputColumn++ {
					validCount := 0
					for kernelRow := 0; kernelRow < kernelShape[0]; kernelRow++ {
						inputRow := outputRow*window.strideH - window.padTop + kernelRow
						if inputRow < 0 || inputRow >= input.shape[2] {
							continue
						}
						for kernelColumn := 0; kernelColumn < kernelShape[1]; kernelColumn++ {
							inputColumn := outputColumn*window.strideW - window.padLeft + kernelColumn
							if inputColumn >= 0 && inputColumn < input.shape[3] {
								validCount++
							}
						}
					}
					denominator := validCount
					if opts.CountIncludePad {
						denominator = kernelShape[0] * kernelShape[1]
					}
					if denominator == 0 {
						continue
					}
					upstreamIndex := ((batch*input.shape[1]+channel)*window.outputH+outputRow)*window.outputW + outputColumn
					contribution := float64(upstream.data[upstreamIndex]) / float64(denominator)
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
							values[inputIndex] += contribution
						}
					}
				}
			}
		}
	}
	for index, value := range values {
		gradient.data[index] = float32(value)
	}
	return gradient, nil
}

func globalAveragePoolVJP(input, upstream *Tensor) (*Tensor, error) {
	if err := requireFloat32(upstream, "global average pool upstream"); err != nil {
		return nil, err
	}
	wantShape := []int{input.shape[0], input.shape[1], 1, 1}
	if !sameShape(upstream.shape, wantShape) {
		return nil, fmt.Errorf("global average pool upstream shape %v does not match output shape %v", upstream.shape, wantShape)
	}
	gradient, err := newZeroFloat32Tensor(input.shape)
	if err != nil {
		return nil, err
	}
	spatialCount := input.shape[2] * input.shape[3]
	for batch := 0; batch < input.shape[0]; batch++ {
		for channel := 0; channel < input.shape[1]; channel++ {
			upstreamIndex := batch*input.shape[1] + channel
			contribution := upstream.data[upstreamIndex] / float32(spatialCount)
			base := upstreamIndex * spatialCount
			for index := 0; index < spatialCount; index++ {
				gradient.data[base+index] = contribution
			}
		}
	}
	return gradient, nil
}

func batchNormalizationVJP(input, scale, bias, mean, variance *Tensor, epsilon float32, upstream *Tensor) ([]*Tensor, error) {
	if err := requireFloat32(upstream, "batch normalization upstream"); err != nil {
		return nil, err
	}
	if !sameShape(upstream.shape, input.shape) {
		return nil, fmt.Errorf("batch normalization upstream shape %v does not match input shape %v", upstream.shape, input.shape)
	}
	dInputValues := make([]float64, len(input.data))
	dScaleValues := make([]float64, len(scale.data))
	dBiasValues := make([]float64, len(bias.data))
	channelStride := input.strides[1]
	for index, value := range input.data {
		channel := (index / channelStride) % input.shape[1]
		denominator := math.Sqrt(float64(variance.data[channel]) + float64(epsilon))
		normalized := (float64(value) - float64(mean.data[channel])) / denominator
		upstreamValue := float64(upstream.data[index])
		dInputValues[index] = upstreamValue * float64(scale.data[channel]) / denominator
		dScaleValues[channel] += upstreamValue * normalized
		dBiasValues[channel] += upstreamValue
	}
	dInput, err := float64Gradient(input.shape, dInputValues)
	if err != nil {
		return nil, err
	}
	dScale, err := float64Gradient(scale.shape, dScaleValues)
	if err != nil {
		return nil, err
	}
	dBias, err := float64Gradient(bias.shape, dBiasValues)
	if err != nil {
		return nil, err
	}
	return []*Tensor{dInput, dScale, dBias}, nil
}

func batchNormalizationTrainingVJP(input, scale, bias *Tensor, mean, variance []float64, epsilon float32, upstream *Tensor) ([]*Tensor, error) {
	if err := requireFloat32(upstream, "training batch normalization upstream"); err != nil {
		return nil, err
	}
	if !sameShape(upstream.shape, input.shape) {
		return nil, fmt.Errorf("training batch normalization upstream shape %v does not match input shape %v", upstream.shape, input.shape)
	}
	channels := input.shape[1]
	countPerChannel := len(input.data) / channels
	dInputValues := make([]float64, len(input.data))
	dScaleValues := make([]float64, channels)
	dBiasValues := make([]float64, channels)
	sums := make([]float64, channels)
	weightedSums := make([]float64, channels)
	for index, value := range input.data {
		channel := (index / input.strides[1]) % channels
		xhat := (float64(value) - mean[channel]) / math.Sqrt(variance[channel]+float64(epsilon))
		dy := float64(upstream.data[index])
		dScaleValues[channel] += dy * xhat
		dBiasValues[channel] += dy
		sums[channel] += dy
		weightedSums[channel] += dy * xhat
	}
	for index, value := range input.data {
		channel := (index / input.strides[1]) % channels
		xhat := (float64(value) - mean[channel]) / math.Sqrt(variance[channel]+float64(epsilon))
		dy := float64(upstream.data[index])
		invStd := 1 / math.Sqrt(variance[channel]+float64(epsilon))
		dInputValues[index] = float64(scale.data[channel]) * invStd * (float64(countPerChannel)*dy - sums[channel] - xhat*weightedSums[channel]) / float64(countPerChannel)
	}
	dInput, err := float64Gradient(input.shape, dInputValues)
	if err != nil {
		return nil, err
	}
	dScale, err := float64Gradient(scale.shape, dScaleValues)
	if err != nil {
		return nil, err
	}
	dBias, err := float64Gradient(bias.shape, dBiasValues)
	if err != nil {
		return nil, err
	}
	return []*Tensor{dInput, dScale, dBias}, nil
}

func float64Gradient(shape []int, values []float64) (*Tensor, error) {
	data := make([]float32, len(values))
	for index, value := range values {
		data[index] = float32(value)
	}
	return newFloat32Tensor(shape, data)
}
