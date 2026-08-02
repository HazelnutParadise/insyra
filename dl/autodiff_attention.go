package dl

import (
	"fmt"
	"math"
)

func matMulVJPBatched(a, b, upstream *Tensor) ([]*Tensor, error) {
	aPromoted, aVector, err := promoteMatMulInput(a, true)
	if err != nil {
		return nil, err
	}
	bPromoted, bVector, err := promoteMatMulInput(b, false)
	if err != nil {
		return nil, err
	}
	if len(aPromoted.shape) < 2 || len(bPromoted.shape) < 2 {
		return nil, fmt.Errorf("matmul VJP requires inputs with rank at least 1, got shapes %v and %v", a.shape, b.shape)
	}
	aBatch := aPromoted.shape[:len(aPromoted.shape)-2]
	bBatch := bPromoted.shape[:len(bPromoted.shape)-2]
	batchShape, err := tensorBroadcastShape(aBatch, bBatch)
	if err != nil {
		return nil, fmt.Errorf("matmul VJP batch shapes %v and %v: %w", aBatch, bBatch, err)
	}
	rows, inner := aPromoted.shape[len(aPromoted.shape)-2], aPromoted.shape[len(aPromoted.shape)-1]
	if inner != bPromoted.shape[len(bPromoted.shape)-2] {
		return nil, fmt.Errorf("matmul VJP shapes %v and %v are incompatible", a.shape, b.shape)
	}
	columns := bPromoted.shape[len(bPromoted.shape)-1]
	promotedOutputShape := append(append([]int(nil), batchShape...), rows, columns)
	outputShape := append([]int(nil), promotedOutputShape...)
	if aVector {
		outputShape = append(outputShape[:len(outputShape)-2], outputShape[len(outputShape)-1])
	}
	if bVector {
		outputShape = outputShape[:len(outputShape)-1]
	}
	if !sameShape(upstream.shape, outputShape) {
		return nil, fmt.Errorf("matmul upstream shape %v does not match output shape %v", upstream.shape, outputShape)
	}
	upstreamPromoted, err := Reshape(upstream, promotedOutputShape)
	if err != nil {
		return nil, fmt.Errorf("matmul VJP promote upstream: %w", err)
	}
	bTranspose, err := transposeLastTwo(bPromoted)
	if err != nil {
		return nil, err
	}
	aTranspose, err := transposeLastTwo(aPromoted)
	if err != nil {
		return nil, err
	}
	dAPromoted, err := MatMul(upstreamPromoted, bTranspose)
	if err != nil {
		return nil, fmt.Errorf("matmul VJP dA: %w", err)
	}
	dBPromoted, err := MatMul(aTranspose, upstreamPromoted)
	if err != nil {
		return nil, fmt.Errorf("matmul VJP dB: %w", err)
	}
	dA, err := reduceBroadcastGradientTape(dAPromoted, aPromoted.shape)
	if err != nil {
		return nil, fmt.Errorf("matmul VJP reduce dA: %w", err)
	}
	dB, err := reduceBroadcastGradientTape(dBPromoted, bPromoted.shape)
	if err != nil {
		return nil, fmt.Errorf("matmul VJP reduce dB: %w", err)
	}
	if aVector {
		dA, err = Reshape(dA, a.shape)
		if err != nil {
			return nil, err
		}
	}
	if bVector {
		dB, err = Reshape(dB, b.shape)
		if err != nil {
			return nil, err
		}
	}
	return []*Tensor{dA, dB}, nil
}

func promoteMatMulInput(input *Tensor, left bool) (*Tensor, bool, error) {
	if err := requireFloat32(input, "matmul VJP input"); err != nil {
		return nil, false, err
	}
	if len(input.shape) != 1 {
		return input, false, nil
	}
	shape := []int{input.shape[0], 1}
	if left {
		shape = []int{1, input.shape[0]}
	}
	promoted, err := Reshape(input, shape)
	return promoted, true, err
}

func transposeLastTwo(input *Tensor) (*Tensor, error) {
	if len(input.shape) < 2 {
		return nil, fmt.Errorf("transpose-last-two requires rank at least 2, got %v", input.shape)
	}
	perm := make([]int, len(input.shape))
	for index := range perm {
		perm[index] = index
	}
	perm[len(perm)-2], perm[len(perm)-1] = perm[len(perm)-1], perm[len(perm)-2]
	return Transpose(input, perm)
}

func mulVJP(left, right, upstream *Tensor) ([]*Tensor, error) {
	return broadcastBinaryVJP(left, right, upstream,
		func(a, b float32) float32 { return b },
		func(a, b float32) float32 { return a })
}

func divVJP(left, right, upstream *Tensor) ([]*Tensor, error) {
	return broadcastBinaryVJP(left, right, upstream,
		func(a, b float32) float32 { return 1 / b },
		func(a, b float32) float32 { return -a / (b * b) })
}

func broadcastBinaryVJP(left, right, upstream *Tensor, leftDerivative, rightDerivative func(float32, float32) float32) ([]*Tensor, error) {
	if err := requireFloat32(left, "binary VJP left operand"); err != nil {
		return nil, err
	}
	if err := requireFloat32(right, "binary VJP right operand"); err != nil {
		return nil, err
	}
	if err := requireFloat32(upstream, "binary VJP upstream"); err != nil {
		return nil, err
	}
	shape, err := tensorBroadcastShape(left.shape, right.shape)
	if err != nil {
		return nil, err
	}
	if !sameShape(shape, upstream.shape) {
		return nil, fmt.Errorf("binary VJP upstream shape %v does not match output shape %v", upstream.shape, shape)
	}
	leftFull, err := newZeroFloat32Tensor(shape)
	if err != nil {
		return nil, err
	}
	rightFull, err := newZeroFloat32Tensor(shape)
	if err != nil {
		return nil, err
	}
	leftStrides := alignedBroadcastStrides(left, len(shape))
	rightStrides := alignedBroadcastStrides(right, len(shape))
	for outputIndex := range upstream.data {
		leftIndex, rightIndex := broadcastIndex(outputIndex, upstream.strides, leftStrides, rightStrides)
		a, b := left.data[leftIndex], right.data[rightIndex]
		leftFull.data[outputIndex] = upstream.data[outputIndex] * leftDerivative(a, b)
		rightFull.data[outputIndex] = upstream.data[outputIndex] * rightDerivative(a, b)
	}
	leftGradient, err := reduceBroadcastGradientTape(leftFull, left.shape)
	if err != nil {
		return nil, err
	}
	rightGradient, err := reduceBroadcastGradientTape(rightFull, right.shape)
	if err != nil {
		return nil, err
	}
	return []*Tensor{leftGradient, rightGradient}, nil
}

func softmaxVJP(output, upstream *Tensor, axis int) *Tensor {
	gradient, _ := newZeroFloat32Tensor(output.shape)
	axisSize, inner, outer := output.shape[axis], 1, 1
	for index := axis + 1; index < len(output.shape); index++ {
		inner *= output.shape[index]
	}
	for index := 0; index < axis; index++ {
		outer *= output.shape[index]
	}
	for outerIndex := 0; outerIndex < outer; outerIndex++ {
		for innerIndex := 0; innerIndex < inner; innerIndex++ {
			base := outerIndex*axisSize*inner + innerIndex
			var dot float64
			for axisIndex := 0; axisIndex < axisSize; axisIndex++ {
				index := base + axisIndex*inner
				dot += float64(upstream.data[index]) * float64(output.data[index])
			}
			for axisIndex := 0; axisIndex < axisSize; axisIndex++ {
				index := base + axisIndex*inner
				gradient.data[index] = output.data[index] * (upstream.data[index] - float32(dot))
			}
		}
	}
	return gradient
}

func layerNormalizationVJP(input, scale, bias *Tensor, axis int, epsilon float32, upstream *Tensor) ([]*Tensor, error) {
	if err := requireFloat32(upstream, "layer normalization upstream"); err != nil {
		return nil, err
	}
	if !sameShape(upstream.shape, input.shape) {
		return nil, fmt.Errorf("layer normalization upstream shape %v does not match input shape %v", upstream.shape, input.shape)
	}
	normalizedShape := input.shape[axis:]
	groupSize := 1
	for _, dimension := range normalizedShape {
		groupSize *= dimension
	}
	groups := len(input.data) / groupSize
	dInput, err := newZeroFloat32Tensor(input.shape)
	if err != nil {
		return nil, err
	}
	dScale, err := newZeroFloat32Tensor(scale.shape)
	if err != nil {
		return nil, err
	}
	dBias, err := newZeroFloat32Tensor(bias.shape)
	if err != nil {
		return nil, err
	}
	dScaleSums := make([]float64, len(scale.data))
	dBiasSums := make([]float64, len(bias.data))
	for group := 0; group < groups; group++ {
		base := group * groupSize
		var mean float64
		for index := 0; index < groupSize; index++ {
			mean += float64(input.data[base+index])
		}
		mean /= float64(groupSize)
		var variance float64
		for index := 0; index < groupSize; index++ {
			delta := float64(input.data[base+index]) - mean
			variance += delta * delta
		}
		variance /= float64(groupSize)
		denominator := math.Sqrt(variance + float64(epsilon))
		var sumDy, sumDyNormalized float64
		for index := 0; index < groupSize; index++ {
			normalized := (float64(input.data[base+index]) - mean) / denominator
			dOutput := float64(upstream.data[base+index])
			dY := dOutput * float64(scale.data[index])
			dScaleSums[index] += dOutput * normalized
			dBiasSums[index] += dOutput
			sumDy += dY
			sumDyNormalized += dY * normalized
		}
		for index := 0; index < groupSize; index++ {
			normalized := (float64(input.data[base+index]) - mean) / denominator
			dx := 1 / denominator * (float64(upstream.data[base+index])*float64(scale.data[index]) - sumDy/float64(groupSize) - normalized*sumDyNormalized/float64(groupSize))
			dInput.data[base+index] = float32(dx)
		}
	}
	for index := range dScale.data {
		dScale.data[index] = float32(dScaleSums[index])
		dBias.data[index] = float32(dBiasSums[index])
	}
	return []*Tensor{dInput, dScale, dBias}, nil
}

func geluVJP(input, upstream *Tensor) *Tensor {
	gradient, _ := newZeroFloat32Tensor(input.shape)
	const sqrtTwo = 1.4142135623730950488
	const sqrtTwoPi = 2.5066282746310005024
	for index, value := range input.data {
		x := float64(value)
		cdf := 0.5 * (1 + math.Erf(x/sqrtTwo))
		density := math.Exp(-0.5*x*x) / sqrtTwoPi
		gradient.data[index] = upstream.data[index] * float32(cdf+x*density)
	}
	return gradient
}

func erfVJP(input, upstream *Tensor) *Tensor {
	gradient, _ := newZeroFloat32Tensor(input.shape)
	coefficient := 2 / math.Sqrt(math.Pi)
	for index, value := range input.data {
		gradient.data[index] = upstream.data[index] * float32(coefficient*math.Exp(-float64(value)*float64(value)))
	}
	return gradient
}

func sqrtVJP(output, upstream *Tensor) *Tensor {
	gradient, _ := newZeroFloat32Tensor(output.shape)
	for index, value := range output.data {
		gradient.data[index] = upstream.data[index] / (2 * value)
	}
	return gradient
}

func powVJP(left, right, output, upstream *Tensor) ([]*Tensor, error) {
	if err := requireFloat32(upstream, "pow upstream"); err != nil {
		return nil, err
	}
	shape, err := tensorBroadcastShape(left.shape, right.shape)
	if err != nil {
		return nil, err
	}
	if !sameShape(shape, upstream.shape) {
		return nil, fmt.Errorf("pow upstream shape %v does not match output shape %v", upstream.shape, shape)
	}
	leftFull, err := newZeroFloat32Tensor(shape)
	if err != nil {
		return nil, err
	}
	rightFull, err := newZeroFloat32Tensor(shape)
	if err != nil {
		return nil, err
	}
	leftStrides := alignedBroadcastStrides(left, len(shape))
	rightStrides := alignedBroadcastStrides(right, len(shape))
	for index := range upstream.data {
		leftIndex, rightIndex := broadcastIndex(index, upstream.strides, leftStrides, rightStrides)
		base, exponent := float64(left.data[leftIndex]), float64(right.data[rightIndex])
		if base <= 0 {
			return nil, fmt.Errorf("backward Pow exponent gradient requires positive base, got %g", base)
		}
		leftFull.data[index] = upstream.data[index] * float32(exponent*math.Pow(base, exponent-1))
		rightFull.data[index] = upstream.data[index] * output.data[index] * float32(math.Log(base))
	}
	leftGradient, err := reduceBroadcastGradientTape(leftFull, left.shape)
	if err != nil {
		return nil, err
	}
	rightGradient, err := reduceBroadcastGradientTape(rightFull, right.shape)
	if err != nil {
		return nil, err
	}
	return []*Tensor{leftGradient, rightGradient}, nil
}

func reduceAxes(axes []int, rank int) ([]int, error) {
	if len(axes) == 0 {
		axes = make([]int, rank)
		for index := range axes {
			axes[index] = index
		}
	}
	resolved := make([]int, 0, len(axes))
	seen := make(map[int]struct{}, len(axes))
	for _, axis := range axes {
		resolvedAxis, err := normalizeAxis(axis, rank, "autodiff reduce mean")
		if err != nil {
			return nil, err
		}
		if _, exists := seen[resolvedAxis]; exists {
			return nil, fmt.Errorf("autodiff reduce mean axis %d is repeated", axis)
		}
		seen[resolvedAxis] = struct{}{}
		resolved = append(resolved, resolvedAxis)
	}
	return resolved, nil
}

func reduceMeanVJP(input *Tensor, axes []int, keepdims bool, upstream *Tensor) *Tensor {
	gradient, _ := newZeroFloat32Tensor(input.shape)
	axisSet := make(map[int]struct{}, len(axes))
	count := 1
	for _, axis := range axes {
		axisSet[axis] = struct{}{}
		count *= input.shape[axis]
	}
	for inputIndex := range input.data {
		coordinates := linearCoordinates(inputIndex, input.shape, input.strides)
		outputCoordinates := make([]int, 0, len(upstream.shape))
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
			outputIndex += coordinate * upstream.strides[axis]
		}
		gradient.data[inputIndex] = upstream.data[outputIndex] / float32(count)
	}
	return gradient
}

func resolvedTransposePermutation(input *Tensor, perms ...[]int) ([]int, error) {
	if input == nil {
		return nil, fmt.Errorf("autodiff transpose input is nil")
	}
	if len(perms) > 1 {
		return nil, fmt.Errorf("autodiff transpose accepts at most one permutation")
	}
	perm := make([]int, len(input.shape))
	if len(perms) == 0 || len(perms[0]) == 0 {
		for index := range perm {
			perm[index] = len(perm) - index - 1
		}
	} else {
		perm = append([]int(nil), perms[0]...)
	}
	if _, err := Transpose(input, perm); err != nil {
		return nil, err
	}
	return perm, nil
}

func invertPermutation(perm []int) []int {
	inverse := make([]int, len(perm))
	for outputAxis, inputAxis := range perm {
		inverse[inputAxis] = outputAxis
	}
	return inverse
}

func sliceAxisIndices(shape []int, starts, ends, axes, steps []int64) ([][]int, error) {
	if len(starts) != len(ends) {
		return nil, fmt.Errorf("slice starts length %d does not match ends length %d", len(starts), len(ends))
	}
	if len(axes) == 0 {
		axes = make([]int64, len(starts))
		for index := range axes {
			axes[index] = int64(index)
		}
	}
	if len(steps) == 0 {
		steps = make([]int64, len(starts))
		for index := range steps {
			steps[index] = 1
		}
	}
	if len(axes) != len(starts) || len(steps) != len(starts) {
		return nil, fmt.Errorf("slice control lengths do not match starts length %d", len(starts))
	}
	indices := make([][]int, len(shape))
	for axis, dimension := range shape {
		indices[axis] = make([]int, dimension)
		for index := range indices[axis] {
			indices[axis][index] = index
		}
	}
	seen := make(map[int]struct{}, len(axes))
	for index, rawAxis := range axes {
		axis, err := normalizeAxis(int(rawAxis), len(shape), "autodiff slice")
		if err != nil {
			return nil, err
		}
		if _, exists := seen[axis]; exists {
			return nil, fmt.Errorf("autodiff slice axis %d is repeated", rawAxis)
		}
		if steps[index] == 0 {
			return nil, fmt.Errorf("autodiff slice step for axis %d is zero", axis)
		}
		seen[axis] = struct{}{}
		indices[axis] = sliceIndices(shape[axis], starts[index], ends[index], steps[index])
	}
	return indices, nil
}

func sliceVJP(input *Tensor, starts, ends, axes, steps []int64, upstream *Tensor) (*Tensor, error) {
	indices, err := sliceAxisIndices(input.shape, starts, ends, axes, steps)
	if err != nil {
		return nil, err
	}
	gradient, err := newZeroFloat32Tensor(input.shape)
	if err != nil {
		return nil, err
	}
	for outputIndex := range upstream.data {
		coordinates := linearCoordinates(outputIndex, upstream.shape, upstream.strides)
		inputIndex := 0
		for axis, coordinate := range coordinates {
			inputIndex += indices[axis][coordinate] * input.strides[axis]
		}
		gradient.data[inputIndex] += upstream.data[outputIndex]
	}
	return gradient, nil
}

func concatVJP(inputs []*Tensor, axis int, upstream *Tensor) ([]*Tensor, error) {
	gradients := make([]*Tensor, len(inputs))
	offset := 0
	for index, input := range inputs {
		gradient, err := newZeroFloat32Tensor(input.shape)
		if err != nil {
			return nil, err
		}
		for gradientIndex := range gradient.data {
			coordinates := linearCoordinates(gradientIndex, gradient.shape, gradient.strides)
			coordinates[axis] += offset
			upstreamIndex := 0
			for dimension, coordinate := range coordinates {
				upstreamIndex += coordinate * upstream.strides[dimension]
			}
			gradient.data[gradientIndex] = upstream.data[upstreamIndex]
		}
		gradients[index] = gradient
		offset += input.shape[axis]
	}
	return gradients, nil
}

func splitVJP(input *Tensor, axis, offset int, upstream *Tensor) *Tensor {
	gradient, _ := newZeroFloat32Tensor(input.shape)
	for upstreamIndex := range upstream.data {
		coordinates := linearCoordinates(upstreamIndex, upstream.shape, upstream.strides)
		coordinates[axis] += offset
		inputIndex := 0
		for dimension, coordinate := range coordinates {
			inputIndex += coordinate * input.strides[dimension]
		}
		gradient.data[inputIndex] += upstream.data[upstreamIndex]
	}
	return gradient
}
