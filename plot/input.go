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

// isNilList reports whether dl carries no usable list: no value at all, a nil
// *insyra.DataList, or a wrapper such as isr.DL that embeds one.
func isNilList(dl insyra.IDataList) bool {
	return insyra.IsNilList(dl)
}

// nonNilLists returns data without its nil entries, each kept list's position
// among the caller's arguments, and the positions it dropped. An error about
// a list names the argument the caller wrote. It logs nothing: a call that
// goes on to fail reports only its error, and one that draws a chart warns
// about the dropped lists through warnNilLists.
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
		insyra.LogWarning("plot", funcName, "data list %d is nil; skipping it", i+1)
	}
}

// requireNonEmptyLists returns an error naming the first list with no values.
// It reads only lengths, so a call that fails here logs nothing.
func requireNonEmptyLists(funcName string, lists []insyra.IDataList, pos []int) error {
	for i, dl := range lists {
		if dl.Len() == 0 {
			return chartError(funcName, "list %d is empty", pos[i]+1)
		}
	}
	return nil
}
