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

// isNilList reports whether dl carries no usable list: no value at all, a nil
// *insyra.DataList, or a wrapper such as isr.DL that embeds one. Each of them
// panics on its first method call, so the constructors refuse it instead.
func isNilList(dl insyra.IDataList) bool {
	return insyra.IsNilList(dl)
}

// isNilTable is isNilList for a table.
func isNilTable(dt insyra.IDataTable) bool {
	return insyra.IsNilTable(dt)
}

// nonNilLists returns data without its nil entries, each kept list's position
// among the caller's arguments, and the positions it dropped, the way the plot
// package drops them. It logs nothing: a call that goes on to fail reports
// only its error, and one that draws a chart warns through warnNilLists.
func nonNilLists(data []insyra.IDataList) (kept []insyra.IDataList, pos, dropped []int) {
	kept = make([]insyra.IDataList, 0, len(data))
	pos = make([]int, 0, len(data))
	for i, dl := range data {
		if isNilList(dl) {
			dropped = append(dropped, i)
			continue
		}
		kept = append(kept, dl)
		pos = append(pos, i)
	}
	return kept, pos, dropped
}

// warnNilLists warns once per nil list a chart was drawn without, naming its
// position counted from one.
func warnNilLists(funcName string, dropped []int) {
	for _, i := range dropped {
		insyra.LogWarning("gplot", funcName, "data list %d is nil; skipping it", i+1)
	}
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
