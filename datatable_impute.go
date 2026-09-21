package insyra

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

func (dt *DataTable) fillColumns(cols []any, methodName string, fill func(*DataList) bool) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		for _, idx := range dt.imputeColumnIndices(methodName, cols) {
			col := dt.columns[idx].Clone()
			if fill(col) {
				dt.columns[idx] = col
			}
		}
		dt.updateTimestamp()
	})
	return dt
}

// FillForward fills missing values in selected columns using previous observed values.
// limit caps how many consecutive missing cells each gap fills; limit <= 0 means unlimited.
func (dt *DataTable) FillForward(limit int, cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillForward", func(dl *DataList) bool {
		dl.FillForward(limit)
		return true
	})
}

// FillBackward fills missing values in selected columns using next observed values.
// limit caps how many consecutive missing cells each gap fills; limit <= 0 means unlimited.
func (dt *DataTable) FillBackward(limit int, cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillBackward", func(dl *DataList) bool {
		dl.FillBackward(limit)
		return true
	})
}

// FillWithMean fills missing values in numeric columns using the mean.
func (dt *DataTable) FillWithMean(cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillWithMean", func(dl *DataList) bool {
		if !hasOnlyNumericObservedValues(dl) {
			return false
		}
		dl.FillWithMean()
		return true
	})
}

// FillWithMedian fills missing values in numeric columns using the median.
func (dt *DataTable) FillWithMedian(cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillWithMedian", func(dl *DataList) bool {
		if !hasOnlyNumericObservedValues(dl) {
			return false
		}
		dl.FillWithMedian()
		return true
	})
}

// FillWithMode fills missing values in selected columns using the mode.
func (dt *DataTable) FillWithMode(cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillWithMode", func(dl *DataList) bool {
		dl.FillWithMode()
		return true
	})
}

// FillByInterpolation fills missing values in numeric columns by linear interpolation.
func (dt *DataTable) FillByInterpolation(cols ...any) *DataTable {
	return dt.fillColumns(cols, "FillByInterpolation", func(dl *DataList) bool {
		if !hasOnlyNumericObservedValues(dl) {
			return false
		}
		dl.FillByInterpolation()
		return true
	})
}
