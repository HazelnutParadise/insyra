package plot

import "github.com/HazelnutParadise/insyra"

// Every chart here reads its data through IDataList.AtomicDo. A nil interface
// value panics on that first method call, and so does a typed nil
// *insyra.DataList, because AtomicDo dereferences the receiver to reach its
// actor. error-philosophy says the library does not panic by default, so a nil
// among real lists is dropped with a warning and the rest of the chart is
// drawn; a chart left with nothing to draw refuses the same way it already
// refuses no data at all.

// isNilList reports whether dl carries no usable list — either no value at all,
// or a nil *insyra.DataList inside the interface.
func isNilList(dl insyra.IDataList) bool {
	if dl == nil {
		return true
	}
	v, ok := dl.(*insyra.DataList)
	return ok && v == nil
}

// nonNilLists returns data without its nil entries, warning once per entry.
func nonNilLists(funcName string, data []insyra.IDataList) []insyra.IDataList {
	kept := make([]insyra.IDataList, 0, len(data))
	for i, dl := range data {
		if isNilList(dl) {
			insyra.LogWarning("plot", funcName, "data list %d is nil; skipping it", i)
			continue
		}
		kept = append(kept, dl)
	}
	return kept
}
