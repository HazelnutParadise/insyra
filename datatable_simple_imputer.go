package insyra

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
)

// ImputationStrategy selects how a SimpleImputer derives replacements.
type ImputationStrategy string

const (
	ImputeMean     ImputationStrategy = "mean"
	ImputeMedian   ImputationStrategy = "median"
	ImputeMode     ImputationStrategy = "mode"
	ImputeConstant ImputationStrategy = "constant"
)

// SimpleImputer is a fitted, reusable missing-value transformer.
//
// Fit derives one replacement per selected column. Transform always reuses
// those replacements, so applying it to validation or production data cannot
// leak statistics from that data back into the transformation.
type SimpleImputer struct {
	strategy  ImputationStrategy
	constant  any
	configErr error
	columns   []simpleImputerColumn
	fitted    bool
}

type simpleImputerColumn struct {
	ref         any // original fit-time reference, retained for inspection/debugging
	name        string
	replacement any
}

// SimpleImputerOptions configures NewSimpleImputer. Both settings decide how
// the missing values are filled, so they travel together.
type SimpleImputerOptions struct {
	// Strategy is how each column's fill value is computed. Empty means
	// ImputeMean, the same default as scikit-learn's SimpleImputer.
	Strategy ImputationStrategy
	// FillValue is the value ImputeConstant fills with. It is required for
	// ImputeConstant and must be left nil for every other strategy.
	FillValue any
}

// NewSimpleImputer returns an unfitted imputer: it learns one fill value per
// column from the table it is fitted on and fills other tables with those
// values, so a test set is filled with what the training set taught it. With
// no options it fills with each column's mean. Invalid options are reported by
// Fit, matching the error-returning lifecycle of the Scaler interface.
func NewSimpleImputer(opts ...SimpleImputerOptions) *SimpleImputer {
	if msg := extraOptional("SimpleImputerOptions", len(opts)); msg != "" {
		return &SimpleImputer{strategy: ImputeMean, configErr: errors.New("SimpleImputer: " + msg)}
	}
	var o SimpleImputerOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Strategy == "" {
		o.Strategy = ImputeMean
	}
	return &SimpleImputer{strategy: o.Strategy, constant: o.FillValue}
}

// Kind reports the imputer family and configured strategy.
func (i *SimpleImputer) Kind() string {
	if i == nil {
		return "imputer"
	}
	return "imputer-" + string(i.strategy)
}

// Params returns the fitted replacement keyed by output column name.
func (i *SimpleImputer) Params() map[string]ScalerParams {
	if i == nil {
		return nil
	}
	out := make(map[string]ScalerParams, len(i.columns))
	for _, column := range i.columns {
		out[column.name] = ScalerParams{
			Column:      column.name,
			Kind:        string(i.strategy),
			Replacement: column.replacement,
		}
	}
	return out
}

// Fit derives replacements from selected columns without modifying dt.
func (i *SimpleImputer) Fit(dt *DataTable, cols ...any) error {
	if i == nil {
		return errors.New("SimpleImputer.Fit: imputer is nil")
	}
	if dt == nil {
		return errors.New("SimpleImputer.Fit: table is nil")
	}
	if len(cols) == 0 {
		return errors.New("SimpleImputer.Fit: at least one column is required")
	}
	if err := i.validateConfiguration(); err != nil {
		return err
	}

	fitted := make([]simpleImputerColumn, 0, len(cols))
	var err error
	dt.AtomicDo(func(t *DataTable) {
		seen := make(map[int]struct{}, len(cols))
		for _, ref := range cols {
			idx, label, problem := resolveEncodingColumn(t, ref)
			if problem != "" {
				err = fmt.Errorf("SimpleImputer.Fit: %s", problem)
				return
			}
			if _, duplicate := seen[idx]; duplicate {
				err = fmt.Errorf("SimpleImputer.Fit: column %q listed more than once", ref)
				return
			}
			seen[idx] = struct{}{}

			name := label
			if t.columns[idx].name != "" {
				name = t.columns[idx].name
			}
			replacement, deriveErr := i.deriveReplacement(name, t.columns[idx].data)
			if deriveErr != nil {
				err = deriveErr
				return
			}
			fittedRef := any(idx)
			if t.columns[idx].name != "" {
				fittedRef = Name(t.columns[idx].name)
			}
			fitted = append(fitted, simpleImputerColumn{
				ref:         fittedRef,
				name:        name,
				replacement: replacement,
			})
		}
	})
	if err != nil {
		return err
	}
	i.columns = fitted
	i.fitted = true
	return nil
}

// FitTransform fits on cols and returns a transformed copy of dt.
func (i *SimpleImputer) FitTransform(dt *DataTable, cols ...any) (*DataTable, error) {
	if err := i.Fit(dt, cols...); err != nil {
		return nil, err
	}
	return i.Transform(dt)
}

// Transform applies fitted replacements and returns a new table.
func (i *SimpleImputer) Transform(dt *DataTable) (*DataTable, error) {
	if i == nil {
		return nil, errors.New("SimpleImputer.Transform: imputer is nil")
	}
	if !i.fitted {
		return nil, errors.New("SimpleImputer.Transform: imputer is not fitted")
	}
	if dt == nil {
		return nil, errors.New("SimpleImputer.Transform: table is nil")
	}

	out := NewDataTable()
	var err error
	dt.AtomicDo(func(t *DataTable) {
		byIndex := make(map[int]*simpleImputerColumn, len(i.columns))
		for idx := range i.columns {
			column := &i.columns[idx]
			resolved, _, problem := resolveEncodingColumn(t, Name(column.name))
			if problem != "" {
				err = fmt.Errorf("SimpleImputer.Transform: fitted column %q not found", column.name)
				return
			}
			byIndex[resolved] = column
		}

		outColumns := make([]*DataList, 0, len(t.columns))
		for idx, source := range t.columns {
			column, selected := byIndex[idx]
			if !selected {
				outColumns = append(outColumns, source.Clone())
				continue
			}
			copy := source.Clone()
			for row, value := range copy.data {
				if isMissing(value) {
					copy.data[row] = column.replacement
				}
			}
			outColumns = append(outColumns, copy)
		}
		out.AppendCols(outColumns...)
		copyRowNamesNotAtomic(out, t)
		out.name = t.name
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (i *SimpleImputer) validateConfiguration() error {
	if i.configErr != nil {
		return i.configErr
	}
	switch i.strategy {
	case ImputeMean, ImputeMedian, ImputeMode:
		if i.constant != nil {
			return fmt.Errorf("SimpleImputer: strategy %q does not use FillValue; leave it nil or use ImputeConstant", i.strategy)
		}
	case ImputeConstant:
		if i.constant == nil {
			return errors.New("SimpleImputer: ImputeConstant requires a FillValue")
		}
	default:
		return fmt.Errorf("SimpleImputer: unsupported strategy %q", i.strategy)
	}
	return nil
}

func (i *SimpleImputer) deriveReplacement(name string, data []any) (any, error) {
	if i.strategy == ImputeConstant {
		if !hasObservedValues(data) {
			return nil, fmt.Errorf("SimpleImputer.Fit: column %q has no observed values", name)
		}
		return i.constant, nil
	}

	observed := make([]any, 0, len(data))
	for _, value := range data {
		if !isMissing(value) {
			observed = append(observed, value)
		}
	}
	if len(observed) == 0 {
		return nil, fmt.Errorf("SimpleImputer.Fit: column %q has no observed values", name)
	}

	switch i.strategy {
	case ImputeMean, ImputeMedian:
		values := make([]float64, len(observed))
		for index, value := range observed {
			converted, ok := ToFloat64Safe(value)
			if !ok {
				// The caller selected this column, so leaving it unfilled
				// would let the fit look complete when it is not.
				return nil, fmt.Errorf("SimpleImputer.Fit: column %q holds %s values, and %s needs a number column; use ImputeMode or ImputeConstant for it", name, dataTypeOf(data), i.strategy)
			}
			values[index] = converted
		}
		if i.strategy == ImputeMean {
			return meanOf(values), nil
		}
		sort.Float64s(values)
		middle := len(values) / 2
		if len(values)%2 == 1 {
			return values[middle], nil
		}
		return (values[middle-1] + values[middle]) / 2, nil
	case ImputeMode:
		return firstMode(observed), nil
	default:
		return nil, fmt.Errorf("SimpleImputer: unsupported strategy %q", i.strategy)
	}
}

func hasObservedValues(data []any) bool {
	for _, value := range data {
		if !isMissing(value) {
			return true
		}
	}
	return false
}

func firstMode(values []any) any {
	type entry struct {
		value any
		count int
		first int
	}
	entries := make([]entry, 0, len(values))
	for index, value := range values {
		found := false
		for position := range entries {
			if reflect.DeepEqual(entries[position].value, value) {
				entries[position].count++
				found = true
				break
			}
		}
		if !found {
			entries = append(entries, entry{value: value, count: 1, first: index})
		}
	}
	best := entries[0]
	for _, candidate := range entries[1:] {
		if candidate.count > best.count || candidate.count == best.count && candidate.first < best.first {
			best = candidate
		}
	}
	return best.value
}

// SimpleImputer deliberately does NOT implement Scaler, and does not carry an
// InverseTransform at all.
//
// Imputation is not reversible: once a missing cell holds the fitted value,
// nothing records that it was ever missing. An InverseTransform that always
// returned an error would still satisfy every interface that asks for the
// method, so a caller testing for the capability by type assertion would be
// told the capability is present and then be refused at the call. Not having
// the method is the only form of that answer a type assertion can read.
//
// What it does satisfy is the transformer shape, which is all a pipeline needs.
var _ interface {
	Transform(dt *DataTable) (*DataTable, error)
} = (*SimpleImputer)(nil)
