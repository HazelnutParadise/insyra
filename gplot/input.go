package gplot

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// chartError is the error a constructor returns when it cannot build a chart.
// Its text starts with the package and function, as SaveChart's does.
func chartError(funcName, format string, args ...any) error {
	return fmt.Errorf("gplot: "+funcName+": "+format, args...)
}

// A nil interface panics on its first method call, and so does a typed nil
// *insyra.DataList or *insyra.DataTable, because AtomicDo dereferences the
// receiver. These report either so the constructors can refuse it instead.

func isNilList(dl insyra.IDataList) bool {
	if dl == nil {
		return true
	}
	v, ok := dl.(*insyra.DataList)
	return ok && v == nil
}

func isNilTable(dt insyra.IDataTable) bool {
	if dt == nil {
		return true
	}
	v, ok := dt.(*insyra.DataTable)
	return ok && v == nil
}

// nonNilLists returns data without its nil entries, the way the plot package
// drops them. It warns once per dropped entry, but only when something is
// left: a call left with nothing fails, and its error is the whole report.
func nonNilLists(funcName string, data []insyra.IDataList) []insyra.IDataList {
	kept := make([]insyra.IDataList, 0, len(data))
	var dropped []int
	for i, dl := range data {
		if isNilList(dl) {
			dropped = append(dropped, i)
			continue
		}
		kept = append(kept, dl)
	}
	if len(kept) > 0 {
		for _, i := range dropped {
			insyra.LogWarning("gplot", funcName, "data list %d is nil; skipping it", i)
		}
	}
	return kept
}

// readValues reads dl through ToF64Slice. An empty list gives nil without the
// warning ToF64Slice logs for one, so a call that fails on it logs nothing.
func readValues(dl insyra.IDataList) []float64 {
	if dl.Len() == 0 {
		return nil
	}
	return dl.ToF64Slice()
}

// firstNonFinite returns the index of the first NaN or infinity in vs, or -1.
func firstNonFinite(vs []float64) int {
	for i, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return i
		}
	}
	return -1
}
