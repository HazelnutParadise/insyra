package nn

import (
	"fmt"
	"math"
)

type conv2DLayer struct {
	in, out, kernel int
	opts            ConvOptions
	weight          *Parameter
	bias            *Parameter
}

// Conv2D creates a trainable NCHW convolution. Its weights use torch's
// [out,in/groups,kh,kw] layout, so LoadWeights does not transpose them.
// ConvOptions.NoBias omits the optional bias; the default includes it.
func Conv2D(in, out, kernel int, options ...ConvOptions) Layer {
	opts := ConvOptions{Group: 1}
	if len(options) == 1 {
		opts = options[0]
		if opts.Group == 0 {
			opts.Group = 1
		}
	}
	return &conv2DLayer{in: in, out: out, kernel: kernel, opts: opts}
}

// NewConv2D is the constructor-style spelling of Conv2D.
func NewConv2D(in, out, kernel int, options ...ConvOptions) Layer {
	return Conv2D(in, out, kernel, options...)
}

func (l *conv2DLayer) Build(t *Tape) error {
	if l.in <= 0 || l.out <= 0 || l.kernel <= 0 {
		return fmt.Errorf("conv2d dimensions must be positive, got %d -> %d with kernel %d", l.in, l.out, l.kernel)
	}
	if t == nil {
		return fmt.Errorf("conv2d build tape is nil")
	}
	if l.opts.Group <= 0 || l.in%l.opts.Group != 0 || l.out%l.opts.Group != 0 {
		return fmt.Errorf("conv2d group %d must divide input %d and output %d channels", l.opts.Group, l.in, l.out)
	}
	if l.weight != nil {
		return nil
	}
	if t.rng == nil {
		t.rng = newDefaultTapeRNG()
	}
	perGroup := l.in / l.opts.Group
	scale := math.Sqrt(2 / float64(perGroup*l.kernel*l.kernel))
	weightValues := make([]float32, l.out*perGroup*l.kernel*l.kernel)
	for index := range weightValues {
		weightValues[index] = float32(t.rng.NormFloat64() * scale)
	}
	weight, err := newFloat32Tensor([]int{l.out, perGroup, l.kernel, l.kernel}, weightValues)
	if err != nil {
		return err
	}
	l.weight, err = t.Param(weight)
	if err != nil {
		return err
	}
	if l.opts.NoBias {
		return nil
	}
	bias, err := newZeroFloat32Tensor([]int{l.out})
	if err != nil {
		return err
	}
	l.bias, err = t.Param(bias)
	return err
}

func (l *conv2DLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if l.weight == nil {
		return nil, fmt.Errorf("conv2d layer %d -> %d is not built", l.in, l.out)
	}
	if x == nil || x.dtype != DTypeFloat32 {
		return nil, fmt.Errorf("conv2d input must be float32")
	}
	if len(x.shape) != 4 || x.shape[1] != l.in {
		return nil, fmt.Errorf("conv2d input shape %v, want [N %d H W]", x.shape, l.in)
	}
	if t == nil {
		return nil, fmt.Errorf("conv2d forward tape is nil")
	}
	return t.Conv(x, l.weight.Value(), parameterValue(l.bias), l.opts)
}

func (l *conv2DLayer) Parameters() []*Parameter {
	if l.weight == nil {
		return nil
	}
	parameters := []*Parameter{l.weight}
	if l.bias != nil {
		parameters = append(parameters, l.bias)
	}
	return parameters
}

func (l *conv2DLayer) namedParameters() map[string]*Parameter {
	state := map[string]*Parameter{"weight": l.weight}
	if l.bias != nil {
		state["bias"] = l.bias
	}
	return state
}

func (l *conv2DLayer) stateDict() map[string]*Tensor {
	state := map[string]*Tensor{"weight": l.weight.Value()}
	if l.bias != nil {
		state["bias"] = l.bias.Value()
	}
	return state
}

func (l *conv2DLayer) loadState(weights map[string]*Tensor) error {
	if err := loadCatalogTensor(weights["weight"], l.weight.Value(), "weight", false); err != nil {
		return err
	}
	if l.bias != nil {
		if err := loadCatalogTensor(weights["bias"], l.bias.Value(), "bias", false); err != nil {
			return err
		}
	}
	return nil
}

func (l *conv2DLayer) layerDimensions() (int, int, bool) { return l.in, 0, false }
func (l *conv2DLayer) layerKind() string                 { return "Conv2D" }

type pool2DLayer struct {
	kernel int
	opts   PoolOptions
	max    bool
}

// MaxPool2D creates a max-pooling layer over NCHW tensors.
func MaxPool2D(kernel int, options ...PoolOptions) Layer {
	return newPool2DLayer(kernel, true, options...)
}

// NewMaxPool2D is the constructor-style spelling of MaxPool2D.
func NewMaxPool2D(kernel int, options ...PoolOptions) Layer {
	return MaxPool2D(kernel, options...)
}

// AvgPool2D creates an average-pooling layer over NCHW tensors.
func AvgPool2D(kernel int, options ...PoolOptions) Layer {
	return newPool2DLayer(kernel, false, options...)
}

// NewAvgPool2D is the constructor-style spelling of AvgPool2D.
func NewAvgPool2D(kernel int, options ...PoolOptions) Layer {
	return AvgPool2D(kernel, options...)
}

func newPool2DLayer(kernel int, max bool, options ...PoolOptions) Layer {
	opts := PoolOptions{}
	if len(options) == 1 {
		opts = options[0]
	}
	if len(opts.Strides) == 0 {
		opts.Strides = []int{kernel, kernel}
	}
	return &pool2DLayer{kernel: kernel, opts: opts, max: max}
}

func (l *pool2DLayer) Build(*Tape) error {
	if l.kernel <= 0 {
		return fmt.Errorf("%s kernel must be positive, got %d", l.layerKind(), l.kernel)
	}
	if len(l.opts.Pads) != 0 && len(l.opts.Pads) != 2 && len(l.opts.Pads) != 4 {
		return fmt.Errorf("%s pads must contain 2 or 4 values, got %v", l.layerKind(), l.opts.Pads)
	}
	return nil
}

func (l *pool2DLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if t == nil {
		return nil, fmt.Errorf("%s forward tape is nil", l.layerKind())
	}
	if l.max {
		return t.MaxPool(x, []int{l.kernel, l.kernel}, l.opts)
	}
	return t.AveragePool(x, []int{l.kernel, l.kernel}, l.opts)
}

func (l *pool2DLayer) Parameters() []*Parameter { return nil }
func (l *pool2DLayer) layerKind() string {
	if l.max {
		return "MaxPool2D"
	}
	return "AvgPool2D"
}

type globalAvgPoolLayer struct{}

// GlobalAvgPool creates a global average-pooling layer.
func GlobalAvgPool() Layer { return &globalAvgPoolLayer{} }

// NewGlobalAvgPool is the constructor-style spelling of GlobalAvgPool.
func NewGlobalAvgPool() Layer { return GlobalAvgPool() }

func (*globalAvgPoolLayer) Build(*Tape) error { return nil }
func (l *globalAvgPoolLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	return t.GlobalAveragePool(x)
}
func (*globalAvgPoolLayer) Parameters() []*Parameter { return nil }
func (*globalAvgPoolLayer) layerKind() string        { return "GlobalAvgPool" }

type batchNorm2DLayer struct {
	features                     int
	epsilon, momentum            float32
	weight, bias                 *Parameter
	runningMean, runningVariance *Tensor
}

// BatchNorm2D creates an affine BatchNorm layer with torch defaults
// momentum=0.1 and eps=1e-5. Forward uses batch statistics and updates the
// running buffers; Sequential.Predict uses those buffers instead.
func BatchNorm2D(features int) Layer {
	return &batchNorm2DLayer{features: features, epsilon: 1e-5, momentum: 0.1}
}

// NewBatchNorm2D is the constructor-style spelling of BatchNorm2D.
func NewBatchNorm2D(features int) Layer { return BatchNorm2D(features) }

func (l *batchNorm2DLayer) Build(t *Tape) error {
	if l.features <= 0 {
		return fmt.Errorf("batchnorm2d features must be positive, got %d", l.features)
	}
	if t == nil {
		return fmt.Errorf("batchnorm2d build tape is nil")
	}
	if l.weight != nil {
		return nil
	}
	weight, err := newFloat32Tensor([]int{l.features}, make([]float32, l.features))
	if err != nil {
		return err
	}
	for index := range weight.data {
		weight.data[index] = 1
	}
	l.weight, err = t.Param(weight)
	if err != nil {
		return err
	}
	bias, err := newZeroFloat32Tensor([]int{l.features})
	if err != nil {
		return err
	}
	l.bias, err = t.Param(bias)
	if err != nil {
		return err
	}
	l.runningMean, err = newZeroFloat32Tensor([]int{l.features})
	if err != nil {
		return err
	}
	l.runningVariance, err = newFloat32Tensor([]int{l.features}, make([]float32, l.features))
	if err != nil {
		return err
	}
	for index := range l.runningVariance.data {
		l.runningVariance.data[index] = 1
	}
	return nil
}

func (l *batchNorm2DLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if l.weight == nil || l.runningMean == nil {
		return nil, fmt.Errorf("batchnorm2d layer with %d features is not built", l.features)
	}
	if x == nil || x.dtype != DTypeFloat32 || len(x.shape) != 4 || x.shape[1] != l.features {
		return nil, fmt.Errorf("batchnorm2d input shape %v, want [N %d H W] float32", shapeOf(x), l.features)
	}
	return t.BatchNormalizationTraining(x, l.weight.Value(), l.bias.Value(), l.runningMean, l.runningVariance, l.momentum, l.epsilon)
}

func (l *batchNorm2DLayer) PredictForward(x *Tensor) (*Tensor, error) {
	if l.weight == nil || l.runningMean == nil {
		return nil, fmt.Errorf("batchnorm2d layer with %d features is not built", l.features)
	}
	return BatchNormalization(x, l.weight.Value(), l.bias.Value(), l.runningMean, l.runningVariance, l.epsilon)
}

func (l *batchNorm2DLayer) Parameters() []*Parameter { return []*Parameter{l.weight, l.bias} }
func (l *batchNorm2DLayer) namedParameters() map[string]*Parameter {
	return map[string]*Parameter{"weight": l.weight, "bias": l.bias}
}
func (l *batchNorm2DLayer) stateDict() map[string]*Tensor {
	return map[string]*Tensor{"weight": l.weight.Value(), "bias": l.bias.Value(), "running_mean": l.runningMean, "running_var": l.runningVariance}
}
func (l *batchNorm2DLayer) loadState(weights map[string]*Tensor) error {
	for _, name := range []string{"weight", "bias"} {
		if err := loadCatalogTensor(weights[name], l.stateDict()[name], name, false); err != nil {
			return err
		}
	}
	if err := loadCatalogTensor(weights["running_mean"], l.runningMean, "running_mean", false); err != nil {
		return err
	}
	return loadCatalogTensor(weights["running_var"], l.runningVariance, "running_var", false)
}
func (l *batchNorm2DLayer) layerKind() string { return "BatchNorm2D" }

type layerNormLayer struct {
	dims         []int
	epsilon      float32
	weight, bias *Parameter
}

// LayerNorm normalizes the last dimension and applies learned affine values.
// dims may be an int or a []int normalized shape, matching torch's accepted
// forms.
func LayerNorm(dims interface{}) Layer {
	var shape []int
	switch value := dims.(type) {
	case int:
		shape = []int{value}
	case []int:
		shape = append([]int(nil), value...)
	default:
		shape = []int{0}
	}
	return &layerNormLayer{dims: shape, epsilon: 1e-5}
}

// NewLayerNorm is the constructor-style spelling of LayerNorm.
func NewLayerNorm(dims interface{}) Layer { return LayerNorm(dims) }

func (l *layerNormLayer) Build(t *Tape) error {
	if len(l.dims) == 0 {
		return fmt.Errorf("layernorm dimensions must not be empty")
	}
	for _, dim := range l.dims {
		if dim <= 0 {
			return fmt.Errorf("layernorm dimensions must be positive, got %v", l.dims)
		}
	}
	if t == nil {
		return fmt.Errorf("layernorm build tape is nil")
	}
	if l.weight != nil {
		return nil
	}
	weight, err := newFloat32Tensor(l.dims, make([]float32, elementCount(l.dims)))
	if err != nil {
		return err
	}
	for index := range weight.data {
		weight.data[index] = 1
	}
	l.weight, err = t.Param(weight)
	if err != nil {
		return err
	}
	bias, err := newZeroFloat32Tensor(l.dims)
	if err != nil {
		return err
	}
	l.bias, err = t.Param(bias)
	return err
}

func (l *layerNormLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if l.weight == nil {
		return nil, fmt.Errorf("layernorm with dimensions %v is not built", l.dims)
	}
	if x == nil || len(x.shape) < len(l.dims) || !sequentialSameShape(x.shape[len(x.shape)-len(l.dims):], l.dims) {
		return nil, fmt.Errorf("layernorm input shape %v, want suffix %v", shapeOf(x), l.dims)
	}
	return t.LayerNormalization(x, l.weight.Value(), l.bias.Value(), len(x.shape)-len(l.dims), l.epsilon)
}

func (l *layerNormLayer) Parameters() []*Parameter { return []*Parameter{l.weight, l.bias} }
func (l *layerNormLayer) namedParameters() map[string]*Parameter {
	return map[string]*Parameter{"weight": l.weight, "bias": l.bias}
}
func (l *layerNormLayer) stateDict() map[string]*Tensor {
	return map[string]*Tensor{"weight": l.weight.Value(), "bias": l.bias.Value()}
}
func (l *layerNormLayer) loadState(weights map[string]*Tensor) error {
	if err := loadCatalogTensor(weights["weight"], l.weight.Value(), "weight", false); err != nil {
		return err
	}
	return loadCatalogTensor(weights["bias"], l.bias.Value(), "bias", false)
}
func (l *layerNormLayer) layerKind() string { return "LayerNorm" }

type embeddingLayer struct {
	vocab, dims int
	weight      *Parameter
}

// Embedding creates a trainable [vocab, dims] token table.
func Embedding(vocab, dims int) Layer { return &embeddingLayer{vocab: vocab, dims: dims} }

// NewEmbedding is the constructor-style spelling of Embedding.
func NewEmbedding(vocab, dims int) Layer { return Embedding(vocab, dims) }

func (l *embeddingLayer) Build(t *Tape) error {
	if l.vocab <= 0 || l.dims <= 0 {
		return fmt.Errorf("embedding dimensions must be positive, got vocab=%d dim=%d", l.vocab, l.dims)
	}
	if t == nil {
		return fmt.Errorf("embedding build tape is nil")
	}
	if l.weight != nil {
		return nil
	}
	if t.rng == nil {
		t.rng = newDefaultTapeRNG()
	}
	values := make([]float32, l.vocab*l.dims)
	for index := range values {
		values[index] = float32(t.rng.NormFloat64() * 0.02)
	}
	weight, err := newFloat32Tensor([]int{l.vocab, l.dims}, values)
	if err != nil {
		return err
	}
	l.weight, err = t.Param(weight)
	return err
}

func (l *embeddingLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if l.weight == nil {
		return nil, fmt.Errorf("embedding with vocab=%d dim=%d is not built", l.vocab, l.dims)
	}
	return t.Embedding(l.weight.Value(), x)
}

func (l *embeddingLayer) Parameters() []*Parameter { return []*Parameter{l.weight} }
func (l *embeddingLayer) namedParameters() map[string]*Parameter {
	return map[string]*Parameter{"weight": l.weight}
}
func (l *embeddingLayer) stateDict() map[string]*Tensor {
	return map[string]*Tensor{"weight": l.weight.Value()}
}
func (l *embeddingLayer) loadState(weights map[string]*Tensor) error {
	return loadCatalogTensor(weights["weight"], l.weight.Value(), "weight", false)
}
func (l *embeddingLayer) layerKind() string { return "Embedding" }

func parameterValue(parameter *Parameter) *Tensor {
	if parameter == nil {
		return nil
	}
	return parameter.Value()
}

func shapeOf(tensor *Tensor) []int {
	if tensor == nil {
		return nil
	}
	return tensor.shape
}

func loadCatalogTensor(source, destination *Tensor, name string, transpose bool) error {
	if source == nil {
		return fmt.Errorf("%s is missing", name)
	}
	if source.dtype != DTypeFloat32 {
		return fmt.Errorf("%s dtype %s, want float32", name, source.dtype)
	}
	if transpose {
		return fmt.Errorf("%s transpose is unsupported", name)
	}
	if !sequentialSameShape(source.shape, destination.shape) {
		return fmt.Errorf("%s shape %v, want %v", name, source.shape, destination.shape)
	}
	copy(destination.data, source.data)
	return nil
}
