package nn

import (
	"fmt"
	"math"
	"math/rand"
)

// Tape records differentiable operations and replays their vector-Jacobian
// products in reverse order. It is deliberately separate from Model.Run:
// inference kernels remain plain functions, and the tape is another caller.
type Tape struct {
	ops    []tapeOp
	params []*Parameter
	marked map[*Tensor]*Parameter
	grads  map[*Tensor]*Tensor
	rng    *rand.Rand
}

// Parameter is a tensor marked for gradient retrieval and SGD updates.
type Parameter struct {
	value    *Tensor
	grad     *Tensor
	adamM    []float32
	adamV    []float32
	adamStep uint64
}

// Value returns the parameter tensor used by the tape wrappers.
func (p *Parameter) Value() *Tensor {
	if p == nil {
		return nil
	}
	return p.value
}

// Grad returns the parameter's most recently computed gradient.
func (p *Parameter) Grad() *Tensor {
	if p == nil || p.grad == nil {
		return nil
	}
	gradient, _ := copyTensor(p.grad)
	return gradient
}

type tapeOp struct {
	name   string
	inputs []*Tensor
	output *Tensor
	vjp    func(*Tensor) ([]*Tensor, error)
}

// NewTape creates an empty reverse-mode tape. Dropout uses the tape-owned RNG;
// a seed makes masks reproducible, and the default seed is deterministic.
func NewTape(seed ...int64) *Tape {
	const defaultSeed = int64(1)
	rngSeed := defaultSeed
	if len(seed) > 0 {
		rngSeed = seed[0]
	}
	return &Tape{
		marked: make(map[*Tensor]*Parameter),
		grads:  make(map[*Tensor]*Tensor),
		rng:    rand.New(rand.NewSource(rngSeed)),
	}
}

// Param marks a float32 tensor for gradient retrieval and SGD updates.
func (t *Tape) Param(value *Tensor) (*Parameter, error) {
	if err := requireFloat32(value, "parameter"); err != nil {
		return nil, err
	}
	if parameter := t.marked[value]; parameter != nil {
		return parameter, nil
	}
	parameter := &Parameter{value: value}
	t.marked[value] = parameter
	t.params = append(t.params, parameter)
	return parameter, nil
}

// MatMul runs the existing MatMul kernel and records its VJP. Rank-1 inputs
// use the same vector promotion and squeezing rules as the forward kernel.
func (t *Tape) MatMul(a, b *Tensor) (*Tensor, error) {
	if a == nil || b == nil {
		return nil, fmt.Errorf("autodiff matmul inputs must not be nil")
	}
	output, err := MatMul(a, b)
	if err != nil {
		return nil, err
	}
	t.record("MatMul", []*Tensor{a, b}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return matMulVJP(a, b, upstream)
	})
	return output, nil
}

// Mul runs the existing broadcast multiplication kernel and records its VJP.
func (t *Tape) Mul(a, b *Tensor) (*Tensor, error) {
	output, err := Mul(a, b)
	if err != nil {
		return nil, err
	}
	t.record("Mul", []*Tensor{a, b}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return mulVJP(a, b, upstream)
	})
	return output, nil
}

// Div runs the existing broadcast division kernel and records its VJP.
func (t *Tape) Div(a, b *Tensor) (*Tensor, error) {
	output, err := Div(a, b)
	if err != nil {
		return nil, err
	}
	t.record("Div", []*Tensor{a, b}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return divVJP(a, b, upstream)
	})
	return output, nil
}

// Add runs the existing broadcast Add kernel and records its VJP.
func (t *Tape) Add(a, b *Tensor) (*Tensor, error) {
	output, err := Add(a, b)
	if err != nil {
		return nil, err
	}
	t.record("Add", []*Tensor{a, b}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return addVJP(a, b, upstream)
	})
	return output, nil
}

// Relu runs the existing Relu kernel and records its VJP.
func (t *Tape) Relu(input *Tensor) (*Tensor, error) {
	output, err := Relu(input)
	if err != nil {
		return nil, err
	}
	t.record("Relu", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return []*Tensor{reluVJP(input, output, upstream)}, nil
	})
	return output, nil
}

// Sigmoid runs the existing Sigmoid kernel and records its VJP.
func (t *Tape) Sigmoid(input *Tensor) (*Tensor, error) {
	output, err := Sigmoid(input)
	if err != nil {
		return nil, err
	}
	t.record("Sigmoid", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return []*Tensor{sigmoidVJP(output, upstream)}, nil
	})
	return output, nil
}

// Tanh runs the existing Tanh kernel and records its VJP.
func (t *Tape) Tanh(input *Tensor) (*Tensor, error) {
	output, err := Tanh(input)
	if err != nil {
		return nil, err
	}
	t.record("Tanh", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return []*Tensor{tanhVJP(output, upstream)}, nil
	})
	return output, nil
}

// Softmax runs the existing axis-softmax kernel and records its VJP.
func (t *Tape) Softmax(input *Tensor, axes ...int) (*Tensor, error) {
	if err := requireFloat32(input, "autodiff softmax input"); err != nil {
		return nil, err
	}
	axis := -1
	if len(axes) > 1 {
		return nil, fmt.Errorf("autodiff softmax accepts at most one axis")
	}
	if len(axes) == 1 {
		axis = axes[0]
	}
	axis, err := normalizeAxis(axis, len(input.shape), "autodiff softmax")
	if err != nil {
		return nil, err
	}
	output, err := Softmax(input, axis)
	if err != nil {
		return nil, err
	}
	t.record("Softmax", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return []*Tensor{softmaxVJP(output, upstream, axis)}, nil
	})
	return output, nil
}

// LayerNormalization runs suffix LayerNormalization and records input, scale,
// and bias gradients.
func (t *Tape) LayerNormalization(input, scale, bias *Tensor, axis int, epsilon float32) (*Tensor, error) {
	if err := requireFloat32(input, "autodiff layer normalization input"); err != nil {
		return nil, err
	}
	resolvedAxis, err := normalizeAxis(axis, len(input.shape), "autodiff layer normalization")
	if err != nil {
		return nil, err
	}
	output, err := LayerNormalization(input, scale, bias, resolvedAxis, epsilon)
	if err != nil {
		return nil, err
	}
	t.record("LayerNormalization", []*Tensor{input, scale, bias}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return layerNormalizationVJP(input, scale, bias, resolvedAxis, epsilon, upstream)
	})
	return output, nil
}

// Gelu records the exact erf-form VJP. The tanh approximation is intentionally
// refused because this training slice proves only the exact form.
func (t *Tape) Gelu(input *Tensor, approximate ...string) (*Tensor, error) {
	if len(approximate) > 1 {
		return nil, fmt.Errorf("autodiff gelu accepts at most one approximate mode")
	}
	mode := "none"
	if len(approximate) == 1 {
		mode = approximate[0]
	}
	if mode != "none" {
		return nil, fmt.Errorf("backward Gelu approximate mode %q has no VJP", mode)
	}
	output, err := Gelu(input, mode)
	if err != nil {
		return nil, err
	}
	t.record("Gelu", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return []*Tensor{geluVJP(input, upstream)}, nil
	})
	return output, nil
}

// Erf records the derivative of the error function.
func (t *Tape) Erf(input *Tensor) (*Tensor, error) {
	output, err := Erf(input)
	if err != nil {
		return nil, err
	}
	t.record("Erf", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return []*Tensor{erfVJP(input, upstream)}, nil
	})
	return output, nil
}

// Sqrt records the square-root VJP.
func (t *Tape) Sqrt(input *Tensor) (*Tensor, error) {
	output, err := Sqrt(input)
	if err != nil {
		return nil, err
	}
	t.record("Sqrt", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return []*Tensor{sqrtVJP(output, upstream)}, nil
	})
	return output, nil
}

// Pow records gradients for both broadcast operands.
func (t *Tape) Pow(left, right *Tensor) (*Tensor, error) {
	output, err := Pow(left, right)
	if err != nil {
		return nil, err
	}
	t.record("Pow", []*Tensor{left, right}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return powVJP(left, right, output, upstream)
	})
	return output, nil
}

// ReduceMean records a mean reduction over the selected axes.
func (t *Tape) ReduceMean(input *Tensor, axes []int, keepdims bool) (*Tensor, error) {
	if err := requireFloat32(input, "autodiff reduce mean input"); err != nil {
		return nil, err
	}
	resolvedAxes, err := reduceAxes(axes, len(input.shape))
	if err != nil {
		return nil, err
	}
	output, err := ReduceMean(input, resolvedAxes, keepdims)
	if err != nil {
		return nil, err
	}
	t.record("ReduceMean", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		return []*Tensor{reduceMeanVJP(input, resolvedAxes, keepdims, upstream)}, nil
	})
	return output, nil
}

// Reshape records the inverse reshape VJP.
func (t *Tape) Reshape(input *Tensor, shape []int) (*Tensor, error) {
	output, err := Reshape(input, shape)
	if err != nil {
		return nil, err
	}
	t.record("Reshape", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		gradient, err := Reshape(upstream, input.shape)
		return []*Tensor{gradient}, err
	})
	return output, nil
}

// Flatten records the inverse reshape VJP for Flatten.
func (t *Tape) Flatten(input *Tensor, axes ...int) (*Tensor, error) {
	output, err := Flatten(input, axes...)
	if err != nil {
		return nil, err
	}
	t.record("Flatten", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		gradient, err := Reshape(upstream, input.shape)
		return []*Tensor{gradient}, err
	})
	return output, nil
}

// Transpose records the inverse permutation VJP.
func (t *Tape) Transpose(input *Tensor, perms ...[]int) (*Tensor, error) {
	if err := requireFloat32(input, "autodiff transpose input"); err != nil {
		return nil, err
	}
	perm, err := resolvedTransposePermutation(input, perms...)
	if err != nil {
		return nil, err
	}
	output, err := Transpose(input, perm)
	if err != nil {
		return nil, err
	}
	inverse := invertPermutation(perm)
	t.record("Transpose", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		gradient, err := Transpose(upstream, inverse)
		return []*Tensor{gradient}, err
	})
	return output, nil
}

// Unsqueeze records the inverse reshape VJP.
func (t *Tape) Unsqueeze(input *Tensor, axes []int) (*Tensor, error) {
	if err := requireFloat32(input, "autodiff unsqueeze input"); err != nil {
		return nil, err
	}
	output, err := Unsqueeze(input, axes)
	if err != nil {
		return nil, err
	}
	t.record("Unsqueeze", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		gradient, err := Reshape(upstream, input.shape)
		return []*Tensor{gradient}, err
	})
	return output, nil
}

// Squeeze records the inverse reshape VJP.
func (t *Tape) Squeeze(input *Tensor, axes []int) (*Tensor, error) {
	if err := requireFloat32(input, "autodiff squeeze input"); err != nil {
		return nil, err
	}
	output, err := Squeeze(input, axes)
	if err != nil {
		return nil, err
	}
	t.record("Squeeze", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		gradient, err := Reshape(upstream, input.shape)
		return []*Tensor{gradient}, err
	})
	return output, nil
}

// Slice records a scatter-back VJP for the selected input elements.
func (t *Tape) Slice(input *Tensor, starts, ends, axes, steps []int64) (*Tensor, error) {
	if err := requireFloat32(input, "autodiff slice input"); err != nil {
		return nil, err
	}
	output, err := Slice(input, starts, ends, axes, steps)
	if err != nil {
		return nil, err
	}
	t.record("Slice", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		gradient, err := sliceVJP(input, starts, ends, axes, steps, upstream)
		return []*Tensor{gradient}, err
	})
	return output, nil
}

// Concat records one slice VJP per participating input.
func (t *Tape) Concat(inputs []*Tensor, axis int) (*Tensor, error) {
	for index, input := range inputs {
		if err := requireFloat32(input, fmt.Sprintf("autodiff concat input %d", index)); err != nil {
			return nil, err
		}
	}
	output, err := Concat(inputs, axis)
	if err != nil {
		return nil, err
	}
	resolvedAxis, err := normalizeAxis(axis, len(output.shape), "autodiff concat")
	if err != nil {
		return nil, err
	}
	t.record("Concat", inputs, output, func(upstream *Tensor) ([]*Tensor, error) {
		return concatVJP(inputs, resolvedAxis, upstream)
	})
	return output, nil
}

// Split records one reverse slice for each output. Gradients from multiple
// outputs are accumulated by the tape when they meet at the input tensor.
func (t *Tape) Split(input *Tensor, splits []int, axis int, outputCount ...int) ([]*Tensor, error) {
	if err := requireFloat32(input, "autodiff split input"); err != nil {
		return nil, err
	}
	outputs, err := Split(input, splits, axis, outputCount...)
	if err != nil {
		return nil, err
	}
	resolvedAxis, err := normalizeAxis(axis, len(input.shape), "autodiff split")
	if err != nil {
		return nil, err
	}
	resolvedSplits := append([]int(nil), splits...)
	if len(resolvedSplits) == 0 {
		resolvedSplits = make([]int, len(outputs))
		for index := range resolvedSplits {
			resolvedSplits[index] = outputs[index].shape[resolvedAxis]
		}
	}
	offset := 0
	for index, output := range outputs {
		start := offset
		size := resolvedSplits[index]
		t.record("Split", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
			return []*Tensor{splitVJP(input, resolvedAxis, start, upstream)}, nil
		})
		offset += size
	}
	return outputs, nil
}

// Gemm runs the existing Gemm kernel and records its VJP. All Gemm options
// accepted by the kernel are differentiable, including alpha, beta, and both
// transpose flags.
func (t *Tape) Gemm(a, b, c *Tensor, options ...GemmOptions) (*Tensor, error) {
	output, err := Gemm(a, b, c, options...)
	if err != nil {
		return nil, err
	}
	opts, err := gemmOptions(options)
	if err != nil {
		return nil, err
	}
	inputs := []*Tensor{a, b}
	if c != nil {
		inputs = append(inputs, c)
	}
	t.record("Gemm", inputs, output, func(upstream *Tensor) ([]*Tensor, error) {
		return gemmVJP(a, b, c, opts, upstream)
	})
	return output, nil
}

// SoftmaxCrossEntropy computes a stable mean cross-entropy loss in one
// operation. Its VJP is softmax(logits)-onehot(labels), divided by the batch
// size; there is intentionally no separate training Softmax or log-loss API.
func (t *Tape) SoftmaxCrossEntropy(logits, labels *Tensor) (*Tensor, error) {
	loss, err := softmaxCrossEntropyForward(logits, labels)
	if err != nil {
		return nil, err
	}
	t.record("SoftmaxCrossEntropy", []*Tensor{logits, labels}, loss, func(upstream *Tensor) ([]*Tensor, error) {
		gradient, err := softmaxCrossEntropyVJP(logits, labels, upstream)
		if err != nil {
			return nil, err
		}
		return []*Tensor{gradient, nil}, nil
	})
	return loss, nil
}

// Backward clears previous gradients and walks the recorded operations in
// reverse order from a scalar loss.
func (t *Tape) Backward(loss *Tensor) error {
	if loss == nil {
		return fmt.Errorf("backward loss is nil")
	}
	if err := requireFloat32(loss, "backward loss"); err != nil {
		return err
	}
	if len(loss.shape) != 0 {
		return fmt.Errorf("backward requires a scalar loss, got shape %v", loss.shape)
	}
	t.grads = make(map[*Tensor]*Tensor)
	initial, err := newFloat32Tensor(nil, []float32{1})
	if err != nil {
		return err
	}
	t.grads[loss] = initial
	for index := len(t.ops) - 1; index >= 0; index-- {
		op := t.ops[index]
		upstream := t.grads[op.output]
		if upstream == nil {
			continue
		}
		if op.vjp == nil {
			return fmt.Errorf("backward %s: no VJP implemented for %s", op.name, op.name)
		}
		inputGradients, err := op.vjp(upstream)
		if err != nil {
			return fmt.Errorf("backward %s: %w", op.name, err)
		}
		if len(inputGradients) != len(op.inputs) {
			return fmt.Errorf("backward %s returned %d gradients for %d inputs", op.name, len(inputGradients), len(op.inputs))
		}
		for inputIndex, gradient := range inputGradients {
			if gradient == nil {
				continue
			}
			if err := addGradient(t.grads, op.inputs[inputIndex], gradient); err != nil {
				return fmt.Errorf("backward %s input %d: %w", op.name, inputIndex, err)
			}
		}
	}
	for value, parameter := range t.marked {
		parameter.grad = t.grads[value]
	}
	return nil
}

// Grad returns a copy of a tracked parameter's gradient. A parameter not
// reached by the loss receives an all-zero tensor of its own shape.
func (t *Tape) Grad(param *Tensor) (*Tensor, error) {
	if _, ok := t.marked[param]; !ok {
		return nil, fmt.Errorf("tensor is not a tracked parameter")
	}
	if gradient := t.grads[param]; gradient != nil {
		return copyTensor(gradient)
	}
	return newZeroFloat32Tensor(param.shape)
}

// SGD applies one in-place gradient-descent step to every tracked parameter.
func (t *Tape) SGD(rate float32) error {
	if math.IsNaN(float64(rate)) || math.IsInf(float64(rate), 0) {
		return fmt.Errorf("sgd learning rate must be finite")
	}
	for _, param := range t.params {
		value := param.value
		if err := requireFloat32(value, "sgd parameter"); err != nil {
			return err
		}
		gradient := t.grads[value]
		if gradient == nil {
			continue
		}
		for index := range value.data {
			value.data[index] -= rate * gradient.data[index]
		}
	}
	return nil
}

// Adam applies one bias-corrected Adam step to every tracked parameter that
// has a gradient. State belongs to each parameter and survives later steps.
// The defaults match torch.optim.Adam: beta1=0.9, beta2=0.999, eps=1e-8,
// without weight decay or AMSGrad.
func (t *Tape) Adam(rate float32) error {
	return t.adamStep(rate, 0)
}

// AdamW applies one bias-corrected AdamW step to every tracked parameter that
// has a gradient. Weight decay is decoupled from the moments and uses the
// scheduled learning rate, matching torch.optim.AdamW.
func (t *Tape) AdamW(rate, weightDecay float32) error {
	if math.IsNaN(float64(weightDecay)) || math.IsInf(float64(weightDecay), 0) || weightDecay < 0 {
		return fmt.Errorf("adamw weight decay must be finite and non-negative")
	}
	return t.adamStep(rate, weightDecay)
}

func (t *Tape) adamStep(rate, weightDecay float32) error {
	if math.IsNaN(float64(rate)) || math.IsInf(float64(rate), 0) || rate < 0 {
		return fmt.Errorf("adam learning rate must be finite and non-negative")
	}
	const (
		beta1 = float32(0.9)
		beta2 = float32(0.999)
		eps   = float64(1e-8)
	)
	for _, param := range t.params {
		value := param.value
		if err := requireFloat32(value, "adam parameter"); err != nil {
			return err
		}
		gradient := t.grads[value]
		if gradient == nil {
			continue
		}
		if len(param.adamM) == 0 {
			param.adamM = make([]float32, len(value.data))
			param.adamV = make([]float32, len(value.data))
		}
		if len(param.adamM) != len(value.data) || len(param.adamV) != len(value.data) {
			return fmt.Errorf("adam state shape does not match parameter shape %v", value.shape)
		}
		if weightDecay != 0 {
			for index := range value.data {
				value.data[index] -= rate * weightDecay * value.data[index]
			}
		}
		param.adamStep++
		step := float64(param.adamStep)
		bias1 := 1 - math.Pow(float64(beta1), step)
		bias2 := 1 - math.Pow(float64(beta2), step)
		for index, grad := range gradient.data {
			param.adamM[index] = beta1*param.adamM[index] + (1-beta1)*grad
			param.adamV[index] = beta2*param.adamV[index] + (1-beta2)*grad*grad
			mHat := float64(param.adamM[index]) / bias1
			vHat := float64(param.adamV[index]) / bias2
			value.data[index] -= rate * float32(mHat/(math.Sqrt(vHat)+eps))
		}
	}
	return nil
}

func (t *Tape) record(name string, inputs []*Tensor, output *Tensor, vjp func(*Tensor) ([]*Tensor, error)) {
	t.ops = append(t.ops, tapeOp{name: name, inputs: inputs, output: output, vjp: vjp})
}

func matMulVJP(a, b, upstream *Tensor) ([]*Tensor, error) {
	if err := requireFloat32(upstream, "matmul upstream"); err != nil {
		return nil, err
	}
	return matMulVJPBatched(a, b, upstream)
}

func addVJP(a, b, upstream *Tensor) ([]*Tensor, error) {
	if err := requireFloat32(upstream, "add upstream"); err != nil {
		return nil, err
	}
	dA, err := reduceBroadcastGradientTape(upstream, a.shape)
	if err != nil {
		return nil, err
	}
	dB, err := reduceBroadcastGradientTape(upstream, b.shape)
	if err != nil {
		return nil, err
	}
	return []*Tensor{dA, dB}, nil
}

func reluVJP(input, output, upstream *Tensor) *Tensor {
	gradient, _ := newZeroFloat32Tensor(input.shape)
	for index := range gradient.data {
		if input.data[index] > 0 {
			gradient.data[index] = upstream.data[index]
		}
	}
	return gradient
}

func sigmoidVJP(output, upstream *Tensor) *Tensor {
	gradient, _ := newZeroFloat32Tensor(output.shape)
	for index, value := range output.data {
		gradient.data[index] = upstream.data[index] * value * (1 - value)
	}
	return gradient
}

func tanhVJP(output, upstream *Tensor) *Tensor {
	gradient, _ := newZeroFloat32Tensor(output.shape)
	for index, value := range output.data {
		gradient.data[index] = upstream.data[index] * (1 - value*value)
	}
	return gradient
}

func gemmOptions(options []GemmOptions) (GemmOptions, error) {
	if len(options) > 1 {
		return GemmOptions{}, fmt.Errorf("gemm accepts at most one options value")
	}
	if len(options) == 0 {
		return GemmOptions{Alpha: 1, Beta: 1}, nil
	}
	return options[0], nil
}

func gemmVJP(a, b, c *Tensor, options GemmOptions, upstream *Tensor) ([]*Tensor, error) {
	if err := requireFloat32(upstream, "gemm upstream"); err != nil {
		return nil, err
	}
	opA, err := transposeIf(a, options.TransA)
	if err != nil {
		return nil, err
	}
	opB, err := transposeIf(b, options.TransB)
	if err != nil {
		return nil, err
	}
	opBT, err := Transpose(opB)
	if err != nil {
		return nil, err
	}
	opAT, err := Transpose(opA)
	if err != nil {
		return nil, err
	}
	dOpA, err := MatMul(upstream, opBT)
	if err != nil {
		return nil, err
	}
	dOpB, err := MatMul(opAT, upstream)
	if err != nil {
		return nil, err
	}
	dA, err := scaleTensor(dOpA, options.Alpha)
	if err != nil {
		return nil, err
	}
	dB, err := scaleTensor(dOpB, options.Alpha)
	if err != nil {
		return nil, err
	}
	if options.TransA {
		dA, err = Transpose(dA)
		if err != nil {
			return nil, err
		}
	}
	if options.TransB {
		dB, err = Transpose(dB)
		if err != nil {
			return nil, err
		}
	}
	gradients := []*Tensor{dA, dB}
	if c != nil {
		dC, err := scaleTensor(upstream, options.Beta)
		if err != nil {
			return nil, err
		}
		dC, err = reduceBroadcastGradientTape(dC, c.shape)
		if err != nil {
			return nil, err
		}
		gradients = append(gradients, dC)
	}
	return gradients, nil
}

func transposeIf(input *Tensor, transpose bool) (*Tensor, error) {
	if !transpose {
		return input, nil
	}
	return Transpose(input)
}

func scaleTensor(input *Tensor, scale float32) (*Tensor, error) {
	if err := requireFloat32(input, "scale input"); err != nil {
		return nil, err
	}
	result, err := newZeroFloat32Tensor(input.shape)
	if err != nil {
		return nil, err
	}
	for index, value := range input.data {
		result.data[index] = value * scale
	}
	return result, nil
}

func reduceBroadcastGradientTape(upstream *Tensor, targetShape []int) (*Tensor, error) {
	if err := requireFloat32(upstream, "broadcast upstream"); err != nil {
		return nil, err
	}
	if _, err := tensorBroadcastShape(targetShape, upstream.shape); err != nil {
		return nil, fmt.Errorf("cannot reduce gradient with shape %v to %v: %w", upstream.shape, targetShape, err)
	}
	result, err := newZeroFloat32Tensor(targetShape)
	if err != nil {
		return nil, err
	}
	outputRank := len(upstream.shape)
	targetStrides := alignedBroadcastStrides(result, outputRank)
	for outputIndex := range upstream.data {
		remaining := outputIndex
		targetIndex := 0
		for axis, stride := range upstream.strides {
			coordinate := 0
			if stride != 0 {
				coordinate = remaining / stride
				remaining %= stride
			}
			targetIndex += coordinate * targetStrides[axis]
		}
		result.data[targetIndex] += upstream.data[outputIndex]
	}
	return result, nil
}

func addGradient(grads map[*Tensor]*Tensor, input, gradient *Tensor) error {
	if input == nil {
		return fmt.Errorf("gradient input is nil")
	}
	if err := requireFloat32(gradient, "gradient"); err != nil {
		return err
	}
	if existing := grads[input]; existing != nil {
		if !sameShape(existing.shape, gradient.shape) {
			return fmt.Errorf("gradient shape %v does not match existing shape %v", gradient.shape, existing.shape)
		}
		for index := range existing.data {
			existing.data[index] += gradient.data[index]
		}
		return nil
	}
	grads[input], _ = copyTensor(gradient)
	return nil
}

func softmaxCrossEntropyForward(logits, labels *Tensor) (*Tensor, error) {
	if err := requireFloat32(logits, "cross-entropy logits"); err != nil {
		return nil, err
	}
	if labels == nil || labels.dtype != DTypeInt64 {
		return nil, fmt.Errorf("cross-entropy labels must have dtype int64")
	}
	if len(logits.shape) != 2 || len(labels.shape) != 1 || logits.shape[0] != labels.shape[0] {
		return nil, fmt.Errorf("cross-entropy requires logits [N,C] and labels [N], got %v and %v", logits.shape, labels.shape)
	}
	rows, classes := logits.shape[0], logits.shape[1]
	if rows == 0 || classes == 0 {
		return nil, fmt.Errorf("cross-entropy requires non-empty logits")
	}
	var total float64
	for row := 0; row < rows; row++ {
		label := labels.int64Data[row]
		if label < 0 || label >= int64(classes) {
			return nil, fmt.Errorf("cross-entropy label %d at row %d is outside [0,%d)", label, row, classes)
		}
		maxValue := float32(math.Inf(-1))
		for class := 0; class < classes; class++ {
			if logits.data[row*classes+class] > maxValue {
				maxValue = logits.data[row*classes+class]
			}
		}
		var sum float64
		for class := 0; class < classes; class++ {
			sum += math.Exp(float64(logits.data[row*classes+class] - maxValue))
		}
		total += -float64(logits.data[row*classes+int(label)]-maxValue) + math.Log(sum)
	}
	return newFloat32Tensor(nil, []float32{float32(total / float64(rows))})
}

func softmaxCrossEntropyVJP(logits, labels, upstream *Tensor) (*Tensor, error) {
	if err := requireFloat32(upstream, "cross-entropy upstream"); err != nil {
		return nil, err
	}
	if len(upstream.shape) != 0 {
		return nil, fmt.Errorf("cross-entropy upstream must be scalar, got shape %v", upstream.shape)
	}
	gradient, err := newZeroFloat32Tensor(logits.shape)
	if err != nil {
		return nil, err
	}
	rows, classes := logits.shape[0], logits.shape[1]
	scale := upstream.data[0] / float32(rows)
	for row := 0; row < rows; row++ {
		maxValue := float32(math.Inf(-1))
		for class := 0; class < classes; class++ {
			if logits.data[row*classes+class] > maxValue {
				maxValue = logits.data[row*classes+class]
			}
		}
		var sum float64
		for class := 0; class < classes; class++ {
			sum += math.Exp(float64(logits.data[row*classes+class] - maxValue))
		}
		for class := 0; class < classes; class++ {
			probability := float32(math.Exp(float64(logits.data[row*classes+class]-maxValue)) / sum)
			gradient.data[row*classes+class] = probability * scale
		}
		gradient.data[row*classes+int(labels.int64Data[row])] -= scale
	}
	return gradient, nil
}
