package ta

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

func numericSeries(dl insyra.IDataList, label string) ([]float64, error) {
	out, err := numericSeriesAllowNaN(dl, label)
	if err != nil {
		return nil, err
	}
	for i, v := range out {
		if math.IsNaN(v) {
			return nil, fmt.Errorf("%s contains a non-finite value at row index %d: %v", label, i, v)
		}
	}
	return out, nil
}
func numericSeriesAllowNaN(dl insyra.IDataList, label string) ([]float64, error) {
	if insyra.IsNilList(dl) {
		return nil, fmt.Errorf("%s is nil", label)
	}
	var raw []any
	dl.AtomicDo(func(l *insyra.DataList) {
		raw = l.Data()
	})
	out := make([]float64, len(raw))
	for i, value := range raw {
		converted, ok := insyra.ToFloat64Safe(value)
		if !ok {
			return nil, fmt.Errorf("%s contains a non-numeric value at row index %d: %v", label, i, value)
		}
		if math.IsInf(converted, 0) {
			return nil, fmt.Errorf("%s contains a non-finite value at row index %d: %v", label, i, converted)
		}
		out[i] = converted
	}
	return out, nil
}
