package nn

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// multiHeadAttentionLayer implements mask-free batch-first self-attention.
// The parameters use the tape's [in,out] MatMul layout internally; the
// torch-compatible [out,in] layout is handled by loadState/saveState.
type multiHeadAttentionLayer struct {
	embed, heads, headDim int
	inProjWeight          *Parameter
	inProjBias            *Parameter
	outProjWeight         *Parameter
	outProjBias           *Parameter
}

// MultiHeadAttention creates mask-free batch-first self-attention over
// [batch, sequence, embed] tensors. It follows torch's fused in-projection
// and out-projection state-dict names.
func MultiHeadAttention(embed, heads int) Layer {
	return &multiHeadAttentionLayer{embed: embed, heads: heads}
}

// NewMultiHeadAttention is the constructor-style spelling of
// MultiHeadAttention.
func NewMultiHeadAttention(embed, heads int) Layer {
	return MultiHeadAttention(embed, heads)
}

func (l *multiHeadAttentionLayer) Build(t *Tape) error {
	if l.embed <= 0 || l.heads <= 0 {
		return fmt.Errorf("multiheadattention dimensions must be positive, got embed=%d heads=%d", l.embed, l.heads)
	}
	if l.embed%l.heads != 0 {
		return fmt.Errorf("multiheadattention embed dimension %d must be divisible by heads %d", l.embed, l.heads)
	}
	if t == nil {
		return fmt.Errorf("multiheadattention build tape is nil")
	}
	if l.inProjWeight != nil {
		return nil
	}
	l.headDim = l.embed / l.heads
	if t.rng == nil {
		t.rng = newDefaultTapeRNG()
	}
	init := func(shape []int, scale float64) (*Parameter, error) {
		values := make([]float32, elementCount(shape))
		for index := range values {
			values[index] = float32(t.rng.NormFloat64() * scale)
		}
		value, err := newFloat32Tensor(shape, values)
		if err != nil {
			return nil, err
		}
		return t.Param(value)
	}
	zero := func(shape []int) (*Parameter, error) {
		value, err := newZeroFloat32Tensor(shape)
		if err != nil {
			return nil, err
		}
		return t.Param(value)
	}
	scale := math.Sqrt(2 / float64(l.embed))
	var err error
	if l.inProjWeight, err = init([]int{l.embed, 3 * l.embed}, scale); err != nil {
		return err
	}
	if l.inProjBias, err = zero([]int{3 * l.embed}); err != nil {
		return err
	}
	if l.outProjWeight, err = init([]int{l.embed, l.embed}, scale); err != nil {
		return err
	}
	if l.outProjBias, err = zero([]int{l.embed}); err != nil {
		return err
	}
	return nil
}

func (l *multiHeadAttentionLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if l.inProjWeight == nil {
		return nil, fmt.Errorf("multiheadattention embed=%d heads=%d is not built", l.embed, l.heads)
	}
	if t == nil {
		return nil, fmt.Errorf("multiheadattention forward tape is nil")
	}
	if err := requireFloat32(x, "multiheadattention input"); err != nil {
		return nil, err
	}
	if len(x.shape) != 3 || x.shape[2] != l.embed {
		return nil, fmt.Errorf("multiheadattention input shape %v, want [batch sequence %d]", shapeOf(x), l.embed)
	}
	batch, sequence := x.shape[0], x.shape[1]
	projected, err := t.MatMul(x, l.inProjWeight.Value())
	if err != nil {
		return nil, err
	}
	projected, err = t.Add(projected, l.inProjBias.Value())
	if err != nil {
		return nil, err
	}
	parts, err := t.Split(projected, []int{l.embed, l.embed, l.embed}, -1)
	if err != nil {
		return nil, err
	}
	q, err := t.Reshape(parts[0], []int{batch, sequence, l.heads, l.headDim})
	if err != nil {
		return nil, err
	}
	q, err = t.Transpose(q, []int{0, 2, 1, 3})
	if err != nil {
		return nil, err
	}
	k, err := t.Reshape(parts[1], []int{batch, sequence, l.heads, l.headDim})
	if err != nil {
		return nil, err
	}
	k, err = t.Transpose(k, []int{0, 2, 3, 1})
	if err != nil {
		return nil, err
	}
	v, err := t.Reshape(parts[2], []int{batch, sequence, l.heads, l.headDim})
	if err != nil {
		return nil, err
	}
	v, err = t.Transpose(v, []int{0, 2, 1, 3})
	if err != nil {
		return nil, err
	}
	q, err = t.Reshape(q, []int{batch * l.heads, sequence, l.headDim})
	if err != nil {
		return nil, err
	}
	k, err = t.Reshape(k, []int{batch * l.heads, l.headDim, sequence})
	if err != nil {
		return nil, err
	}
	v, err = t.Reshape(v, []int{batch * l.heads, sequence, l.headDim})
	if err != nil {
		return nil, err
	}
	scores, err := t.MatMul(q, k)
	if err != nil {
		return nil, err
	}
	scaleTensor, err := newFloat32Tensor(nil, []float32{float32(math.Sqrt(float64(l.headDim)))})
	if err != nil {
		return nil, err
	}
	scores, err = t.Div(scores, scaleTensor)
	if err != nil {
		return nil, err
	}
	probability, err := t.Softmax(scores, -1)
	if err != nil {
		return nil, err
	}
	context, err := t.MatMul(probability, v)
	if err != nil {
		return nil, err
	}
	context, err = t.Reshape(context, []int{batch, l.heads, sequence, l.headDim})
	if err != nil {
		return nil, err
	}
	context, err = t.Transpose(context, []int{0, 2, 1, 3})
	if err != nil {
		return nil, err
	}
	context, err = t.Reshape(context, []int{batch, sequence, l.embed})
	if err != nil {
		return nil, err
	}
	output, err := t.MatMul(context, l.outProjWeight.Value())
	if err != nil {
		return nil, err
	}
	return t.Add(output, l.outProjBias.Value())
}

func (l *multiHeadAttentionLayer) Parameters() []*Parameter {
	if l.inProjWeight == nil {
		return nil
	}
	return []*Parameter{l.inProjWeight, l.inProjBias, l.outProjWeight, l.outProjBias}
}

func (l *multiHeadAttentionLayer) namedParameters() map[string]*Parameter {
	return map[string]*Parameter{
		"in_proj_weight":  l.inProjWeight,
		"in_proj_bias":    l.inProjBias,
		"out_proj.weight": l.outProjWeight,
		"out_proj.bias":   l.outProjBias,
	}
}

func (l *multiHeadAttentionLayer) stateDict() map[string]*Tensor {
	return map[string]*Tensor{
		"in_proj_weight":  l.inProjWeight.Value(),
		"in_proj_bias":    l.inProjBias.Value(),
		"out_proj.weight": l.outProjWeight.Value(),
		"out_proj.bias":   l.outProjBias.Value(),
	}
}

func (l *multiHeadAttentionLayer) saveState() map[string]*Tensor {
	state := l.stateDict()
	for _, name := range []string{"in_proj_weight", "out_proj.weight"} {
		state[name] = mustTransposeLayerWeight(state[name])
	}
	return state
}

func (l *multiHeadAttentionLayer) loadState(weights map[string]*Tensor) error {
	if err := loadAttentionWeight(weights["in_proj_weight"], l.inProjWeight.Value(), "in_proj_weight", []int{3 * l.embed, l.embed}); err != nil {
		return err
	}
	if err := loadCatalogTensor(weights["in_proj_bias"], l.inProjBias.Value(), "in_proj_bias", false); err != nil {
		return err
	}
	if err := loadAttentionWeight(weights["out_proj.weight"], l.outProjWeight.Value(), "out_proj.weight", []int{l.embed, l.embed}); err != nil {
		return err
	}
	return loadCatalogTensor(weights["out_proj.bias"], l.outProjBias.Value(), "out_proj.bias", false)
}

func (l *multiHeadAttentionLayer) layerDimensions() (int, int, bool) {
	return l.embed, l.embed, true
}

func (l *multiHeadAttentionLayer) layerKind() string { return "MultiHeadAttention" }

type residualLayer struct {
	layers []Layer
}

// Residual creates a block whose output is x plus the output of its sub-stack.
// Predict uses EvalLayer paths and a throwaway tape for the sub-stack, then
// performs the residual addition as a plain inference kernel.
func Residual(layers ...Layer) Layer {
	return &residualLayer{layers: append([]Layer(nil), layers...)}
}

func (l *residualLayer) Build(t *Tape) error {
	if len(l.layers) == 0 {
		return fmt.Errorf("residual requires at least one layer")
	}
	for index, layer := range l.layers {
		if layer == nil {
			return fmt.Errorf("residual layer %d (%s): layer is nil", index, sequentialLayerKind(layer))
		}
		if err := layer.Build(t); err != nil {
			return fmt.Errorf("residual layer %d (%s): %w", index, sequentialLayerKind(layer), err)
		}
	}
	return nil
}

func (l *residualLayer) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	output, err := l.forwardStack(t, x, false)
	if err != nil {
		return nil, err
	}
	return t.Add(x, output)
}

func (l *residualLayer) PredictForward(x *Tensor) (*Tensor, error) {
	tape := NewTape()
	output, err := l.forwardStack(tape, x, true)
	if err != nil {
		return nil, err
	}
	return Add(x, output)
}

func (l *residualLayer) forwardStack(t *Tape, x *Tensor, inference bool) (*Tensor, error) {
	if t == nil {
		return nil, fmt.Errorf("residual forward tape is nil")
	}
	if x == nil {
		return nil, fmt.Errorf("residual input is nil")
	}
	output := x
	for index, layer := range l.layers {
		if inference {
			if _, trainingOnly := layer.(TrainingOnly); trainingOnly {
				continue
			}
		}
		var err error
		if inference {
			if eval, ok := layer.(EvalLayer); ok {
				output, err = eval.PredictForward(output)
			} else {
				output, err = layer.Forward(t, output)
			}
		} else {
			output, err = layer.Forward(t, output)
		}
		if err != nil {
			return nil, fmt.Errorf("residual layer %d (%s): %w", index, sequentialLayerKind(layer), err)
		}
	}
	return output, nil
}

func (l *residualLayer) Parameters() []*Parameter {
	var parameters []*Parameter
	for _, layer := range l.layers {
		parameters = append(parameters, layer.Parameters()...)
	}
	return parameters
}

func (l *residualLayer) namedParameters() map[string]*Parameter {
	named := make(map[string]*Parameter)
	for index, layer := range l.layers {
		if child, ok := layer.(namedLayer); ok {
			for name, parameter := range child.namedParameters() {
				if parameter != nil {
					named[fmt.Sprintf("%d.%s", index, name)] = parameter
				}
			}
			continue
		}
		for parameterIndex, parameter := range layer.Parameters() {
			if parameter != nil {
				named[fmt.Sprintf("%d.parameter%d", index, parameterIndex)] = parameter
			}
		}
	}
	return named
}

func (l *residualLayer) stateDict() map[string]*Tensor {
	return l.stateDictWith(func(layer Layer) map[string]*Tensor {
		if stateful, ok := layer.(statefulLayer); ok {
			return stateful.stateDict()
		}
		return layerNamedState(layer)
	})
}

func (l *residualLayer) saveState() map[string]*Tensor {
	return l.stateDictWith(layerStateForSave)
}

func (l *residualLayer) stateDictWith(childState func(Layer) map[string]*Tensor) map[string]*Tensor {
	state := make(map[string]*Tensor)
	for index, layer := range l.layers {
		for name, tensor := range childState(layer) {
			state[fmt.Sprintf("%d.%s", index, name)] = tensor
		}
	}
	return state
}

func (l *residualLayer) loadState(weights map[string]*Tensor) error {
	children := make([]map[string]*Tensor, len(l.layers))
	for name, tensor := range weights {
		parts := strings.SplitN(name, ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("residual state name %q is missing a child layer index", name)
		}
		index, err := strconv.Atoi(parts[0])
		if err != nil || index < 0 || index >= len(children) {
			return fmt.Errorf("residual state name %q has invalid child layer index", name)
		}
		if children[index] == nil {
			children[index] = make(map[string]*Tensor)
		}
		children[index][parts[1]] = tensor
	}
	for index, layer := range l.layers {
		stateful, ok := layer.(statefulLayer)
		if !ok {
			continue
		}
		if err := stateful.loadState(children[index]); err != nil {
			return fmt.Errorf("residual layer %d (%s): %w", index, sequentialLayerKind(layer), err)
		}
	}
	return nil
}

func (l *residualLayer) layerDimensions() (int, int, bool) {
	var input, output int
	var inputKnown, outputKnown bool
	for _, layer := range l.layers {
		if dimensions, ok := layer.(dimensionedLayer); ok {
			in, out, known := dimensions.layerDimensions()
			if !inputKnown && known {
				input, inputKnown = in, true
			}
			if known {
				output, outputKnown = out, true
			}
		}
	}
	return input, output, inputKnown && outputKnown
}

func (l *residualLayer) layerKind() string { return "Residual" }

type saveStateLayer interface {
	saveState() map[string]*Tensor
}

func layerStateForSave(layer Layer) map[string]*Tensor {
	if saver, ok := layer.(saveStateLayer); ok {
		return saver.saveState()
	}
	var state map[string]*Tensor
	if stateful, ok := layer.(statefulLayer); ok {
		state = stateful.stateDict()
	} else {
		state = layerNamedState(layer)
	}
	if _, ok := layer.(*denseLayer); ok && state["weight"] != nil {
		state["weight"] = mustTransposeLayerWeight(state["weight"])
	}
	return state
}

func mustTransposeLayerWeight(source *Tensor) *Tensor {
	transposed, err := Transpose(source, []int{1, 0})
	if err != nil {
		return nil
	}
	return transposed
}

func loadAttentionWeight(source, destination *Tensor, name string, wantShape []int) error {
	if source == nil {
		return fmt.Errorf("%s is missing", name)
	}
	if source.dtype != DTypeFloat32 {
		return fmt.Errorf("%s dtype %s, want float32", name, source.dtype)
	}
	if !sequentialSameShape(source.shape, wantShape) {
		return fmt.Errorf("%s shape %v, want %v in torch [out,in] layout", name, source.shape, wantShape)
	}
	if len(destination.shape) != 2 {
		return fmt.Errorf("%s destination shape %v is not a matrix", name, destination.shape)
	}
	for output := 0; output < source.shape[0]; output++ {
		for input := 0; input < source.shape[1]; input++ {
			destination.data[input*source.shape[0]+output] = source.data[output*source.shape[1]+input]
		}
	}
	return nil
}
