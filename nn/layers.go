package nn

import (
	"fmt"
	"math"
)

// Layer is a trainable or stateless operation that can be composed by
// Sequential. Build materializes parameters on the supplied tape.
type Layer interface {
	Build(t *Tape) error
	Forward(t *Tape, x *Tensor) (*Tensor, error)
	Parameters() []*Parameter
}

// TrainingOnly marks a layer that is present in the training path but must be
// structurally omitted by Sequential.Predict.
type TrainingOnly interface {
	TrainingOnly()
}

type dimensionedLayer interface {
	layerDimensions() (input, output int, known bool)
}

type layerKind interface {
	layerKind() string
}

type namedLayer interface {
	namedParameters() map[string]*Parameter
}

type denseLayer struct {
	in, out int
	weight  *Parameter
	bias    *Parameter
}

// Dense creates a fully connected layer. Its internal weight layout is
// [in,out], matching the tape's MatMul convention. LoadWeights accepts torch's
// [out,in] layout and transposes at that boundary.
func Dense(in, out int) Layer {
	return &denseLayer{in: in, out: out}
}

// NewDense is the constructor-style spelling of Dense.
func NewDense(in, out int) Layer { return Dense(in, out) }

func (l *denseLayer) Build(t *Tape) error {
	if l.in <= 0 || l.out <= 0 {
		return fmt.Errorf("dense dimensions must be positive, got %d -> %d", l.in, l.out)
	}
	if t == nil {
		return fmt.Errorf("dense build tape is nil")
	}
	if l.weight != nil {
		return nil
	}
	if t.rng == nil {
		t.rng = newDefaultTapeRNG()
	}
	scale := math.Sqrt(2 / float64(l.in))
	values := make([]float32, l.in*l.out)
	for index := range values {
		values[index] = float32(t.rng.NormFloat64() * scale)
	}
	weight, err := newFloat32Tensor([]int{l.in, l.out}, values)
	if err != nil {
		return err
	}
	l.weight, err = t.Param(weight)
	if err != nil {
		return err
	}
	bias, err := newZeroFloat32Tensor([]int{l.out})
	if err != nil {
		return err
	}
	l.bias, err = t.Param(bias)
	return err
}

func (l *denseLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if l.weight == nil || l.bias == nil {
		return nil, fmt.Errorf("dense layer %d -> %d is not built", l.in, l.out)
	}
	if err := requireFloat32(x, "dense input"); err != nil {
		return nil, err
	}
	if len(x.shape) == 0 || x.shape[len(x.shape)-1] != l.in {
		return nil, fmt.Errorf("dense input shape %v has last dimension %d, want %d", x.shape, lastDimension(x), l.in)
	}
	if t == nil {
		return nil, fmt.Errorf("dense forward tape is nil")
	}
	output, err := t.MatMul(x, l.weight.Value())
	if err != nil {
		return nil, err
	}
	return t.Add(output, l.bias.Value())
}

func (l *denseLayer) Parameters() []*Parameter {
	if l.weight == nil {
		return nil
	}
	return []*Parameter{l.weight, l.bias}
}

func (l *denseLayer) namedParameters() map[string]*Parameter {
	return map[string]*Parameter{"weight": l.weight, "bias": l.bias}
}

func (l *denseLayer) layerDimensions() (int, int, bool) { return l.in, l.out, true }
func (l *denseLayer) layerKind() string                 { return "Dense" }

type activationLayer struct {
	kind    string
	forward func(*Tape, *Tensor) (*Tensor, error)
}

func (l *activationLayer) Build(*Tape) error { return nil }
func (l *activationLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if t == nil {
		return nil, fmt.Errorf("%s forward tape is nil", l.kind)
	}
	if err := requireFloat32(x, stringsToLower(l.kind)+" input"); err != nil {
		return nil, err
	}
	return l.forward(t, x)
}
func (l *activationLayer) Parameters() []*Parameter { return nil }
func (l *activationLayer) layerKind() string        { return l.kind }

// ReLU creates a differentiable rectified-linear activation layer.
func ReLU() Layer {
	return &activationLayer{kind: "ReLU", forward: func(t *Tape, x *Tensor) (*Tensor, error) { return t.Relu(x) }}
}

// NewReLU is the constructor-style spelling of ReLU.
func NewReLU() Layer { return ReLU() }

// NewSigmoid creates a differentiable sigmoid activation layer.
func NewSigmoid() Layer {
	return &activationLayer{kind: "Sigmoid", forward: func(t *Tape, x *Tensor) (*Tensor, error) { return t.Sigmoid(x) }}
}

// NewTanh creates a differentiable hyperbolic-tangent activation layer.
func NewTanh() Layer {
	return &activationLayer{kind: "Tanh", forward: func(t *Tape, x *Tensor) (*Tensor, error) { return t.Tanh(x) }}
}

// NewGelu creates an exact-form GELU activation layer.
func NewGelu() Layer {
	return &activationLayer{kind: "Gelu", forward: func(t *Tape, x *Tensor) (*Tensor, error) { return t.Gelu(x) }}
}

type dropoutLayer struct {
	p float32
}

// Dropout creates an inverted training-only dropout layer. Predict skips it.
func Dropout(p float32) Layer { return &dropoutLayer{p: p} }

// NewDropout is the constructor-style spelling of Dropout.
func NewDropout(p float32) Layer { return Dropout(p) }

func (l *dropoutLayer) Build(*Tape) error {
	if math.IsNaN(float64(l.p)) || math.IsInf(float64(l.p), 0) || l.p < 0 || l.p >= 1 {
		return fmt.Errorf("dropout probability must be in [0, 1), got %g", l.p)
	}
	return nil
}
func (l *dropoutLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if t == nil {
		return nil, fmt.Errorf("Dropout forward tape is nil")
	}
	return t.Dropout(x, l.p)
}
func (l *dropoutLayer) Parameters() []*Parameter { return nil }
func (l *dropoutLayer) TrainingOnly()            {}
func (l *dropoutLayer) layerKind() string        { return "Dropout" }

type flattenLayer struct{}

// NewFlatten creates a layer that keeps the batch dimension and collapses the
// remaining dimensions.
func NewFlatten() Layer { return &flattenLayer{} }

func (l *flattenLayer) Build(*Tape) error { return nil }
func (l *flattenLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if t == nil {
		return nil, fmt.Errorf("Flatten forward tape is nil")
	}
	return t.Flatten(x)
}
func (l *flattenLayer) Parameters() []*Parameter { return nil }
func (l *flattenLayer) layerKind() string        { return "Flatten" }

type funcLayer struct {
	fn func(*Tape, *Tensor) (*Tensor, error)
}

// Func wraps an arbitrary tape operation, including residual or other
// composite blocks that are not a catalogued layer.
func Func(fn func(*Tape, *Tensor) (*Tensor, error)) Layer { return &funcLayer{fn: fn} }

// NewFunc is the constructor-style spelling of Func.
func NewFunc(fn func(*Tape, *Tensor) (*Tensor, error)) Layer { return Func(fn) }

func (l *funcLayer) Build(*Tape) error {
	if l.fn == nil {
		return fmt.Errorf("Func callback is nil")
	}
	return nil
}
func (l *funcLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if l.fn == nil {
		return nil, fmt.Errorf("Func callback is nil")
	}
	if t == nil {
		return nil, fmt.Errorf("Func forward tape is nil")
	}
	output, err := l.fn(t, x)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, fmt.Errorf("Func callback returned nil tensor")
	}
	return output, nil
}
func (l *funcLayer) Parameters() []*Parameter { return nil }
func (l *funcLayer) layerKind() string        { return "Func" }

func lastDimension(t *Tensor) int {
	if t == nil || len(t.shape) == 0 {
		return 0
	}
	return t.shape[len(t.shape)-1]
}

func stringsToLower(value string) string {
	if value == "ReLU" {
		return "relu"
	}
	if value == "Gelu" {
		return "gelu"
	}
	return value
}
