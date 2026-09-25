package insyra

import "fmt"

func (dt *DataTable) imputeColumnIndices(methodName string, cols []any) []int {
	if len(cols) == 0 {
		indices := make([]int, len(dt.columns))
		for i := range dt.columns {
			indices[i] = i
		}
		return indices
	}
	indices := make([]int, 0, len(cols))
	for _, col := range cols {
		idx, ok := dt.resolveColSelector(methodName, col)
		if !ok {
			continue
		}
		indices = append(indices, idx)
	}
	return indices
}

func (dt *DataTable) fillColumns(cols []any, methodName string, fill func(*DataList) (bool, string)) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		named := len(cols) > 0
		for _, idx := range dt.imputeColumnIndices(methodName, cols) {
			col := dt.columns[idx].Clone()
			filled, reason := fill(col)
			if filled {
				dt.columns[idx] = col
				continue
			}
			// A column the caller named and that cannot be filled is a
			// failure; one swept up by filling every column is skipped,
			// because the call asked to fill what it can.
			if named {
				dt.fail(methodName, "column %s %s", fillColumnLabel(dt, idx), reason)
			}
		}
		dt.updateTimestamp()
	})
	return dt
}

func fillColumnLabel(dt *DataTable, idx int) string {
	if name := dt.columns[idx].name; name != "" {
		return name
	}
	if letters, ok := alphaColIndex(idx); ok {
		return letters
	}
	return fmt.Sprintf("%d", idx)
}

// numericFill runs fill on a column of numbers, and otherwise says why the
// column cannot be filled that way.
func numericFill(fill func(*DataList)) func(*DataList) (bool, string) {
	return func(dl *DataList) (bool, string) {
		switch t := dataTypeOf(dl.data); t {
		case DataTypeNumber:
			fill(dl)
			return true, ""
		case DataTypeEmpty:
			return false, "has no values to compute a fill value from"
		default:
			return false, fmt.Sprintf("holds %s values, and only a number column can be filled this way", t)
		}
	}
}

// FillForward fills missing values in selected columns using previous observed values.
// limit caps how many consecutive missing cells each gap fills; limit <= 0 means unlimited.
func (dt *DataTable) FillForward(limit int, cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillForward", func(dl *DataList) (bool, string) {
		dl.FillForward(limit)
		return true, ""
	})
}

// FillBackward fills missing values in selected columns using next observed values.
// limit caps how many consecutive missing cells each gap fills; limit <= 0 means unlimited.
func (dt *DataTable) FillBackward(limit int, cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillBackward", func(dl *DataList) (bool, string) {
		dl.FillBackward(limit)
		return true, ""
	})
}

// FillWithMean fills missing values in numeric columns using the mean.
func (dt *DataTable) FillWithMean(cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillWithMean", numericFill(func(dl *DataList) { dl.FillWithMean() }))
}

// FillWithMedian fills missing values in numeric columns using the median.
func (dt *DataTable) FillWithMedian(cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillWithMedian", numericFill(func(dl *DataList) { dl.FillWithMedian() }))
}

// FillWithMode fills missing values in selected columns using the mode.
func (dt *DataTable) FillWithMode(cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillWithMode", func(dl *DataList) (bool, string) {
		if dataTypeOf(dl.data) == DataTypeEmpty {
			return false, "has no values to compute a fill value from"
		}
		dl.FillWithMode()
		return true, ""
	})
}

// FillByInterpolation fills missing values in numeric columns by linear
// interpolation, and extrapolates past the first and last observed values when
// extrapolate is true, the same as DataList.FillByInterpolation. The setting
// comes before the columns, as the fill limit does in FillForward.
func (dt *DataTable) FillByInterpolation(extrapolate bool, cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillByInterpolation", numericFill(func(dl *DataList) { dl.FillByInterpolation(extrapolate) }))
}
