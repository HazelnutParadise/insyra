package dl

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// BoundRegressor adapts a single-input ONNX model to the ml.Model protocol.
// The adapter deliberately does not import ml: the protocol is structural.
type BoundRegressor struct {
	model     *Model
	inputName string
	features  []string
	inputSpec ValueInfo
	output    ValueInfo
}

// Regressor is a short alias for BoundRegressor.
type Regressor = BoundRegressor

// BindRegressor binds feature names to a two-dimensional float32 ONNX input.
// The model must have exactly one runtime input and one runtime output. Extra
// columns in a prediction table are ignored; bound columns are read in the
// order supplied here.
func BindRegressor(model *Model, inputName string, features []string) (*BoundRegressor, error) {
	input, err := bindInput(model, inputName, features)
	if err != nil {
		return nil, err
	}
	outputs := model.Outputs()
	if len(outputs) != 1 {
		return nil, fmt.Errorf("regressor model must have exactly one output, got %d", len(outputs))
	}
	if outputWidth, known := declaredWidth(outputs[0]); known && outputWidth != 1 {
		return nil, fmt.Errorf("regressor output %q has width %d, want 1", outputs[0].Name, outputWidth)
	}
	return &BoundRegressor{
		model:     model,
		inputName: inputName,
		features:  append([]string(nil), features...),
		inputSpec: input,
		output:    outputs[0],
	}, nil
}

// Features reports the names and order used to construct the model input.
func (m *BoundRegressor) Features() []string {
	if m == nil {
		return nil
	}
	return append([]string(nil), m.features...)
}

// Predict converts named table columns to the model input and returns one
// prediction per row.
func (m *BoundRegressor) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	if m == nil {
		return nil, fmt.Errorf("regressor adapter is nil")
	}
	output, rows, err := m.predictTensor(dt)
	if err != nil {
		return nil, err
	}
	values := make([]any, rows)
	for row := range values {
		values[row] = float64(output[row])
	}
	return insyra.NewDataList(values...), nil
}

func (m *BoundRegressor) predictTensor(dt *insyra.DataTable) ([]float32, int, error) {
	input, rows, err := tableInputTensor(dt, m.inputSpec, m.features)
	if err != nil {
		return nil, 0, err
	}
	outputs, err := m.model.Run(map[string]*Tensor{m.inputName: input})
	if err != nil {
		return nil, 0, err
	}
	value := outputs[m.output.Name]
	data, outputRows, width, err := tensorMatrixData(value, m.output.Name)
	if err != nil {
		return nil, 0, err
	}
	if outputRows != rows {
		return nil, 0, fmt.Errorf("output %q has %d rows, want %d", m.output.Name, outputRows, rows)
	}
	if width != 1 {
		return nil, 0, fmt.Errorf("output %q has width %d, want 1", m.output.Name, width)
	}
	return data, rows, nil
}

// BoundClassifier adapts a single-input ONNX model whose output is a class
// probability matrix to the ml.Classifier and ml.ProbaModel protocols.
type BoundClassifier struct {
	model         *Model
	inputName     string
	features      []string
	inputSpec     ValueInfo
	probabilities ValueInfo
	classes       *insyra.DataList
}

// Classifier is a short alias for BoundClassifier.
type Classifier = BoundClassifier

// BindClassifier binds feature names and caller-owned class labels to a model.
// The probability output is selected by the conventional ONNX name
// "probabilities", or is the sole output when a model has only one output.
func BindClassifier(model *Model, inputName string, features []string, classes *insyra.DataList) (*BoundClassifier, error) {
	input, err := bindInput(model, inputName, features)
	if err != nil {
		return nil, err
	}
	if classes == nil || classes.Len() == 0 {
		return nil, fmt.Errorf("classifier classes must not be empty")
	}
	outputs := model.Outputs()
	if len(outputs) == 0 {
		return nil, fmt.Errorf("classifier model has no outputs")
	}
	probabilityOutput, found := ValueInfo{}, false
	for _, output := range outputs {
		if output.Name == "probabilities" {
			probabilityOutput, found = output, true
			break
		}
	}
	if !found {
		if len(outputs) != 1 {
			return nil, fmt.Errorf("classifier probability output %q is not declared", "probabilities")
		}
		probabilityOutput = outputs[0]
	}
	if probabilityOutput.DType != DTypeFloat32 {
		return nil, fmt.Errorf("classifier probability output %q has dtype %s, want %s", probabilityOutput.Name, dtypeName(probabilityOutput.DType), dtypeName(DTypeFloat32))
	}
	if width, known := declaredWidth(probabilityOutput); known && width != classes.Len() {
		return nil, fmt.Errorf("classifier probability output %q has width %d, want class count %d", probabilityOutput.Name, width, classes.Len())
	}
	return &BoundClassifier{
		model:         model,
		inputName:     inputName,
		features:      append([]string(nil), features...),
		inputSpec:     input,
		probabilities: probabilityOutput,
		classes:       classes.Clone(),
	}, nil
}

// Features reports the names and order used to construct the model input.
func (m *BoundClassifier) Features() []string {
	if m == nil {
		return nil
	}
	return append([]string(nil), m.features...)
}

// Classes returns an independent copy of the caller-supplied class labels.
func (m *BoundClassifier) Classes() *insyra.DataList {
	if m == nil || m.classes == nil {
		return nil
	}
	return m.classes.Clone()
}

// Predict chooses the first class holding the maximum probability in each
// row, matching the protocol's tie behavior.
func (m *BoundClassifier) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	probabilities, err := m.PredictProba(dt)
	if err != nil {
		return nil, err
	}
	classes := m.classes.Data()
	labels := make([]any, probabilities.NumRows())
	for row := range labels {
		best := 0
		bestValue := float64(-1)
		for column := range classes {
			value, ok := insyra.ToFloat64Safe(probabilities.GetColByNumber(column).Get(row))
			if !ok {
				return nil, fmt.Errorf("probability column %q has a non-numeric value at row %d", probabilities.ColNames()[column], row)
			}
			if column == 0 || value > bestValue {
				best, bestValue = column, value
			}
		}
		labels[row] = classes[best]
	}
	return insyra.NewDataList(labels...), nil
}

// PredictProba returns one probability column per class, in the bound class
// order.
func (m *BoundClassifier) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error) {
	if m == nil {
		return nil, fmt.Errorf("classifier adapter is nil")
	}
	input, rows, err := tableInputTensor(dt, m.inputSpec, m.features)
	if err != nil {
		return nil, err
	}
	outputs, err := m.model.Run(map[string]*Tensor{m.inputName: input})
	if err != nil {
		return nil, err
	}
	value := outputs[m.probabilities.Name]
	data, outputRows, width, err := tensorMatrixData(value, m.probabilities.Name)
	if err != nil {
		return nil, err
	}
	if outputRows != rows {
		return nil, fmt.Errorf("output %q has %d rows, want %d", m.probabilities.Name, outputRows, rows)
	}
	if width != m.classes.Len() {
		return nil, fmt.Errorf("classifier probability output %q has width %d, want class count %d", m.probabilities.Name, width, m.classes.Len())
	}
	columns := make([]*insyra.DataList, width)
	classes := m.classes.Data()
	for column := range columns {
		values := make([]any, rows)
		for row := range values {
			rowStart := row * width
			var sum float64
			for index := 0; index < width; index++ {
				probability := float64(data[rowStart+index])
				if math.IsNaN(probability) || math.IsInf(probability, 0) || probability < 0 {
					return nil, fmt.Errorf("classifier probability output %q has invalid value %v at row %d column %d", m.probabilities.Name, probability, row, index)
				}
				sum += probability
			}
			if !math.IsInf(sum, 0) && sum <= 0 {
				return nil, fmt.Errorf("classifier probability output %q has non-positive sum at row %d", m.probabilities.Name, row)
			}
			if math.IsInf(sum, 0) || math.IsNaN(sum) {
				return nil, fmt.Errorf("classifier probability output %q has invalid sum at row %d", m.probabilities.Name, row)
			}
			values[row] = float64(data[rowStart+column]) / sum
		}
		columns[column] = insyra.NewDataList(values...).SetName(fmt.Sprint(classes[column]))
	}
	return insyra.NewDataTable(columns...), nil
}

func bindInput(model *Model, inputName string, features []string) (ValueInfo, error) {
	if model == nil {
		return ValueInfo{}, fmt.Errorf("ONNX model is nil")
	}
	if err := validateBoundFeatures(features); err != nil {
		return ValueInfo{}, err
	}
	inputs := model.Inputs()
	var input ValueInfo
	found := false
	for _, candidate := range inputs {
		if candidate.Name == inputName {
			input, found = candidate, true
			break
		}
	}
	if !found {
		return ValueInfo{}, fmt.Errorf("model input %q is not declared", inputName)
	}
	if len(inputs) != 1 {
		return ValueInfo{}, fmt.Errorf("model input %q cannot be bound alone; model declares %d runtime inputs", inputName, len(inputs))
	}
	if input.DType != DTypeFloat32 {
		return ValueInfo{}, fmt.Errorf("model input %q has dtype %s, want %s", inputName, dtypeName(input.DType), dtypeName(DTypeFloat32))
	}
	if width, known := declaredWidth(input); known && width != len(features) {
		return ValueInfo{}, fmt.Errorf("model input %q has width %d, want feature count %d", inputName, width, len(features))
	}
	if input.HasShape && len(input.Shape) != 2 {
		return ValueInfo{}, fmt.Errorf("model input %q has shape %v, want a two-dimensional feature matrix", inputName, input.Shape)
	}
	return input, nil
}

func validateBoundFeatures(features []string) error {
	if len(features) == 0 {
		return fmt.Errorf("bound features must not be empty")
	}
	seen := make(map[string]struct{}, len(features))
	for index, feature := range features {
		if feature == "" {
			return fmt.Errorf("bound feature %d has no name", index)
		}
		if _, exists := seen[feature]; exists {
			return fmt.Errorf("bound feature %q is not unique", feature)
		}
		seen[feature] = struct{}{}
	}
	return nil
}

func declaredWidth(info ValueInfo) (int, bool) {
	if !info.HasShape {
		return 0, false
	}
	if len(info.Shape) == 1 {
		return 1, true
	}
	if len(info.Shape) == 2 && info.Shape[1] >= 0 {
		return info.Shape[1], true
	}
	return 0, false
}

func tableInputTensor(dt *insyra.DataTable, spec ValueInfo, features []string) (*Tensor, int, error) {
	if dt == nil {
		return nil, 0, fmt.Errorf("prediction data table is nil")
	}
	columns := make([][]any, len(features))
	for index, feature := range features {
		column := dt.GetColByName(feature)
		if column == nil {
			return nil, 0, fmt.Errorf("missing feature column %q", feature)
		}
		columns[index] = column.Data()
	}
	rows := dt.NumRows()
	shape := []int{rows, len(features)}
	if spec.HasShape && len(spec.Shape) == 1 {
		shape = []int{rows}
	}
	data := make([]float32, rows*len(features))
	for row := 0; row < rows; row++ {
		for column, values := range columns {
			if row >= len(values) {
				return nil, 0, fmt.Errorf("feature column %q has no value at row %d", features[column], row)
			}
			value, ok := insyra.ToFloat64Safe(values[row])
			if !ok {
				return nil, 0, fmt.Errorf("feature column %q has a non-numeric value at row %d", features[column], row)
			}
			if len(features) == 1 && len(shape) == 1 {
				data[row] = float32(value)
			} else {
				data[row*len(features)+column] = float32(value)
			}
		}
	}
	if len(shape) == 1 {
		tensor, err := newFloat32Tensor(shape, data[:rows])
		return tensor, rows, err
	}
	tensor, err := newFloat32Tensor(shape, data)
	return tensor, rows, err
}

func tensorMatrixData(value *Tensor, name string) ([]float32, int, int, error) {
	if value == nil {
		return nil, 0, 0, fmt.Errorf("output %q was not produced", name)
	}
	if value.dtype != DTypeFloat32 {
		return nil, 0, 0, fmt.Errorf("output %q has dtype %s, want %s", name, dtypeName(value.dtype), dtypeName(DTypeFloat32))
	}
	data := append([]float32(nil), value.data...)
	switch len(value.shape) {
	case 1:
		return data, value.shape[0], 1, nil
	case 2:
		return data, value.shape[0], value.shape[1], nil
	default:
		return nil, 0, 0, fmt.Errorf("output %q has shape %v, want [rows] or [rows, width]", name, value.shape)
	}
}
