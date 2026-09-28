package plot

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
)

// chartError is the error a constructor returns when it cannot build a chart.
// Its text starts with the package and function, as gplot.SaveChart's does.
func chartError(funcName, format string, args ...any) error {
	return fmt.Errorf("plot: "+funcName+": "+format, args...)
}

// Every chart here reads its data through IDataList.AtomicDo. A nil interface
// value panics on that first method call, and so does a typed nil
// *insyra.DataList, because AtomicDo dereferences the receiver to reach its
// actor. error-philosophy says the library does not panic by default, so a nil
// among real lists is dropped with a warning and the rest of the chart is
// drawn; a chart left with nothing to draw returns an error, the same as a
// call given no data at all.

// isNilList reports whether dl carries no usable list — either no value at all,
// or a nil *insyra.DataList inside the interface.
func isNilList(dl insyra.IDataList) bool {
	if dl == nil {
		return true
	}
	v, ok := dl.(*insyra.DataList)
	return ok && v == nil
}

// nonNilLists returns data without its nil entries. It warns once per
// dropped entry, but only when something is left: a call left with nothing
// fails, and its error is the whole report.
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
			insyra.LogWarning("plot", funcName, "data list %d is nil; skipping it", i)
		}
	}
	return kept
}
