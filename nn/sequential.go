package nn

import (
	"fmt"
	"sort"
)

// Sequential composes layers in construction order. Layer positions include
// parameterless layers, matching torch.nn.Sequential state-dict names.
type Sequential struct {
	layers []Layer
	tape   *Tape
}

// NewSequential eagerly builds every layer on t. Known adjacent dimensions
// are checked before the next layer is built so errors identify the layer that
// cannot attach.
func NewSequential(t *Tape, layers ...Layer) (*Sequential, error) {
	if t == nil {
		return nil, fmt.Errorf("sequential build tape is nil")
	}
	if len(layers) == 0 {
		return nil, fmt.Errorf("sequential requires at least one layer")
	}
	model := &Sequential{layers: append([]Layer(nil), layers...), tape: t}
	knownOutput := 0
	for index, layer := range model.layers {
		if layer == nil {
			return nil, fmt.Errorf("layer %d (%s): layer is nil", index, sequentialLayerKind(layer))
		}
		kind := sequentialLayerKind(layer)
		if dimensions, ok := layer.(dimensionedLayer); ok {
			input, output, known := dimensions.layerDimensions()
			if known && knownOutput != 0 && input != knownOutput {
				return nil, fmt.Errorf("layer %d (%s): input dimension %d does not match previous output dimension %d", index, kind, input, knownOutput)
			}
			if err := layer.Build(t); err != nil {
				return nil, fmt.Errorf("layer %d (%s): %w", index, kind, err)
			}
			if known {
				knownOutput = output
			}
			continue
		}
		if err := layer.Build(t); err != nil {
			return nil, fmt.Errorf("layer %d (%s): %w", index, kind, err)
		}
	}
	return model, nil
}

// Forward runs the training path and records all differentiable operations
// onto t, including TrainingOnly layers.
func (s *Sequential) Forward(t *Tape, x *Tensor) (*Tensor, error) {
	if s == nil {
		return nil, fmt.Errorf("sequential is nil")
	}
	if t == nil {
		return nil, fmt.Errorf("sequential forward tape is nil")
	}
	if x == nil {
		return nil, fmt.Errorf("sequential input is nil")
	}
	output := x
	for index, layer := range s.layers {
		var err error
		output, err = layer.Forward(t, output)
		if err != nil {
			return nil, fmt.Errorf("layer %d (%s): %w", index, sequentialLayerKind(layer), err)
		}
	}
	return output, nil
}

// Predict runs inference on a throwaway tape and structurally omits every
// TrainingOnly layer.
func (s *Sequential) Predict(x *Tensor) (*Tensor, error) {
	if s == nil {
		return nil, fmt.Errorf("sequential is nil")
	}
	if x == nil {
		return nil, fmt.Errorf("sequential input is nil")
	}
	tape := NewTape()
	output := x
	for index, layer := range s.layers {
		if _, trainingOnly := layer.(TrainingOnly); trainingOnly {
			continue
		}
		var err error
		output, err = layer.Forward(tape, output)
		if err != nil {
			return nil, fmt.Errorf("layer %d (%s): %w", index, sequentialLayerKind(layer), err)
		}
	}
	return output, nil
}

// Parameters returns all layer parameters in layer and creation order.
func (s *Sequential) Parameters() []*Parameter {
	if s == nil {
		return nil
	}
	parameters := make([]*Parameter, 0)
	for _, layer := range s.layers {
		parameters = append(parameters, layer.Parameters()...)
	}
	return parameters
}

// NamedParameters returns torch.nn.Sequential-style names. Parameterless
// layers still consume their position index.
func (s *Sequential) NamedParameters() map[string]*Parameter {
	if s == nil {
		return nil
	}
	named := make(map[string]*Parameter)
	for index, layer := range s.layers {
		if layerWithNames, ok := layer.(namedLayer); ok {
			for name, parameter := range layerWithNames.namedParameters() {
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

// LoadWeights loads a SafeTensors map using torch Sequential names. Dense
// weights are expected in torch Linear's [out,in] layout and are transposed
// into the tape's internal [in,out] layout.
func (s *Sequential) LoadWeights(weights map[string]*Tensor) error {
	if s == nil {
		return fmt.Errorf("sequential is nil")
	}
	expected := s.NamedParameters()
	missing := make([]string, 0)
	for name := range expected {
		if _, ok := weights[name]; !ok {
			missing = append(missing, name)
		}
	}
	extra := make([]string, 0)
	for name := range weights {
		if _, ok := expected[name]; !ok {
			extra = append(extra, name)
		}
	}
	if len(missing) > 0 || len(extra) > 0 {
		sort.Strings(missing)
		sort.Strings(extra)
		return fmt.Errorf("load sequential weights: missing names %v; extra names %v", missing, extra)
	}
	for index, layer := range s.layers {
		dense, ok := layer.(*denseLayer)
		if !ok {
			continue
		}
		weight := weights[fmt.Sprintf("%d.weight", index)]
		bias := weights[fmt.Sprintf("%d.bias", index)]
		if err := loadDenseWeights(dense, weight, bias); err != nil {
			return fmt.Errorf("layer %d (Dense): %w", index, err)
		}
	}
	return nil
}

func loadDenseWeights(layer *denseLayer, torchWeight, bias *Tensor) error {
	if torchWeight == nil {
		return fmt.Errorf("weight is nil")
	}
	if torchWeight.dtype != DTypeFloat32 {
		return fmt.Errorf("weight dtype %s, want float32", torchWeight.dtype)
	}
	if !sequentialSameShape(torchWeight.shape, []int{layer.out, layer.in}) {
		return fmt.Errorf("weight shape %v, want [%d %d] in torch [out,in] layout", torchWeight.shape, layer.out, layer.in)
	}
	if bias == nil {
		return fmt.Errorf("bias is nil")
	}
	if bias.dtype != DTypeFloat32 {
		return fmt.Errorf("bias dtype %s, want float32", bias.dtype)
	}
	if !sequentialSameShape(bias.shape, []int{layer.out}) {
		return fmt.Errorf("bias shape %v, want [%d]", bias.shape, layer.out)
	}
	for output := 0; output < layer.out; output++ {
		for input := 0; input < layer.in; input++ {
			layer.weight.value.data[input*layer.out+output] = torchWeight.data[output*layer.in+input]
		}
	}
	copy(layer.bias.value.data, bias.data)
	return nil
}

func sequentialSameShape(left, right []int) bool {
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

func sequentialLayerKind(layer Layer) string {
	if layer == nil {
		return "<nil>"
	}
	if kind, ok := layer.(layerKind); ok {
		return kind.layerKind()
	}
	return "Layer"
}
