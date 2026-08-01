package ml

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

const (
	onnxTensorFloat  int32 = 1
	onnxTensorInt64  int32 = 7
	onnxTensorString int32 = 8
)

// ExportONNX writes a fitted model as a standard ONNX ModelProto. The model
// is built completely before w is touched, so unsupported models never leave
// a partial file behind.
func ExportONNX(w io.Writer, fitted any) error {
	if w == nil {
		return errors.New("ml: ONNX export writer is nil")
	}
	model, err := buildONNXModel(fitted)
	if err != nil {
		return err
	}
	payload := model.marshal()
	n, err := w.Write(payload)
	if err != nil {
		return err
	}
	if n != len(payload) {
		return io.ErrShortWrite
	}
	return nil
}

// WriteONNX is an explicit synonym for ExportONNX for callers that prefer a
// verb describing the destination.
func WriteONNX(w io.Writer, fitted any) error { return ExportONNX(w, fitted) }

func (m *LinearModel) ExportONNX(w io.Writer) error { return ExportONNX(w, m) }
func (m *RidgeModel) ExportONNX(w io.Writer) error  { return ExportONNX(w, m) }
func (m *LassoModel) ExportONNX(w io.Writer) error  { return ExportONNX(w, m) }

// ExportONNX writes the fitted weighted linear model.
func (m *WeightedLinearModel) ExportONNX(w io.Writer) error { return ExportONNX(w, m) }

// ExportONNX writes the fitted forest as one multi-tree ensemble.
func (m *RandomForestClassifier) ExportONNX(w io.Writer) error { return ExportONNX(w, m) }

// ExportONNX writes the fitted forest as one multi-tree ensemble.
func (m *RandomForestRegressor) ExportONNX(w io.Writer) error { return ExportONNX(w, m) }

// ExportONNX writes the boosted ensemble with the learning rate baked in.
func (m *GradientBoostingRegressor) ExportONNX(w io.Writer) error { return ExportONNX(w, m) }

// ExportONNX writes the boosted binary classifier as its log-odds ensemble.
func (m *GradientBoostingClassifier) ExportONNX(w io.Writer) error { return ExportONNX(w, m) }
func (m *LogisticModel) ExportONNX(w io.Writer) error              { return ExportONNX(w, m) }
func (m *DecisionTreeClassifier) ExportONNX(w io.Writer) error {
	return ExportONNX(w, m)
}
func (m *DecisionTreeRegressor) ExportONNX(w io.Writer) error {
	return ExportONNX(w, m)
}
func (p *fittedPipeline) ExportONNX(w io.Writer) error { return ExportONNX(w, p) }

type onnxGroup struct {
	name  string
	value string
	dtype int32
}

type onnxBuilder struct {
	graph  onnxGraphProto
	serial int
	axes1  string
}

func newONNXBuilder(features []string, inputTypes map[string]int32) (*onnxBuilder, []onnxGroup, error) {
	if len(features) == 0 {
		return nil, nil, errors.New("ml: ONNX export requires at least one feature")
	}
	b := &onnxBuilder{graph: onnxGraphProto{Name: "insyra_ml"}}
	groups := make([]onnxGroup, 0, len(features))
	seen := make(map[string]struct{}, len(features))
	for index, feature := range features {
		if feature == "" {
			return nil, nil, fmt.Errorf("ml: ONNX export feature %d has no name", index)
		}
		if _, exists := seen[feature]; exists {
			return nil, nil, fmt.Errorf("ml: ONNX export feature %q is not unique", feature)
		}
		seen[feature] = struct{}{}
		dtype := inputTypes[feature]
		if dtype == 0 {
			dtype = onnxTensorFloat
		}
		input := fmt.Sprintf("input_%d_%s", index, onnxSafeName(feature))
		b.graph.Inputs = append(b.graph.Inputs, onnxValueInfoProto{Name: input, ElemType: dtype, Shape: []int64{-1}})
		groups = append(groups, onnxGroup{name: feature, value: input, dtype: dtype})
	}
	return b, groups, nil
}

func (b *onnxBuilder) unique(prefix string) string {
	b.serial++
	return fmt.Sprintf("%s_%d", onnxSafeName(prefix), b.serial)
}

func (b *onnxBuilder) addNode(opType, domain string, inputs, outputs []string, attributes ...onnxAttributeProto) string {
	name := b.unique(opType)
	b.graph.Nodes = append(b.graph.Nodes, onnxNodeProto{
		Inputs: inputs, Outputs: outputs, Name: name, OpType: opType,
		Attributes: attributes, Domain: domain,
	})
	return name
}

func (b *onnxBuilder) addFloatInitializer(prefix string, value float32) string {
	name := b.unique(prefix)
	b.graph.Initializers = append(b.graph.Initializers, onnxTensorProto{
		Dims: []int64{1}, DataType: onnxTensorFloat, FloatData: []float32{value}, Name: name,
	})
	return name
}

func (b *onnxBuilder) addInt64Initializer(prefix string, values ...int64) string {
	name := b.unique(prefix)
	b.graph.Initializers = append(b.graph.Initializers, onnxTensorProto{
		Dims: []int64{int64(len(values))}, DataType: onnxTensorInt64, Int64Data: append([]int64(nil), values...), Name: name,
	})
	return name
}

func (b *onnxBuilder) addInt64ScalarInitializer(prefix string, value int64) string {
	name := b.unique(prefix)
	b.graph.Initializers = append(b.graph.Initializers, onnxTensorProto{
		DataType: onnxTensorInt64, Int64Data: []int64{value}, Name: name,
	})
	return name
}

func (b *onnxBuilder) castToFloat(group onnxGroup) onnxGroup {
	if group.dtype == onnxTensorFloat {
		return group
	}
	output := b.unique("cast_float")
	b.addNode("Cast", "", []string{group.value}, []string{output}, onnxAttrInt("to", int64(onnxTensorFloat)))
	group.value = output
	group.dtype = onnxTensorFloat
	return group
}

func (b *onnxBuilder) unsqueeze(group onnxGroup) string {
	if b.axes1 == "" {
		b.axes1 = b.addInt64Initializer("axes", 1)
	}
	output := b.unique("unsqueeze")
	b.addNode("Unsqueeze", "", []string{group.value, b.axes1}, []string{output})
	return output
}

func (b *onnxBuilder) matrix(groups []onnxGroup) (string, error) {
	if len(groups) == 0 {
		return "", errors.New("ml: ONNX export produced no model features")
	}
	inputs := make([]string, 0, len(groups))
	for _, group := range groups {
		group = b.castToFloat(group)
		inputs = append(inputs, b.unsqueeze(group))
	}
	if len(inputs) == 1 {
		return inputs[0], nil
	}
	output := b.unique("features")
	b.addNode("Concat", "", inputs, []string{output}, onnxAttrInt("axis", 1))
	return output, nil
}

func (b *onnxBuilder) addOutput(name string, dtype int32, shape []int64) {
	b.graph.Outputs = append(b.graph.Outputs, onnxValueInfoProto{Name: name, ElemType: dtype, Shape: shape})
}

func buildONNXModel(fitted any) (*onnxModelProto, error) {
	switch model := fitted.(type) {
	case *fittedPipeline:
		return buildPipelineONNX(model)
	case *fittedPipelineClassifier:
		return buildPipelineONNX(model.fittedPipeline)
	case *fittedPipelineProba:
		return buildPipelineONNX(model.fittedPipeline)
	case *fittedPipelineImportances:
		return buildPipelineONNX(model.fittedPipeline)
	case *fittedPipelineClassifierImportances:
		return buildPipelineONNX(model.fittedPipeline)
	case *fittedPipelineProbaImportances:
		return buildPipelineONNX(model.fittedPipeline)
	case *LinearModel:
		return buildDirectONNX(model, model.Features(), nil)
	case *LogisticModel:
		return buildDirectONNX(model, model.Features(), nil)
	case *DecisionTreeClassifier:
		return buildDirectONNX(model, model.Features(), model.featureSchemas)
	case *DecisionTreeRegressor:
		return buildDirectONNX(model, model.Features(), model.featureSchemas)
	case *RidgeModel:
		return buildDirectONNX(model, model.Features(), nil)
	case *LassoModel:
		return buildDirectONNX(model, model.Features(), nil)
	case *WeightedLinearModel:
		return buildDirectONNX(model, model.Features(), nil)
	case *RandomForestClassifier:
		return buildDirectONNX(model, model.Features(), forestSchemas(model.trees))
	case *RandomForestRegressor:
		return buildDirectONNX(model, model.Features(), forestSchemas(model.trees))
	case *GradientBoostingRegressor:
		return buildDirectONNX(model, model.Features(), forestSchemas(model.trees))
	case *GradientBoostingClassifier:
		return buildDirectONNX(model, model.Features(), forestSchemas(model.trees))
	case Model:
		return nil, unsupportedONNXModel(model)
	default:
		return nil, fmt.Errorf("ml: ONNX export does not support %T", fitted)
	}
}

func buildDirectONNX(model Model, features []string, schemas []treeFeature) (*onnxModelProto, error) {
	inputTypes, err := treeInputTypes(features, schemas)
	if err != nil {
		return nil, err
	}
	b, groups, err := newONNXBuilder(features, inputTypes)
	if err != nil {
		return nil, err
	}
	if len(schemas) != 0 {
		groups, err = applyTreeInputEncoding(b, groups, schemas)
		if err != nil {
			return nil, err
		}
	}
	if err := appendONNXPredictor(b, model, groups); err != nil {
		return nil, err
	}
	return finishONNXModel(b), nil
}

func buildPipelineONNX(p *fittedPipeline) (*onnxModelProto, error) {
	if p == nil || p.model == nil || isNilPointer(p.model) {
		return nil, errors.New("ml: ONNX export pipeline is nil")
	}
	inputTypes, err := pipelineInputTypes(p.features, p.model, p.steps)
	if err != nil {
		return nil, err
	}
	b, groups, err := newONNXBuilder(p.features, inputTypes)
	if err != nil {
		return nil, err
	}
	for _, step := range p.steps {
		groups, err = applyONNXTransformer(b, groups, step.transformer)
		if err != nil {
			return nil, fmt.Errorf("ml: ONNX export pipeline step %q: %w", step.name, err)
		}
	}
	if tree := treeSchemas(p.model); len(tree) != 0 {
		groups, err = applyTreeInputEncoding(b, groups, tree)
		if err != nil {
			return nil, err
		}
	}
	if err := appendONNXPredictor(b, p.model, groups); err != nil {
		return nil, err
	}
	return finishONNXModel(b), nil
}

func finishONNXModel(b *onnxBuilder) *onnxModelProto {
	return &onnxModelProto{
		IRVersion:    9,
		ProducerName: "insyra",
		Graph:        &b.graph,
		OpsetImport:  []onnxOperatorSetIdProto{{Version: 13}, {Domain: "ai.onnx.ml", Version: 3}},
	}
}

func unsupportedONNXModel(model Model) error {
	return fmt.Errorf("ml: ONNX export does not support model %T", model)
}

func treeSchemas(model Model) []treeFeature {
	switch model := model.(type) {
	case *DecisionTreeClassifier:
		return model.featureSchemas
	case *DecisionTreeRegressor:
		return model.featureSchemas
	case *RandomForestClassifier:
		return forestSchemas(model.trees)
	case *RandomForestRegressor:
		return forestSchemas(model.trees)
	case *GradientBoostingRegressor:
		return forestSchemas(model.trees)
	case *GradientBoostingClassifier:
		return forestSchemas(model.trees)
	default:
		return nil
	}
}

// forestSchemas returns one representative feature schema for an ensemble.
// Every tree encodes from the same data with the same options, so the schemas
// agree; the first tree's stand for all of them.
func forestSchemas(trees []*treeFit) []treeFeature {
	if len(trees) == 0 {
		return nil
	}
	return trees[0].schemas
}

func treeInputTypes(features []string, schemas []treeFeature) (map[string]int32, error) {
	types := make(map[string]int32)
	if len(schemas) == 0 {
		return types, nil
	}
	if len(features) != len(schemas) {
		return nil, errors.New("ml: ONNX export tree schema does not match features")
	}
	for index, schema := range schemas {
		if !schema.categorical {
			continue
		}
		dtype, err := onnxCategoryType(schema.categoryVals[1:])
		if err != nil {
			return nil, fmt.Errorf("ml: ONNX export tree feature %q: %w", features[index], err)
		}
		types[features[index]] = dtype
	}
	return types, nil
}

func pipelineInputTypes(features []string, model Model, steps []fittedPipelineStep) (map[string]int32, error) {
	types := make(map[string]int32)
	rawFeatures := make(map[string]struct{}, len(features))
	for _, feature := range features {
		rawFeatures[feature] = struct{}{}
	}
	modelFeatures := model.Features()
	for index, schema := range treeSchemas(model) {
		if !schema.categorical || index >= len(modelFeatures) {
			continue
		}
		if _, raw := rawFeatures[modelFeatures[index]]; !raw {
			continue
		}
		dtype, err := onnxCategoryType(schema.categoryVals[1:])
		if err != nil {
			return nil, fmt.Errorf("tree feature %q: %w", modelFeatures[index], err)
		}
		types[modelFeatures[index]] = dtype
	}
	for _, step := range steps {
		if err := transformerInputTypes(types, step.transformer); err != nil {
			return nil, fmt.Errorf("ml: ONNX export pipeline step %q: %w", step.name, err)
		}
	}
	return types, nil
}

func transformerInputTypes(types map[string]int32, transformer Transformer) error {
	switch transformer := transformer.(type) {
	case *insyra.OneHotEncoder:
		categories := transformer.Categories()
		for _, column := range transformer.Columns() {
			dtype, err := onnxCategoryType(categories[column])
			if err != nil {
				return fmt.Errorf("encoder column %q: %w", column, err)
			}
			types[column] = dtype
		}
	case *insyra.LabelEncoder:
		dtype, err := onnxCategoryType(transformer.Classes())
		if err != nil {
			return fmt.Errorf("encoder column %q: %w", transformer.SourceColumn(), err)
		}
		types[transformer.SourceColumn()] = dtype
	case *insyra.OrdinalEncoder:
		dtype, err := onnxCategoryType(transformer.Classes())
		if err != nil {
			return fmt.Errorf("encoder column %q: %w", transformer.SourceColumn(), err)
		}
		types[transformer.SourceColumn()] = dtype
	case *ColumnTransformer:
		if transformer == nil || transformer.transformer == nil {
			return errors.New("column transformer is nil")
		}
		return transformerInputTypes(types, transformer.transformer)
	case *insyra.SimpleImputer, *insyra.StandardScaler, *insyra.MinMaxScaler, *insyra.RobustScaler, *insyra.MaxAbsScaler:
		return nil
	default:
		return nil
	}
	return nil
}

func onnxCategoryType(values []any) (int32, error) {
	var dtype int32
	for _, value := range values {
		if value == nil {
			continue
		}
		candidate := onnxValueType(value)
		if candidate == 0 {
			return 0, fmt.Errorf("category value %v has no ONNX scalar type", value)
		}
		if dtype == 0 {
			dtype = candidate
		} else if dtype != candidate {
			return 0, errors.New("categories have mixed scalar types")
		}
	}
	if dtype == 0 {
		return 0, errors.New("categories are empty")
	}
	return dtype, nil
}

func onnxValueType(value any) int32 {
	switch value.(type) {
	case string:
		return onnxTensorString
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return onnxTensorInt64
	case float32, float64:
		return onnxTensorFloat
	default:
		return 0
	}
}

func onnxSafeName(value string) string {
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "value"
	}
	return b.String()
}

func onnxAttrString(name, value string) onnxAttributeProto {
	return onnxAttributeProto{Name: name, Type: 3, String: []byte(value)}
}

func onnxAttrInt(name string, value int64) onnxAttributeProto {
	return onnxAttributeProto{Name: name, Type: 2, Int: value, HasInt: true}
}

func onnxAttrInts(name string, values []int64) onnxAttributeProto {
	return onnxAttributeProto{Name: name, Type: 7, Ints: append([]int64(nil), values...)}
}

func onnxAttrFloats(name string, values []float32) onnxAttributeProto {
	return onnxAttributeProto{Name: name, Type: 6, Floats: append([]float32(nil), values...)}
}

func onnxAttrStrings(name string, values []string) onnxAttributeProto {
	encoded := make([][]byte, len(values))
	for index, value := range values {
		encoded[index] = []byte(value)
	}
	return onnxAttributeProto{Name: name, Type: 8, Strings: encoded}
}

func onnxAttrTensor(name string, tensor onnxTensorProto) onnxAttributeProto {
	return onnxAttributeProto{Name: name, Type: 4, Tensor: &tensor}
}

func float32Slice(values []float64) []float32 {
	out := make([]float32, len(values))
	for index, value := range values {
		out[index] = float32(value)
	}
	return out
}

func flattenONNXGroups(groups []onnxGroup) []string {
	names := make([]string, len(groups))
	for index, group := range groups {
		names[index] = group.name
	}
	return names
}

func requireONNXFeatureOrder(groups []onnxGroup, features []string) error {
	got := flattenONNXGroups(groups)
	if len(got) != len(features) {
		return fmt.Errorf("ONNX model expects %d features after preprocessing, got %d", len(features), len(got))
	}
	for index := range got {
		if got[index] != features[index] {
			return fmt.Errorf("ONNX model feature %d is %q, want %q", index, got[index], features[index])
		}
		if groups[index].dtype == onnxTensorString {
			return fmt.Errorf("ONNX model feature %q remains a string after preprocessing", got[index])
		}
	}
	return nil
}

func finiteONNX(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return errors.New("fitted parameter is not finite")
	}
	return nil
}

func parseONNXInt(value any) (int64, bool) {
	switch value := value.(type) {
	case int:
		return int64(value), true
	case int8:
		return int64(value), true
	case int16:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	case uint:
		return int64(value), uint64(value) <= math.MaxInt64
	case uint8:
		return int64(value), true
	case uint16:
		return int64(value), true
	case uint32:
		return int64(value), true
	case uint64:
		return int64(value), value <= math.MaxInt64
	case bool:
		if value {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

func parseONNXFloat(value any) (float32, bool) {
	switch value := value.(type) {
	case float32:
		return value, true
	case float64:
		return float32(value), true
	default:
		return 0, false
	}
}

func parseONNXString(value any) (string, bool) {
	stringValue, ok := value.(string)
	return stringValue, ok
}

func sortedInt64(values map[uint32]struct{}) []int64 {
	out := make([]int64, 0, len(values))
	for value := range values {
		out = append(out, int64(value))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func onnxCategoryKey(value any) string {
	if value == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T:%v", value, value)
}

func onnxNumberName(value int64) string { return strconv.FormatInt(value, 10) }

func applyONNXTransformer(b *onnxBuilder, groups []onnxGroup, transformer Transformer) ([]onnxGroup, error) {
	switch transformer := transformer.(type) {
	case *ColumnTransformer:
		return applyONNXColumnTransformer(b, groups, transformer)
	case *insyra.StandardScaler:
		return applyONNXScaler(b, groups, transformer.Params(), transformer.Kind())
	case *insyra.MinMaxScaler:
		return applyONNXScaler(b, groups, transformer.Params(), transformer.Kind())
	case *insyra.RobustScaler:
		return applyONNXScaler(b, groups, transformer.Params(), transformer.Kind())
	case *insyra.MaxAbsScaler:
		return applyONNXScaler(b, groups, transformer.Params(), transformer.Kind())
	case *insyra.OneHotEncoder:
		return applyONNXOneHot(b, groups, transformer)
	case *insyra.LabelEncoder:
		return applyONNXScalarEncoder(b, groups, transformer.SourceColumn(), transformer.OutputColumn(), transformer.Classes(), transformer.Options().Unknown, transformer.Options().NewColumn != "", transformer.Options().KeepOriginal)
	case *insyra.OrdinalEncoder:
		return applyONNXScalarEncoder(b, groups, transformer.SourceColumn(), transformer.OutputColumn(), transformer.Classes(), transformer.Options().Unknown, transformer.Options().NewColumn != "", transformer.Options().KeepOriginal)
	case *insyra.SimpleImputer:
		return nil, errors.New("ONNX export does not support SimpleImputer in this change")
	case *PCATransformer:
		return nil, errors.New("ONNX export does not support PCA transformers")
	case nil:
		return nil, errors.New("ONNX export transformer is nil")
	default:
		return nil, fmt.Errorf("ONNX export does not support transformer %T", transformer)
	}
}

func applyONNXColumnTransformer(b *onnxBuilder, groups []onnxGroup, transformer *ColumnTransformer) ([]onnxGroup, error) {
	if transformer == nil || transformer.transformer == nil {
		return nil, errors.New("column transformer is nil")
	}
	if len(transformer.columns) == 0 {
		return nil, errors.New("column transformer has no selected columns")
	}
	selected := make(map[string]struct{}, len(transformer.columns))
	for _, name := range transformer.columns {
		selected[name] = struct{}{}
	}
	selectedGroups := make([]onnxGroup, 0, len(transformer.columns))
	selectedIndexes := make([]int, 0, len(transformer.columns))
	for index, group := range groups {
		if _, ok := selected[group.name]; !ok {
			continue
		}
		selectedGroups = append(selectedGroups, group)
		selectedIndexes = append(selectedIndexes, index)
	}
	if len(selectedGroups) != len(selected) {
		return nil, fmt.Errorf("column transformer selected column is missing from ONNX graph")
	}
	transformed, err := applyONNXTransformer(b, selectedGroups, transformer.transformer)
	if err != nil {
		return nil, err
	}
	first := selectedIndexes[0]
	selectedSet := make(map[int]struct{}, len(selectedIndexes))
	for _, index := range selectedIndexes {
		selectedSet[index] = struct{}{}
	}
	out := make([]onnxGroup, 0, len(groups)-len(selectedGroups)+len(transformed))
	for index, group := range groups {
		if index == first {
			out = append(out, transformed...)
		}
		if _, ok := selectedSet[index]; ok {
			continue
		}
		out = append(out, group)
	}
	return out, nil
}

func applyONNXScaler(b *onnxBuilder, groups []onnxGroup, params map[string]insyra.ScalerParams, kind string) ([]onnxGroup, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("ONNX export %s scaler is not fitted", kind)
	}
	out := append([]onnxGroup(nil), groups...)
	for index, group := range out {
		parameter, ok := params[group.name]
		if !ok {
			continue
		}
		if group.dtype == onnxTensorString {
			return nil, fmt.Errorf("scaler column %q is string", group.name)
		}
		group = b.castToFloat(group)
		center, scale, gain, offset := 0.0, 1.0, 1.0, 0.0
		switch kind {
		case "standard":
			center, scale = parameter.Mean, parameter.Std
		case "minmax":
			center, scale = parameter.Min, parameter.Max-parameter.Min
			gain, offset = parameter.OutputMax-parameter.OutputMin, parameter.OutputMin
		case "robust":
			center, scale = parameter.Median, parameter.IQR
		case "maxabs":
			scale = parameter.MaxAbs
		default:
			return nil, fmt.Errorf("scaler kind %q is not supported", kind)
		}
		if err := finiteONNX(center); err != nil {
			return nil, fmt.Errorf("scaler column %q center: %w", group.name, err)
		}
		if err := finiteONNX(scale); err != nil {
			return nil, fmt.Errorf("scaler column %q scale: %w", group.name, err)
		}
		if err := finiteONNX(gain); err != nil {
			return nil, fmt.Errorf("scaler column %q gain: %w", group.name, err)
		}
		if err := finiteONNX(offset); err != nil {
			return nil, fmt.Errorf("scaler column %q offset: %w", group.name, err)
		}
		if scale == 0 {
			scale = 1
		}
		group.value = b.addArithmetic("Sub", group.value, b.addFloatInitializer("center", float32(center)))
		group.value = b.addArithmetic("Div", group.value, b.addFloatInitializer("scale", float32(scale)))
		if gain != 1 {
			group.value = b.addArithmetic("Mul", group.value, b.addFloatInitializer("gain", float32(gain)))
		}
		if offset != 0 {
			group.value = b.addArithmetic("Add", group.value, b.addFloatInitializer("offset", float32(offset)))
		}
		group.dtype = onnxTensorFloat
		out[index] = group
	}
	return out, nil
}

func (b *onnxBuilder) addArithmetic(op, input, operand string) string {
	output := b.unique(strings.ToLower(op))
	b.addNode(op, "", []string{input, operand}, []string{output})
	return output
}

func applyONNXOneHot(b *onnxBuilder, groups []onnxGroup, encoder *insyra.OneHotEncoder) ([]onnxGroup, error) {
	if encoder == nil {
		return nil, errors.New("one-hot encoder is nil")
	}
	options := encoder.Options()
	if options.Unknown != insyra.UnknownIgnore {
		return nil, fmt.Errorf("one-hot encoder unknown policy %d cannot be represented by ONNX", options.Unknown)
	}
	columns := encoder.Columns()
	categories := encoder.Categories()
	outputNames := encoder.OutputColumnsByColumn()
	if len(columns) == 0 {
		return nil, errors.New("one-hot encoder has no fitted columns")
	}
	result := make([]onnxGroup, 0, len(groups)+len(encoder.OutputColumns()))
	for _, group := range groups {
		columnIndex := -1
		for index, name := range columns {
			if name == group.name {
				columnIndex = index
				break
			}
		}
		if columnIndex < 0 {
			result = append(result, group)
			continue
		}
		if options.KeepOriginal {
			result = append(result, group)
		}
		values := categories[group.name]
		start := 0
		if options.DropFirst && len(values) > 0 {
			start = 1
		}
		selected := values[start:]
		if len(selected) == 0 {
			continue
		}
		categoryType, err := onnxCategoryType(selected)
		if err != nil {
			return nil, fmt.Errorf("one-hot encoder column %q: %w", group.name, err)
		}
		if categoryType == onnxTensorFloat {
			return nil, fmt.Errorf("one-hot encoder column %q has floating categories", group.name)
		}
		if group.dtype != categoryType {
			return nil, fmt.Errorf("one-hot encoder column %q input type does not match fitted categories", group.name)
		}
		attrs := []onnxAttributeProto{onnxAttrInt("zeros", 1)}
		switch categoryType {
		case onnxTensorString:
			stringsValue := make([]string, len(selected))
			for index, value := range selected {
				var ok bool
				stringsValue[index], ok = parseONNXString(value)
				if !ok {
					return nil, fmt.Errorf("one-hot encoder column %q category: unsupported value %v", group.name, value)
				}
			}
			attrs = append(attrs, onnxAttrStrings("cats_strings", stringsValue))
		case onnxTensorInt64:
			ints := make([]int64, len(selected))
			for index, value := range selected {
				var ok bool
				ints[index], ok = parseONNXInt(value)
				if !ok {
					return nil, fmt.Errorf("one-hot encoder column %q category %v is not an integer", group.name, value)
				}
			}
			attrs = append(attrs, onnxAttrInts("cats_int64s", ints))
		}
		matrix := b.unique("onehot")
		b.addNode("OneHotEncoder", "ai.onnx.ml", []string{group.value}, []string{matrix}, attrs...)
		columnOutputNames, ok := outputNames[group.name]
		if !ok || len(columnOutputNames) != len(selected) {
			return nil, fmt.Errorf("one-hot encoder column %q output metadata is inconsistent", group.name)
		}
		for categoryIndex, name := range columnOutputNames {
			index := b.addInt64ScalarInitializer("category", int64(categoryIndex))
			value := b.unique("onehot_column")
			b.addNode("Gather", "", []string{matrix, index}, []string{value}, onnxAttrInt("axis", 1))
			result = append(result, onnxGroup{name: name, value: value, dtype: onnxTensorFloat})
		}
	}
	return result, nil
}

func applyONNXScalarEncoder(b *onnxBuilder, groups []onnxGroup, source, output string, classes []any, unknown insyra.UnknownPolicy, hasNewColumn, keepOriginal bool) ([]onnxGroup, error) {
	if source == "" || output == "" {
		return nil, errors.New("scalar encoder has no fitted column name")
	}
	if unknown != insyra.UnknownIgnore {
		return nil, fmt.Errorf("encoder column %q unknown policy %d cannot be represented by ONNX", source, unknown)
	}
	if len(classes) == 0 {
		return nil, fmt.Errorf("encoder column %q has no fitted classes", source)
	}
	categoryType, err := onnxCategoryType(classes)
	if err != nil {
		return nil, fmt.Errorf("encoder column %q: %w", source, err)
	}
	result := make([]onnxGroup, 0, len(groups)+1)
	found := false
	for _, group := range groups {
		if group.name != source {
			result = append(result, group)
			continue
		}
		found = true
		if group.dtype != categoryType {
			return nil, fmt.Errorf("encoder column %q input type does not match fitted classes", source)
		}
		if keepOriginal {
			result = append(result, group)
		}
		attrs := []onnxAttributeProto{onnxAttrInt("default_int64", -1)}
		values := make([]int64, len(classes))
		for index := range classes {
			values[index] = int64(index)
		}
		attrs = append(attrs, onnxAttrInts("values_int64s", values))
		switch categoryType {
		case onnxTensorString:
			keys := make([]string, len(classes))
			for index, value := range classes {
				keys[index], _ = parseONNXString(value)
			}
			attrs = append(attrs, onnxAttrStrings("keys_strings", keys))
		case onnxTensorInt64:
			keys := make([]int64, len(classes))
			for index, value := range classes {
				var ok bool
				keys[index], ok = parseONNXInt(value)
				if !ok {
					return nil, fmt.Errorf("encoder column %q class %v is not an integer", source, value)
				}
			}
			attrs = append(attrs, onnxAttrInts("keys_int64s", keys))
		case onnxTensorFloat:
			keys := make([]float32, len(classes))
			for index, value := range classes {
				var ok bool
				keys[index], ok = parseONNXFloat(value)
				if !ok {
					return nil, fmt.Errorf("encoder column %q class %v is not a float", source, value)
				}
			}
			attrs = append(attrs, onnxAttrFloats("keys_floats", keys))
		}
		encoded := b.unique("label_encode")
		b.addNode("LabelEncoder", "ai.onnx.ml", []string{group.value}, []string{encoded}, attrs...)
		encodedGroup := b.castToFloat(onnxGroup{name: output, value: encoded, dtype: onnxTensorInt64})
		zero := b.addInt64ScalarInitializer("label_zero", 0)
		known := b.unique("label_known")
		b.addNode("GreaterOrEqual", "", []string{encoded, zero}, []string{known})
		missing := b.addFloatInitializer("label_missing", float32(math.NaN()))
		value := b.unique("label_value")
		b.addNode("Where", "", []string{known, encodedGroup.value, missing}, []string{value})
		result = append(result, onnxGroup{name: output, value: value, dtype: onnxTensorFloat})
	}
	if !found {
		return nil, fmt.Errorf("encoder column %q is missing from ONNX graph", source)
	}
	_ = hasNewColumn
	return result, nil
}

func applyTreeInputEncoding(b *onnxBuilder, groups []onnxGroup, schemas []treeFeature) ([]onnxGroup, error) {
	if len(groups) != len(schemas) {
		return nil, fmt.Errorf("ONNX tree has %d fitted features but graph has %d", len(schemas), len(groups))
	}
	result := append([]onnxGroup(nil), groups...)
	for index, schema := range schemas {
		if !schema.categorical {
			continue
		}
		group := result[index]
		categoryType, err := onnxCategoryType(schema.categoryVals[1:])
		if err != nil {
			return nil, fmt.Errorf("tree feature %q: %w", schema.name, err)
		}
		if group.dtype != categoryType {
			return nil, fmt.Errorf("tree feature %q input type does not match fitted categories", schema.name)
		}
		attrs := []onnxAttributeProto{onnxAttrInt("default_int64", 0)}
		codes := schema.categoryVals[1:]
		values := make([]int64, len(codes))
		for code := range codes {
			values[code] = int64(code + 1)
		}
		attrs = append(attrs, onnxAttrInts("values_int64s", values))
		switch categoryType {
		case onnxTensorString:
			keys := make([]string, len(codes))
			for code, value := range codes {
				keys[code], _ = parseONNXString(value)
			}
			attrs = append(attrs, onnxAttrStrings("keys_strings", keys))
		case onnxTensorInt64:
			keys := make([]int64, len(codes))
			for code, value := range codes {
				var ok bool
				keys[code], ok = parseONNXInt(value)
				if !ok {
					return nil, fmt.Errorf("tree feature %q category %v is not an integer", schema.name, value)
				}
			}
			attrs = append(attrs, onnxAttrInts("keys_int64s", keys))
		case onnxTensorFloat:
			keys := make([]float32, len(codes))
			for code, value := range codes {
				var ok bool
				keys[code], ok = parseONNXFloat(value)
				if !ok {
					return nil, fmt.Errorf("tree feature %q category %v is not a float", schema.name, value)
				}
			}
			attrs = append(attrs, onnxAttrFloats("keys_floats", keys))
		}
		encoded := b.unique("tree_category")
		b.addNode("LabelEncoder", "ai.onnx.ml", []string{group.value}, []string{encoded}, attrs...)
		cast := b.castToFloat(onnxGroup{name: group.name, value: encoded, dtype: onnxTensorInt64})
		result[index] = cast
	}
	return result, nil
}

func appendONNXPredictor(b *onnxBuilder, model Model, groups []onnxGroup) error {
	if err := requireONNXFeatureOrder(groups, model.Features()); err != nil {
		return err
	}
	input, err := b.matrix(groups)
	if err != nil {
		return err
	}
	switch model := model.(type) {
	case *LinearModel:
		if model == nil || model.Result == nil || len(model.Result.Coefficients) != len(groups)+1 {
			return errors.New("ml: ONNX export linear model has invalid coefficients")
		}
		b.addNode("LinearRegressor", "ai.onnx.ml", []string{input}, []string{"prediction"},
			onnxAttrFloats("coefficients", float32Slice(model.Result.Coefficients[1:])),
			onnxAttrFloats("intercepts", []float32{float32(model.Result.Coefficients[0])}),
			onnxAttrInt("targets", 1), onnxAttrString("post_transform", "NONE"))
		b.addOutput("prediction", onnxTensorFloat, []int64{-1, 1})
		return nil
	case *LogisticModel:
		if model == nil || model.Result == nil || len(model.Result.Coefficients) != len(groups)+1 {
			return errors.New("ml: ONNX export logistic model has invalid coefficients")
		}
		attrs, err := onnxLinearClassLabelAttributes(model.Result.ClassLabels)
		if err != nil {
			return err
		}
		attrs = append(attrs,
			onnxAttrFloats("coefficients", float32Slice(model.Result.Coefficients[1:])),
			onnxAttrFloats("intercepts", []float32{float32(model.Result.Coefficients[0])}),
			onnxAttrInt("multi_class", 0), onnxAttrString("post_transform", "LOGISTIC"))
		b.addNode("LinearClassifier", "ai.onnx.ml", []string{input}, []string{"label", "probabilities"}, attrs...)
		labelType, _ := onnxClassLabelType(model.Result.ClassLabels)
		b.addOutput("label", labelType, []int64{-1})
		b.addOutput("probabilities", onnxTensorFloat, []int64{-1, int64(len(model.Result.ClassLabels))})
		return nil
	case *DecisionTreeClassifier:
		return appendONNXTreeClassifier(b, input, model)
	case *DecisionTreeRegressor:
		return appendONNXTreeRegressor(b, input, model)
	case *RidgeModel:
		if model == nil || model.Result == nil {
			return errors.New("ml: ONNX export ridge model is nil")
		}
		return appendONNXLinearRegressor(b, input, model.Result.Coefficients, len(groups))
	case *LassoModel:
		if model == nil || model.Result == nil {
			return errors.New("ml: ONNX export lasso model is nil")
		}
		return appendONNXLinearRegressor(b, input, model.Result.Coefficients, len(groups))
	case *WeightedLinearModel:
		if model == nil || model.Result == nil {
			return errors.New("ml: ONNX export weighted linear model is nil")
		}
		return appendONNXLinearRegressor(b, input, model.Result.Coefficients, len(groups))
	case *RandomForestClassifier:
		return appendONNXForestClassifier(b, input, model)
	case *RandomForestRegressor:
		return appendONNXForestRegressor(b, input, model)
	case *GradientBoostingRegressor:
		return appendONNXBoostedRegressor(b, input, model)
	case *GradientBoostingClassifier:
		return appendONNXBoostedClassifier(b, input, model)
	default:
		return unsupportedONNXModel(model)
	}
}

// appendONNXLinearRegressor writes the LinearRegressor node the plain linear
// path uses; ridge, lasso and WLS share it because their coefficient layout
// is the same — intercept first, one coefficient per feature.
func appendONNXLinearRegressor(b *onnxBuilder, input string, coefficients []float64, featureCount int) error {
	if len(coefficients) != featureCount+1 {
		return errors.New("ml: ONNX export linear model has invalid coefficients")
	}
	b.addNode("LinearRegressor", "ai.onnx.ml", []string{input}, []string{"prediction"},
		onnxAttrFloats("coefficients", float32Slice(coefficients[1:])),
		onnxAttrFloats("intercepts", []float32{float32(coefficients[0])}),
		onnxAttrInt("targets", 1), onnxAttrString("post_transform", "NONE"))
	b.addOutput("prediction", onnxTensorFloat, []int64{-1, 1})
	return nil
}

func appendONNXForestClassifier(b *onnxBuilder, input string, model *RandomForestClassifier) error {
	if model == nil || len(model.trees) == 0 || model.classes == nil {
		return errors.New("ml: ONNX export random-forest classifier is nil")
	}
	classes := model.classes.Data()
	// The runtime sums leaf contributions; scaling each tree's probabilities
	// by 1/T makes that sum the forest's average, which is what Predict does.
	ensemble := &onnxTreeEnsemble{leafScale: 1 / float64(len(model.trees))}
	for index, tree := range model.trees {
		ensemble.currentTree = int64(index)
		ensemble.emit(tree.root, true, len(classes), tree.schemas)
	}
	if len(classes) == 2 {
		// The runtime routes two-class ensembles through a single-score path:
		// one entry per leaf on class slot 0 carrying the second class's
		// probability, the complement and the half-threshold label computed by
		// the runtime. Leaving both classes' weights in place puts the label
		// through that binary path with the wrong convention — measured as
		// every label coming back 1 while the probabilities were exactly
		// right.
		ensemble.toBinaryScores()
	}
	attrs, err := ensemble.attributes(true, classes)
	if err != nil {
		return err
	}
	b.addNode("TreeEnsembleClassifier", "ai.onnx.ml", []string{input}, []string{"label", "probabilities"}, attrs...)
	labelType, _ := onnxClassLabelType(classes)
	b.addOutput("label", labelType, []int64{-1})
	b.addOutput("probabilities", onnxTensorFloat, []int64{-1, int64(len(classes))})
	return nil
}

func appendONNXForestRegressor(b *onnxBuilder, input string, model *RandomForestRegressor) error {
	if model == nil || len(model.trees) == 0 {
		return errors.New("ml: ONNX export random-forest regressor is nil")
	}
	ensemble := &onnxTreeEnsemble{leafScale: 1 / float64(len(model.trees))}
	for index, tree := range model.trees {
		ensemble.currentTree = int64(index)
		ensemble.emit(tree.root, false, 0, tree.schemas)
	}
	attrs, err := ensemble.attributes(false, nil)
	if err != nil {
		return err
	}
	b.addNode("TreeEnsembleRegressor", "ai.onnx.ml", []string{input}, []string{"prediction"}, attrs...)
	b.addOutput("prediction", onnxTensorFloat, []int64{-1, 1})
	return nil
}

func appendONNXBoostedRegressor(b *onnxBuilder, input string, model *GradientBoostingRegressor) error {
	if model == nil || len(model.trees) == 0 {
		return errors.New("ml: ONNX export gradient-boosting regressor is nil")
	}
	// base + Σ lr·leaf is exactly the runtime's base_values + sum over scaled
	// leaf weights, so the export is the model, not an approximation of it.
	ensemble := &onnxTreeEnsemble{
		leafScale:  model.learningRate,
		baseValues: []float32{float32(model.base)},
	}
	for index, tree := range model.trees {
		ensemble.currentTree = int64(index)
		ensemble.emit(tree.root, false, 0, tree.schemas)
	}
	attrs, err := ensemble.attributes(false, nil)
	if err != nil {
		return err
	}
	b.addNode("TreeEnsembleRegressor", "ai.onnx.ml", []string{input}, []string{"prediction"}, attrs...)
	b.addOutput("prediction", onnxTensorFloat, []int64{-1, 1})
	return nil
}

func appendONNXBoostedClassifier(b *onnxBuilder, input string, model *GradientBoostingClassifier) error {
	if model == nil || len(model.trees) == 0 || model.classes == nil {
		return errors.New("ml: ONNX export gradient-boosting classifier is nil")
	}
	classes := model.classes.Data()
	if len(classes) != 2 {
		return errors.New("ml: ONNX export gradient-boosting classifier requires two classes")
	}
	// The stage trees are regression trees over log-odds. They are emitted as
	// regression leaves and their entries rewritten onto class id 1: the
	// second class carries the raw score base + Σ lr·leaf, the runtime's
	// LOGISTIC transform turns it into a probability, and the runtime's
	// binary handling gives the first class the complement.
	ensemble := &onnxTreeEnsemble{
		leafScale:  model.learningRate,
		baseValues: []float32{float32(model.base)},
	}
	for index, tree := range model.trees {
		ensemble.currentTree = int64(index)
		ensemble.emit(tree.root, false, 0, tree.schemas)
	}
	// Regression leaves rewritten onto the binary single-score slot: class id
	// 0 carries the raw log-odds, the runtime's LOGISTIC turns it into the
	// second class's probability and fills the first with the complement.
	ensemble.classTreeIDs = ensemble.targetTreeIDs
	ensemble.classNodeIDs = ensemble.targetNodeIDs
	ensemble.classWeights = ensemble.targetWeights
	ensemble.classIDs = make([]int64, len(ensemble.targetIDs))
	ensemble.targetTreeIDs, ensemble.targetNodeIDs, ensemble.targetIDs, ensemble.targetWeights = nil, nil, nil, nil
	attrs, err := ensemble.attributesWithTransform(true, classes, "LOGISTIC")
	if err != nil {
		return err
	}
	b.addNode("TreeEnsembleClassifier", "ai.onnx.ml", []string{input}, []string{"label", "probabilities"}, attrs...)
	labelType, _ := onnxClassLabelType(classes)
	b.addOutput("label", labelType, []int64{-1})
	b.addOutput("probabilities", onnxTensorFloat, []int64{-1, 2})
	return nil
}

func onnxClassLabelType(classes []any) (int32, error) {
	if len(classes) == 0 {
		return 0, errors.New("ml: ONNX classifier has no class labels")
	}
	dtype := int32(0)
	for _, class := range classes {
		candidate := onnxValueType(class)
		if candidate == onnxTensorFloat {
			return 0, errors.New("ml: ONNX classifier does not support floating class labels")
		}
		if candidate == 0 {
			return 0, fmt.Errorf("ml: ONNX classifier class label %v has no scalar type", class)
		}
		if dtype == 0 {
			dtype = candidate
		} else if dtype != candidate {
			return 0, errors.New("ml: ONNX classifier class labels have mixed types")
		}
	}
	return dtype, nil
}

func onnxClassLabelAttributes(classes []any) ([]onnxAttributeProto, error) {
	dtype, err := onnxClassLabelType(classes)
	if err != nil {
		return nil, err
	}
	switch dtype {
	case onnxTensorString:
		values := make([]string, len(classes))
		for index, class := range classes {
			values[index], _ = parseONNXString(class)
		}
		return []onnxAttributeProto{onnxAttrStrings("classlabels_strings", values)}, nil
	case onnxTensorInt64:
		values := make([]int64, len(classes))
		for index, class := range classes {
			values[index], _ = parseONNXInt(class)
		}
		return []onnxAttributeProto{onnxAttrInts("classlabels_int64s", values)}, nil
	default:
		return nil, errors.New("ml: ONNX classifier class labels are unsupported")
	}
}

func onnxLinearClassLabelAttributes(classes []any) ([]onnxAttributeProto, error) {
	dtype, err := onnxClassLabelType(classes)
	if err != nil {
		return nil, err
	}
	switch dtype {
	case onnxTensorString:
		values := make([]string, len(classes))
		for index, class := range classes {
			values[index], _ = parseONNXString(class)
		}
		return []onnxAttributeProto{onnxAttrStrings("classlabels_strings", values)}, nil
	case onnxTensorInt64:
		values := make([]int64, len(classes))
		for index, class := range classes {
			values[index], _ = parseONNXInt(class)
		}
		return []onnxAttributeProto{onnxAttrInts("classlabels_ints", values)}, nil
	default:
		return nil, errors.New("ml: ONNX linear classifier class labels are unsupported")
	}
}

type onnxTreeEnsemble struct {
	nextID int64

	nodesTreeIDs      []int64
	nodesNodeIDs      []int64
	nodesFeatureIDs   []int64
	nodesValues       []float32
	nodesModes        []string
	nodesTrueNodeIDs  []int64
	nodesFalseNodeIDs []int64
	nodesMissingTrue  []int64

	classTreeIDs  []int64
	classNodeIDs  []int64
	classIDs      []int64
	classWeights  []float32
	targetTreeIDs []int64
	targetNodeIDs []int64
	targetIDs     []int64
	targetWeights []float32

	// currentTree is the tree id the next emit writes. The single-tree paths
	// leave it zero; an ensemble sets it per tree, and the runtime finds each
	// tree's root as the first array entry carrying its id.
	currentTree int64
	// leafScale multiplies every leaf contribution. The runtime aggregates by
	// summing, so 1/T turns the sum into a forest's average and a learning
	// rate turns it into a boosted stage's shrunk step. Zero means one.
	leafScale float64
	// baseValues, when set, becomes the ensemble's base_values attribute —
	// what the sum starts from, which for boosting is the prior.
	baseValues []float32
}

// toBinaryScores rewrites two-class entries — interleaved class 0, class 1
// per leaf — into the single-score form the runtime's binary path expects:
// one entry per leaf on class slot 0 carrying the second class's weight.
func (t *onnxTreeEnsemble) toBinaryScores() {
	treeIDs := make([]int64, 0, len(t.classIDs)/2)
	nodeIDs := make([]int64, 0, len(t.classIDs)/2)
	weights := make([]float32, 0, len(t.classIDs)/2)
	for i, classID := range t.classIDs {
		if classID != 1 {
			continue
		}
		treeIDs = append(treeIDs, t.classTreeIDs[i])
		nodeIDs = append(nodeIDs, t.classNodeIDs[i])
		weights = append(weights, t.classWeights[i])
	}
	t.classTreeIDs = treeIDs
	t.classNodeIDs = nodeIDs
	t.classWeights = weights
	t.classIDs = make([]int64, len(treeIDs))
}

func (t *onnxTreeEnsemble) scale() float64 {
	if t.leafScale == 0 {
		return 1
	}
	return t.leafScale
}

func (t *onnxTreeEnsemble) newID() int64 {
	id := t.nextID
	t.nextID++
	return id
}

func (t *onnxTreeEnsemble) addNode(id int64, feature int64, value float32, mode string, trueID, falseID int64, missingTrue bool) {
	t.nodesTreeIDs = append(t.nodesTreeIDs, t.currentTree)
	t.nodesNodeIDs = append(t.nodesNodeIDs, id)
	t.nodesFeatureIDs = append(t.nodesFeatureIDs, feature)
	t.nodesValues = append(t.nodesValues, value)
	t.nodesModes = append(t.nodesModes, mode)
	t.nodesTrueNodeIDs = append(t.nodesTrueNodeIDs, trueID)
	t.nodesFalseNodeIDs = append(t.nodesFalseNodeIDs, falseID)
	if missingTrue {
		t.nodesMissingTrue = append(t.nodesMissingTrue, 1)
	} else {
		t.nodesMissingTrue = append(t.nodesMissingTrue, 0)
	}
}

func (t *onnxTreeEnsemble) emitLeaf(node *DecisionTreeNode, classifier bool, classCount int) int64 {
	id := t.newID()
	t.addNode(id, -1, 0, "LEAF", 0, 0, false)
	if classifier {
		probabilities := append([]float64(nil), node.Probabilities...)
		if len(probabilities) < classCount {
			probabilities = make([]float64, classCount)
			if node.Samples > 0 {
				for class, count := range node.ClassCounts {
					if class < len(probabilities) {
						probabilities[class] = float64(count) / float64(node.Samples)
					}
				}
			}
		}
		for class := 0; class < classCount; class++ {
			t.classTreeIDs = append(t.classTreeIDs, t.currentTree)
			t.classNodeIDs = append(t.classNodeIDs, id)
			t.classIDs = append(t.classIDs, int64(class))
			weight := float32(0)
			if class < len(probabilities) {
				weight = float32(probabilities[class] * t.scale())
			}
			t.classWeights = append(t.classWeights, weight)
		}
	} else {
		t.targetTreeIDs = append(t.targetTreeIDs, t.currentTree)
		t.targetNodeIDs = append(t.targetNodeIDs, id)
		t.targetIDs = append(t.targetIDs, 0)
		t.targetWeights = append(t.targetWeights, float32(node.Value*t.scale()))
	}
	return id
}

// emit writes the subtree rooted at node and returns its node id.
//
// The traversal is pre-order on purpose: onnxruntime treats the FIRST array
// entry carrying each tree id as that tree's root and discovers every other
// node by following the true/false ids from there. The previous post-order
// form appended the deepest leaf first, so the runtime started its walk at a
// leaf, found exactly one reachable node, and rejected the model with
// "Number of nodes in nodes_ (1) is different from n_nodes (N)". A branch
// therefore appends its own row before recursing, with placeholder child ids
// patched once the children have allocated theirs.
func (t *onnxTreeEnsemble) emit(node *DecisionTreeNode, classifier bool, classCount int, schemas []treeFeature) int64 {
	if node == nil {
		return t.emitLeaf(&DecisionTreeNode{IsLeaf: true}, classifier, classCount)
	}
	if node.IsLeaf {
		return t.emitLeaf(node, classifier, classCount)
	}
	if node.Categorical {
		codes := sortedInt64(node.leftCategories)
		if node.MissingGoLeft {
			codes = append([]int64{0}, codes...)
		}
		if len(codes) == 0 {
			return t.emit(node.Right, classifier, classCount, schemas)
		}
		// The chain tests one category per node, falling through to the next;
		// the chain itself is emitted head first so the runtime's walk enters
		// at the top of it.
		rows := make([]int, len(codes))
		ids := make([]int64, len(codes))
		for index := range codes {
			ids[index] = t.newID()
			rows[index] = len(t.nodesNodeIDs)
			t.addNode(ids[index], int64(node.Feature), float32(codes[index]), "BRANCH_EQ", 0, 0, false)
		}
		leftID := t.emit(node.Left, classifier, classCount, schemas)
		rightID := t.emit(node.Right, classifier, classCount, schemas)
		for index := range codes {
			t.nodesTrueNodeIDs[rows[index]] = leftID
			if index+1 < len(codes) {
				t.nodesFalseNodeIDs[rows[index]] = ids[index+1]
			} else {
				t.nodesFalseNodeIDs[rows[index]] = rightID
			}
		}
		return ids[0]
	}
	id := t.newID()
	row := len(t.nodesNodeIDs)
	t.addNode(id, int64(node.Feature), float32(node.Threshold), "BRANCH_LEQ", 0, 0, node.MissingGoLeft)
	leftID := t.emit(node.Left, classifier, classCount, schemas)
	rightID := t.emit(node.Right, classifier, classCount, schemas)
	t.nodesTrueNodeIDs[row] = leftID
	t.nodesFalseNodeIDs[row] = rightID
	return id
}

func (t *onnxTreeEnsemble) attributes(classifier bool, classes []any) ([]onnxAttributeProto, error) {
	return t.attributesWithTransform(classifier, classes, "NONE")
}

func (t *onnxTreeEnsemble) attributesWithTransform(classifier bool, classes []any, postTransform string) ([]onnxAttributeProto, error) {
	attrs := []onnxAttributeProto{}
	if len(t.baseValues) > 0 {
		attrs = append(attrs, onnxAttrFloats("base_values", t.baseValues))
	}
	attrs = append(attrs,
		onnxAttrInts("nodes_treeids", t.nodesTreeIDs),
		onnxAttrInts("nodes_nodeids", t.nodesNodeIDs),
		onnxAttrInts("nodes_featureids", t.nodesFeatureIDs),
		onnxAttrFloats("nodes_values", t.nodesValues),
		onnxAttrStrings("nodes_modes", t.nodesModes),
		onnxAttrInts("nodes_truenodeids", t.nodesTrueNodeIDs),
		onnxAttrInts("nodes_falsenodeids", t.nodesFalseNodeIDs),
		onnxAttrInts("nodes_missing_value_tracks_true", t.nodesMissingTrue),
		onnxAttrString("post_transform", postTransform),
	)
	if classifier {
		classAttrs, err := onnxClassLabelAttributes(classes)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, classAttrs...)
		attrs = append(attrs,
			onnxAttrInts("class_treeids", t.classTreeIDs),
			onnxAttrInts("class_nodeids", t.classNodeIDs),
			onnxAttrInts("class_ids", t.classIDs),
			onnxAttrFloats("class_weights", t.classWeights))
	} else {
		attrs = append(attrs,
			onnxAttrInt("n_targets", 1),
			onnxAttrInts("target_treeids", t.targetTreeIDs),
			onnxAttrInts("target_nodeids", t.targetNodeIDs),
			onnxAttrInts("target_ids", t.targetIDs),
			onnxAttrFloats("target_weights", t.targetWeights))
	}
	return attrs, nil
}

func appendONNXTreeClassifier(b *onnxBuilder, input string, model *DecisionTreeClassifier) error {
	if model == nil || model.Root == nil || model.classes == nil {
		return errors.New("ml: ONNX export decision-tree classifier is nil")
	}
	classes := model.classes.Data()
	tree := &onnxTreeEnsemble{}
	tree.emit(model.Root, true, len(classes), model.featureSchemas)
	attrs, err := tree.attributes(true, classes)
	if err != nil {
		return err
	}
	b.addNode("TreeEnsembleClassifier", "ai.onnx.ml", []string{input}, []string{"label", "probabilities"}, attrs...)
	labelType, _ := onnxClassLabelType(classes)
	b.addOutput("label", labelType, []int64{-1})
	b.addOutput("probabilities", onnxTensorFloat, []int64{-1, int64(len(classes))})
	return nil
}

func appendONNXTreeRegressor(b *onnxBuilder, input string, model *DecisionTreeRegressor) error {
	if model == nil || model.Root == nil {
		return errors.New("ml: ONNX export decision-tree regressor is nil")
	}
	tree := &onnxTreeEnsemble{}
	tree.emit(model.Root, false, 0, model.featureSchemas)
	attrs, err := tree.attributes(false, nil)
	if err != nil {
		return err
	}
	b.addNode("TreeEnsembleRegressor", "ai.onnx.ml", []string{input}, []string{"prediction"}, attrs...)
	b.addOutput("prediction", onnxTensorFloat, []int64{-1, 1})
	return nil
}
