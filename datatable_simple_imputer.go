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
	strategy     ImputationStrategy
	constant     any
	constantArgs int
	columns      []simpleImputerColumn
	fitted       bool
}

type simpleImputerColumn struct {
	ref         string // original fit-time reference, retained for inspection/debugging
	name        string
	replacement any
	passThrough bool
}

// NewSimpleImputer returns an unfitted imputer. A constant value is required
// for ImputeConstant and must be omitted for every other strategy. Invalid
// constructor arguments are reported by Fit, matching the error-returning
// lifecycle of the existing Scaler interface.
func NewSimpleImputer(strategy ImputationStrategy, constant ...any) *SimpleImputer {
	var value any
	if len(constant) == 1 {
		value = constant[0]
	}
	return &SimpleImputer{
		strategy:     strategy,
		constant:     value,
		constantArgs: len(constant),
	}
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
			PassThrough: column.passThrough,
		}
	}
	return out
}

// Fit derives replacements from selected columns without modifying dt.
func (i *SimpleImputer) Fit(dt *DataTable, cols ...string) error {
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
			idx, label, ok := resolveEncodingColumn(t, ref)
			if !ok {
				err = fmt.Errorf("SimpleImputer.Fit: column %q not found", ref)
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
			replacement, passThrough, deriveErr := i.deriveReplacement(name, t.columns[idx].data)
			if deriveErr != nil {
				err = deriveErr
				return
			}
			fitted = append(fitted, simpleImputerColumn{
				ref:         ref,
				name:        name,
				replacement: replacement,
				passThrough: passThrough,
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
func (i *SimpleImputer) FitTransform(dt *DataTable, cols ...string) (*DataTable, error) {
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
			resolved, _, ok := resolveEncodingColumn(t, column.name)
			if !ok {
				err = fmt.Errorf("SimpleImputer.Transform: fitted column %q not found", column.name)
				return
			}
			byIndex[resolved] = column
		}

		outColumns := make([]*DataList, 0, len(t.columns))
		for idx, source := range t.columns {
			column, selected := byIndex[idx]
			if !selected || column.passThrough {
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

// InverseTransform is unsupported because replacing a missing value loses the
// information needed to restore whether that cell was missing.
func (i *SimpleImputer) InverseTransform(_ *DataTable) (*DataTable, error) {
	return nil, errors.New("SimpleImputer.InverseTransform: imputation is not reversible")
}

func (i *SimpleImputer) validateConfiguration() error {
	switch i.strategy {
	case ImputeMean, ImputeMedian, ImputeMode:
		if i.constantArgs != 0 {
			return fmt.Errorf("SimpleImputer: strategy %q does not accept a constant value", i.strategy)
		}
	case ImputeConstant:
		if i.constantArgs != 1 {
			return errors.New("SimpleImputer: constant strategy requires exactly one constant value")
		}
	default:
		return fmt.Errorf("SimpleImputer: unsupported strategy %q", i.strategy)
	}
	return nil
}

func (i *SimpleImputer) deriveReplacement(name string, data []any) (any, bool, error) {
	if i.strategy == ImputeConstant {
		if !hasObservedValues(data) {
			return nil, false, fmt.Errorf("SimpleImputer.Fit: column %q has no observed values", name)
		}
		return i.constant, false, nil
	}

	observed := make([]any, 0, len(data))
	for _, value := range data {
		if !isMissing(value) {
			observed = append(observed, value)
		}
	}
	if len(observed) == 0 {
		return nil, false, fmt.Errorf("SimpleImputer.Fit: column %q has no observed values", name)
	}

	switch i.strategy {
	case ImputeMean, ImputeMedian:
		values := make([]float64, len(observed))
		for index, value := range observed {
			converted, ok := ToFloat64Safe(value)
			if !ok {
				return nil, true, nil
			}
			values[index] = converted
		}
		if i.strategy == ImputeMean {
			return meanOf(values), false, nil
		}
		sort.Float64s(values)
		middle := len(values) / 2
		if len(values)%2 == 1 {
			return values[middle], false, nil
		}
		return (values[middle-1] + values[middle]) / 2, false, nil
	case ImputeMode:
		return firstMode(observed), false, nil
	default:
		return nil, false, fmt.Errorf("SimpleImputer: unsupported strategy %q", i.strategy)
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

var _ Scaler = (*SimpleImputer)(nil)
