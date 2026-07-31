package ml

import (
	"fmt"
	"reflect"

	"github.com/HazelnutParadise/insyra"
)

type modelBase struct {
	features []string
}

func (m modelBase) Features() []string {
	return append([]string(nil), m.features...)
}

func fitFeatures(dt *insyra.DataTable) ([]string, []insyra.IDataList, error) {
	if err := requireTable(dt); err != nil {
		return nil, nil, err
	}
	features := dt.ColNames()
	lists := make([]insyra.IDataList, len(features))
	for i := range features {
		lists[i] = dt.GetColByNumber(i)
	}
	return features, lists, nil
}

func orderedFeatures(dt *insyra.DataTable, features []string) ([]insyra.IDataList, error) {
	ordered := make([]insyra.IDataList, len(features))
	for i, name := range features {
		col := dt.GetColByName(name)
		if col == nil {
			return nil, fmt.Errorf("ml: missing feature column %q", name)
		}
		ordered[i] = col
	}
	return ordered, nil
}

func orderedTable(dt *insyra.DataTable, features []string) (*insyra.DataTable, error) {
	if err := requireTable(dt); err != nil {
		return nil, err
	}
	cols, err := orderedFeatures(dt, features)
	if err != nil {
		return nil, err
	}
	clones := make([]*insyra.DataList, len(cols))
	for i, col := range cols {
		clones[i] = col.Clone().SetName(features[i])
	}
	return insyra.NewDataTable(clones...), nil
}

func requireTable(dt *insyra.DataTable) error {
	if dt == nil || isNilPointer(dt) {
		return fmt.Errorf("ml: data table is nil")
	}
	return nil
}

func isNilPointer(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.IsValid() && rv.Kind() == reflect.Ptr && rv.IsNil()
}

func oneLogisticOption(opts []LogisticOptions) (LogisticOptions, bool, error) {
	if len(opts) > 1 {
		return LogisticOptions{}, false, fmt.Errorf("ml: opts accepts at most one value")
	}
	if len(opts) == 0 {
		return LogisticOptions{}, false, nil
	}
	return opts[0], true, nil
}

func onePoissonOption(opts []PoissonOptions) (PoissonOptions, bool, error) {
	if len(opts) > 1 {
		return PoissonOptions{}, false, fmt.Errorf("ml: opts accepts at most one value")
	}
	if len(opts) == 0 {
		return PoissonOptions{}, false, nil
	}
	return opts[0], true, nil
}

func oneKMeansOption(opts []KMeansOptions) (KMeansOptions, bool, error) {
	if len(opts) > 1 {
		return KMeansOptions{}, false, fmt.Errorf("ml: opts accepts at most one value")
	}
	if len(opts) == 0 {
		return KMeansOptions{}, false, nil
	}
	return opts[0], true, nil
}

func oneKNNOption(opts []KNNOptions) (KNNOptions, bool, error) {
	if len(opts) > 1 {
		return KNNOptions{}, false, fmt.Errorf("ml: opts accepts at most one value")
	}
	if len(opts) == 0 {
		return KNNOptions{}, false, nil
	}
	return opts[0], true, nil
}

func asDataList(list insyra.IDataList) *insyra.DataList {
	if list == nil {
		return nil
	}
	return list.Clone()
}

func asDataTable(table insyra.IDataTable) *insyra.DataTable {
	if table == nil {
		return nil
	}
	return table.Clone()
}

func numericColumns(dt *insyra.DataTable) ([][]float64, error) {
	if err := requireTable(dt); err != nil {
		return nil, err
	}
	columns := make([][]float64, dt.NumCols())
	for j, name := range dt.ColNames() {
		raw := dt.GetColByNumber(j).Data()
		values := make([]float64, len(raw))
		for i, value := range raw {
			converted, ok := insyra.ToFloat64Safe(value)
			if !ok {
				return nil, fmt.Errorf("ml: column %q contains non-numeric value at row %d", name, i)
			}
			values[i] = converted
		}
		columns[j] = values
	}
	return columns, nil
}

func probabilityTable(classes *insyra.DataList, probabilities [][]float64) *insyra.DataTable {
	columns := make([]*insyra.DataList, len(probabilities))
	for i, values := range probabilities {
		data := make([]any, len(values))
		for j, value := range values {
			data[j] = value
		}
		name := fmt.Sprintf("class_%d", i+1)
		if classes != nil && i < classes.Len() {
			name = fmt.Sprint(classes.Get(i))
		}
		columns[i] = insyra.NewDataList(data...).SetName(name)
	}
	return insyra.NewDataTable(columns...)
}
