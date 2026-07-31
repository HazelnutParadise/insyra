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
			return &fittedPipeline{features: features, steps: fitted, model: model}, nil
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
		indexByName[name] = i
	}
	selected := make(map[int]struct{}, len(t.columns))
	for _, name := range t.columns {
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
	for idx, name := range names {
		if _, ok := selected[idx]; !ok {
			continue
		}
		selectedIndexes = append(selectedIndexes, idx)
		selectedColumns = append(selectedColumns, dt.GetColByName(name))
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
		for idx, name := range names {
			if col, ok := byIndex[idx]; ok {
				outColumns = append(outColumns, col)
				continue
			}
			outColumns = append(outColumns, dt.GetColByName(name))
		}
	} else {
		first := selectedIndexes[0]
		for idx, name := range names {
			if idx == first {
				outColumns = append(outColumns, transformedColumns...)
			}
			if _, ok := selected[idx]; ok {
				continue
			}
			outColumns = append(outColumns, dt.GetColByName(name))
		}
	}

	out := insyra.NewDataTable(outColumns...)
	out.SetName(dt.GetName())
	out.SetRowNames(dt.RowNames())
	return out, nil
}

type fittedPipeline struct {
	features []string
	steps    []fittedPipelineStep
	model    Model
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

func (p *fittedPipeline) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	if p == nil || p.model == nil || isNilPointer(p.model) {
		return nil, fmt.Errorf("ml: fitted pipeline is nil")
	}
	if err := requireTable(dt); err != nil {
		return nil, err
	}
	if _, err := orderedFeatures(dt, p.features); err != nil {
		return nil, err
	}

	current := dt
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
	return p.model.Predict(current)
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
