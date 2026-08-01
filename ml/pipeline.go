package ml

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
)

// NewPipeline builds an unfitted estimator from preprocessing steps and a
// final estimator. Every call to the returned estimator's Fit function fits
// every step again, so the definition can safely be reused for validation or
// for a second training set.
func NewPipeline(steps []Step, estimator Estimator) Estimator {
	definition := append([]Step(nil), steps...)
	return Estimator{
		Name: estimator.Name,
		Fit: func(x *insyra.DataTable, y *insyra.DataList) (Model, error) {
			if err := requireTable(x); err != nil {
				return nil, err
			}

			features := x.ColNames()
			if err := validateFeatureNames(features); err != nil {
				return nil, err
			}
			current := x
			fitted := make([]fittedPipelineStep, 0, len(definition))
			for i, step := range definition {
				name := pipelineStepName(step.Name, i)
				if step.Fit == nil {
					return nil, fmt.Errorf("ml: pipeline step %q has no fit function", name)
				}
				transformer, err := step.Fit(current, y)
				if err != nil {
					return nil, fmt.Errorf("ml: pipeline step %q fit: %w", name, err)
				}
				if transformer == nil || isNilPointer(transformer) {
					return nil, fmt.Errorf("ml: pipeline step %q fit returned a nil transformer", name)
				}
				transformed, err := transformer.Transform(current)
				if err != nil {
					return nil, fmt.Errorf("ml: pipeline step %q transform: %w", name, err)
				}
				if transformed == nil || isNilPointer(transformed) {
					return nil, fmt.Errorf("ml: pipeline step %q returned a nil table", name)
				}
				fitted = append(fitted, fittedPipelineStep{name: name, transformer: transformer})
				current = transformed
			}

			if estimator.Fit == nil {
				return nil, fmt.Errorf("ml: pipeline estimator %q has no fit function", pipelineEstimatorName(estimator.Name))
			}
			model, err := estimator.Fit(current, y)
			if err != nil {
				return nil, fmt.Errorf("ml: pipeline estimator %q fit: %w", pipelineEstimatorName(estimator.Name), err)
			}
			if model == nil || isNilPointer(model) {
				return nil, fmt.Errorf("ml: pipeline estimator %q fit returned a nil model", pipelineEstimatorName(estimator.Name))
			}
			if len(definition) == 0 {
				features = model.Features()
			}
			// The names the estimator actually saw are in hand right here —
			// fitting ran every transform, so `current` is the fully
			// transformed table. They were previously discarded, which left a
			// pipeline's feature importances as a list of anonymous numbers
			// whenever a step changed the column count.
			transformed := current.ColNames()
			if len(definition) == 0 {
				transformed = append([]string(nil), features...)
			}
			return wrapFittedPipeline(&fittedPipeline{
				features:    features,
				transformed: transformed,
				steps:       fitted,
				model:       model,
			}), nil
		},
	}
}

// ColumnTransformer applies a fitted transformer to a named subset of
// columns. Columns outside the subset pass through unchanged.
type ColumnTransformer struct {
	transformer Transformer
	columns     []string
}

// NewColumnTransformer scopes transformer to columns. The transformer is
// fitted separately; this wrapper only controls which columns it sees during
// Transform.
func NewColumnTransformer(transformer Transformer, columns ...string) *ColumnTransformer {
	return &ColumnTransformer{
		transformer: transformer,
		columns:     append([]string(nil), columns...),
	}
}

// TransformColumns is the functional spelling of NewColumnTransformer.
func TransformColumns(transformer Transformer, columns ...string) Transformer {
	return NewColumnTransformer(transformer, columns...)
}

// Transform applies the wrapped transformer to only the configured columns.
func (t *ColumnTransformer) Transform(dt *insyra.DataTable) (*insyra.DataTable, error) {
	if t == nil || t.transformer == nil || isNilPointer(t.transformer) {
		return nil, fmt.Errorf("ml: column transformer is nil")
	}
	if err := requireTable(dt); err != nil {
		return nil, err
	}
	if len(t.columns) == 0 {
		return nil, fmt.Errorf("ml: column transformer requires at least one column")
	}

	names := dt.ColNames()
	indexByName := make(map[string]int, len(names))
	for i, name := range names {
		if name == "" {
			continue
		}
		if _, duplicate := indexByName[name]; duplicate {
			return nil, fmt.Errorf("ml: column transformer column %q is not unique", name)
		}
		indexByName[name] = i
	}
	selected := make(map[int]struct{}, len(t.columns))
	for _, name := range t.columns {
		if name == "" {
			return nil, fmt.Errorf("ml: column transformer requires named columns")
		}
		idx, ok := indexByName[name]
		if !ok {
			return nil, fmt.Errorf("ml: column transformer column %q not found", name)
		}
		if _, duplicate := selected[idx]; duplicate {
			return nil, fmt.Errorf("ml: column transformer column %q listed more than once", name)
		}
		selected[idx] = struct{}{}
	}

	selectedColumns := make([]*insyra.DataList, 0, len(selected))
	selectedIndexes := make([]int, 0, len(selected))
	for idx := range names {
		if _, ok := selected[idx]; !ok {
			continue
		}
		selectedIndexes = append(selectedIndexes, idx)
		selectedColumns = append(selectedColumns, dt.GetColByNumber(idx))
	}
	selectedTable := insyra.NewDataTable(selectedColumns...)
	selectedTable.SetName(dt.GetName())
	selectedTable.SetRowNames(dt.RowNames())

	transformed, err := t.transformer.Transform(selectedTable)
	if err != nil {
		return nil, err
	}
	if transformed == nil || isNilPointer(transformed) {
		return nil, fmt.Errorf("ml: column transformer returned a nil table")
	}
	if transformed.NumRows() != dt.NumRows() {
		return nil, fmt.Errorf("ml: column transformer changed row count from %d to %d", dt.NumRows(), transformed.NumRows())
	}
	transformedColumns := make([]*insyra.DataList, transformed.NumCols())
	for i := range transformedColumns {
		transformedColumns[i] = transformed.GetColByNumber(i)
	}

	outColumns := make([]*insyra.DataList, 0, len(names)-len(selectedIndexes)+len(transformedColumns))
	if len(transformedColumns) == len(selectedIndexes) {
		byIndex := make(map[int]*insyra.DataList, len(selectedIndexes))
		for i, idx := range selectedIndexes {
			byIndex[idx] = transformedColumns[i]
		}
		for idx := range names {
			if col, ok := byIndex[idx]; ok {
				outColumns = append(outColumns, col)
				continue
			}
			outColumns = append(outColumns, dt.GetColByNumber(idx))
		}
	} else {
		first := selectedIndexes[0]
		for idx := range names {
			if idx == first {
				outColumns = append(outColumns, transformedColumns...)
			}
			if _, ok := selected[idx]; ok {
				continue
			}
			outColumns = append(outColumns, dt.GetColByNumber(idx))
		}
	}

	out := insyra.NewDataTable(outColumns...)
	out.SetName(dt.GetName())
	out.SetRowNames(dt.RowNames())
	return out, nil
}

type fittedPipeline struct {
	features    []string
	transformed []string
	steps       []fittedPipelineStep
	model       Model
}

type fittedPipelineStep struct {
	name        string
	transformer Transformer
}

func (p *fittedPipeline) Features() []string {
	if p == nil {
		return nil
	}
	return append([]string(nil), p.features...)
}

// TransformedFeatureNames returns the columns the final estimator was fitted
// on, after every step had been applied. Feature importances reported by a
// pipeline are indexed by these, not by the columns the pipeline itself was
// fitted on.
func (p *fittedPipeline) TransformedFeatureNames() []string {
	if p == nil {
		return nil
	}
	return append([]string(nil), p.transformed...)
}

func (p *fittedPipeline) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	if p == nil || p.model == nil || isNilPointer(p.model) {
		return nil, fmt.Errorf("ml: fitted pipeline is nil")
	}
	current, err := p.transformInput(dt)
	if err != nil {
		return nil, err
	}
	return p.model.Predict(current)
}

func (p *fittedPipeline) transformInput(dt *insyra.DataTable) (*insyra.DataTable, error) {
	if p == nil || p.model == nil || isNilPointer(p.model) {
		return nil, fmt.Errorf("ml: fitted pipeline is nil")
	}
	current, err := orderedTable(dt, p.features)
	if err != nil {
		return nil, err
	}
	for _, step := range p.steps {
		transformed, err := step.transformer.Transform(current)
		if err != nil {
			return nil, fmt.Errorf("ml: pipeline step %q transform: %w", step.name, err)
		}
		if transformed == nil || isNilPointer(transformed) {
			return nil, fmt.Errorf("ml: pipeline step %q returned a nil table", step.name)
		}
		current = transformed
	}
	return current, nil
}

type fittedPipelineClassifier struct{ *fittedPipeline }

func (p *fittedPipelineClassifier) Classes() *insyra.DataList {
	model, ok := p.model.(Classifier)
	if !ok {
		return nil
	}
	return model.Classes()
}

type fittedPipelineProba struct{ *fittedPipeline }

func (p *fittedPipelineProba) Classes() *insyra.DataList {
	model, ok := p.model.(ProbaModel)
	if !ok {
		return nil
	}
	return model.Classes()
}

func (p *fittedPipelineProba) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error) {
	model, ok := p.model.(ProbaModel)
	if !ok {
		return nil, fmt.Errorf("ml: fitted pipeline model does not provide probabilities")
	}
	current, err := p.transformInput(dt)
	if err != nil {
		return nil, err
	}
	return model.PredictProba(current)
}

type fittedPipelineImportances struct{ *fittedPipeline }

func (p *fittedPipelineImportances) FeatureImportances() []float64 {
	model, ok := p.model.(Importances)
	if !ok {
		return nil
	}
	return model.FeatureImportances()
}

type fittedPipelineClassifierImportances struct {
	*fittedPipeline
}

func (p *fittedPipelineClassifierImportances) Classes() *insyra.DataList {
	model, ok := p.model.(Classifier)
	if !ok {
		return nil
	}
	return model.Classes()
}

func (p *fittedPipelineClassifierImportances) FeatureImportances() []float64 {
	model, ok := p.model.(Importances)
	if !ok {
		return nil
	}
	return model.FeatureImportances()
}

type fittedPipelineProbaImportances struct{ *fittedPipeline }

func (p *fittedPipelineProbaImportances) Classes() *insyra.DataList {
	model, ok := p.model.(ProbaModel)
	if !ok {
		return nil
	}
	return model.Classes()
}

func (p *fittedPipelineProbaImportances) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error) {
	model, ok := p.model.(ProbaModel)
	if !ok {
		return nil, fmt.Errorf("ml: fitted pipeline model does not provide probabilities")
	}
	current, err := p.transformInput(dt)
	if err != nil {
		return nil, err
	}
	return model.PredictProba(current)
}

func (p *fittedPipelineProbaImportances) FeatureImportances() []float64 {
	model, ok := p.model.(Importances)
	if !ok {
		return nil
	}
	return model.FeatureImportances()
}

func wrapFittedPipeline(p *fittedPipeline) Model {
	_, hasProba := p.model.(ProbaModel)
	_, hasClassifier := p.model.(Classifier)
	_, hasImportances := p.model.(Importances)
	switch {
	case hasProba && hasImportances:
		return &fittedPipelineProbaImportances{fittedPipeline: p}
	case hasProba:
		return &fittedPipelineProba{fittedPipeline: p}
	case hasClassifier && hasImportances:
		return &fittedPipelineClassifierImportances{fittedPipeline: p}
	case hasClassifier:
		return &fittedPipelineClassifier{fittedPipeline: p}
	case hasImportances:
		return &fittedPipelineImportances{fittedPipeline: p}
	default:
		return p
	}
}

func pipelineStepName(name string, index int) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("step-%d", index+1)
}

func pipelineEstimatorName(name string) string {
	if name != "" {
		return name
	}
	return "estimator"
}

var _ Transformer = (*ColumnTransformer)(nil)
var _ Model = (*fittedPipeline)(nil)

// Every wrapper embeds *fittedPipeline, so the capability survives whichever
// combination of Classifier, ProbaModel and Importances the inner model has.
var (
	_ TransformedFeatures = (*fittedPipeline)(nil)
	_ TransformedFeatures = (*fittedPipelineClassifier)(nil)
	_ TransformedFeatures = (*fittedPipelineProba)(nil)
	_ TransformedFeatures = (*fittedPipelineImportances)(nil)
	_ TransformedFeatures = (*fittedPipelineClassifierImportances)(nil)
	_ TransformedFeatures = (*fittedPipelineProbaImportances)(nil)
)
