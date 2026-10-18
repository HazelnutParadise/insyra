package plot

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// go-echarts writes a chart's options with encoding/json, which refuses NaN and
// an infinity, and drops the error: one such value leaves a page with no chart
// and nothing to say why. A constructor therefore refuses them up front and
// names the value, so the caller sees the problem instead of a blank page.

// requireFinite returns an error naming what when v is NaN or infinite.
func requireFinite(funcName, what string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return chartError(funcName, "%s is %v; a chart cannot draw NaN or an infinity", what, v)
	}
	return nil
}

// requireFiniteLists checks every numeric cell of every list. A cell that is
// not a number is left to the chart's existing handling.
func requireFiniteLists(funcName string, lists []insyra.IDataList) error {
	for i, dl := range lists {
		for row, cell := range dl.Data() {
			if cell == nil {
				continue
			}
			f, ok := insyra.ToFloat64Safe(cell)
			if !ok {
				continue
			}
			if err := requireFinite(funcName, sprintPos("list", i, row), f); err != nil {
				return err
			}
		}
	}
	return nil
}

// sprintPos names a cell the way the rest of the library does: the list by its
// place among the arguments, counting from one, and the row by the index
// Get(i) takes, counting from zero.
func sprintPos(kind string, i, row int) string {
	return fmt.Sprintf("%s %d, row index %d", kind, i+1, row)
}

// hasNumericCell reports whether dl holds at least one number.
func hasNumericCell(dl insyra.IDataList) bool {
	for _, cell := range dl.Data() {
		if cell == nil {
			continue
		}
		if _, ok := insyra.ToFloat64Safe(cell); ok {
			return true
		}
	}
	return false
}
