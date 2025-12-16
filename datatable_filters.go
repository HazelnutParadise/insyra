package insyra

import (
	"strings"
	"time"
)

// ==================== Col Index ====================

// FilterColsByColIndexGreaterThan filters columns with index greater than the specified column.
func (dt *DataTable) FilterColsByColIndexGreaterThan(columnIndexLetter string) *DataTable {
	var newDt *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		columnIndexLetter = strings.ToUpper(columnIndexLetter)
		colIdx, exists := dt.columnIndex[columnIndexLetter]
		if !exists {
			newDt = &DataTable{}
			return
		}

		filteredCols := dt.columns[colIdx+1:]

		newDt = &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          dt.rowNames,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
	})
	return newDt
}

// FilterColsByColIndexGreaterThanOrEqualTo filters columns with index greater than or equal to the specified column.
func (dt *DataTable) FilterColsByColIndexGreaterThanOrEqualTo(columnIndexLetter string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		columnIndexLetter = strings.ToUpper(columnIndexLetter)
		colIdx, exists := dt.columnIndex[columnIndexLetter]
		if !exists {
			result = &DataTable{}
			return
		}

		filteredCols := dt.columns[colIdx:]

		newDt := &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          dt.rowNames,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}

// FilterColsByColIndexEqualTo filters to only keep the column with the specified index.
func (dt *DataTable) FilterColsByColIndexEqualTo(columnIndexLetter string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		columnIndexLetter = strings.ToUpper(columnIndexLetter)
		colIdx, exists := dt.columnIndex[columnIndexLetter]
		if !exists {
			result = &DataTable{}
			return
		}

		filteredCols := []*DataList{dt.columns[colIdx]}

		newDt := &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          dt.rowNames,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}

// FilterColsByColIndexLessThan filters columns with index less than the specified column.
func (dt *DataTable) FilterColsByColIndexLessThan(columnIndexLetter string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		columnIndexLetter = strings.ToUpper(columnIndexLetter)
		colIdx, exists := dt.columnIndex[columnIndexLetter]
		if !exists {
			result = &DataTable{}
			return
		}

		filteredCols := dt.columns[:colIdx]

		newDt := &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          dt.rowNames,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}

// FilterColsByColIndexLessThanOrEqualTo filters columns with index less than or equal to the specified column.
func (dt *DataTable) FilterColsByColIndexLessThanOrEqualTo(columnIndexLetter string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		columnIndexLetter = strings.ToUpper(columnIndexLetter)
		colIdx, exists := dt.columnIndex[columnIndexLetter]
		if !exists {
			result = &DataTable{}
			return
		}

		filteredCols := dt.columns[:colIdx+1]

		newDt := &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          dt.rowNames,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}

// ==================== Col Name ====================

// FilterColsByColNameEqualTo filters to only keep the column with the specified name.
func (dt *DataTable) FilterColsByColNameEqualTo(columnName string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		colIdx := -1
		for i, col := range dt.columns {
			if col.name == columnName {
				colIdx = i
				break
			}
		}
		if colIdx == -1 {
			result = &DataTable{}
			return
		}

		filteredCols := []*DataList{dt.columns[colIdx]}

		newDt := &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          dt.rowNames,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}

// FilterColsByColNameContains filters columns whose name contains the specified substring.
func (dt *DataTable) FilterColsByColNameContains(substring string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		var filteredCols []*DataList
		for _, col := range dt.columns {
			if strings.Contains(col.name, substring) {
				filteredCols = append(filteredCols, col)
			}
		}

		newDt := &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          dt.rowNames,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}

// ==================== Row Index ====================

// FilterRowsByRowIndexGreaterThan filters rows with index greater than the specified threshold.
func (dt *DataTable) FilterRowsByRowIndexGreaterThan(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return rowIndex > threshold
	})
}

// FilterRowsByRowIndexGreaterThanOrEqualTo filters rows with index greater than or equal to the specified threshold.
func (dt *DataTable) FilterRowsByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return rowIndex >= threshold
	})
}

// FilterRowsByRowIndexEqualTo filters to only keep the row with the specified index.
func (dt *DataTable) FilterRowsByRowIndexEqualTo(index int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return rowIndex == index
	})
}

// FilterRowsByRowIndexLessThan filters rows with index less than the specified threshold.
func (dt *DataTable) FilterRowsByRowIndexLessThan(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return rowIndex < threshold
	})
}

// FilterRowsByRowIndexLessThanOrEqualTo filters rows with index less than or equal to the specified threshold.
func (dt *DataTable) FilterRowsByRowIndexLessThanOrEqualTo(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return rowIndex <= threshold
	})
}

// ==================== Row Name ====================

// FilterRowsByRowNameEqualTo filters to only keep the row with the specified name.
func (dt *DataTable) FilterRowsByRowNameEqualTo(rowName string) *DataTable {
	newdt := dt.FilterRowsByRowNameContains(rowName)
	for name, index := range newdt.rowNames {
		if name == rowName {
			return newdt.FilterRowsByRowIndexEqualTo(index)
		}
	}
	return &DataTable{}
}

// FilterRowsByRowNameContains filters rows whose name contains the specified substring.
func (dt *DataTable) FilterRowsByRowNameContains(substring string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		// 找出符合條件的行索引
		var filteredRowIndices []int
		for name, rowIndex := range dt.rowNames {
			if strings.Contains(name, substring) {
				filteredRowIndices = append(filteredRowIndices, rowIndex)
			}
		}

		// 如果沒有符合條件的行，返回空的 DataTable
		if len(filteredRowIndices) == 0 {
			result = &DataTable{}
			return
		}

		// 構建新的 DataTable，只包含符合條件的行
		filteredCols := make([]*DataList, len(dt.columns))
		for i := range dt.columns {
			filteredCols[i] = &DataList{
				data:              make([]any, 0, len(filteredRowIndices)),
				name:              dt.columns[i].name,
				creationTimestamp: dt.columns[i].creationTimestamp,
			}

			filteredCols[i].lastModifiedTimestamp.Store(
				dt.columns[i].lastModifiedTimestamp.Load())
			for _, rowIndex := range filteredRowIndices {
				if rowIndex < len(dt.columns[i].data) {
					filteredCols[i].data = append(filteredCols[i].data, dt.columns[i].data[rowIndex])
				}
			}
		}

		newDt := &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          filterRowNames(dt.rowNames, filteredRowIndices),
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(time.Now().Unix())
		result = newDt
	})
	return result
}

// filterRowNames creates a new map of row names with updated indices after filtering.
func filterRowNames(originalRowNames map[string]int, filteredIndices []int) map[string]int {
	newRowNames := make(map[string]int)
	for name, originalIndex := range originalRowNames {
		for newIndex, filteredIndex := range filteredIndices {
			if originalIndex == filteredIndex {
				newRowNames[name] = newIndex
			}
		}
	}
	return newRowNames
}

// ==================== Custom Element ====================

// FilterByCustomElement filters the table based on a custom function applied to each element.
func (dt *DataTable) FilterByCustomElement(filterFunc func(value any) bool) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return filterFunc(value)
	})
}

// ==================== Custom Filter ====================

// Filter applies a custom filter function to the DataTable and returns the filtered DataTable.
func (dt *DataTable) Filter(filterFunc func(rowIndex int, columnIndex string, value any) bool) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		filteredCols := make([]*DataList, len(dt.columns))
		for i := range dt.columns {
			filteredCols[i] = &DataList{}
		}

		for rowIdx := range dt.columns[0].data {
			keepRow := false
			for colIdx, col := range dt.columns {
				value := col.data[rowIdx]
				colName := ""
				for name, idx := range dt.columnIndex {
					if idx == colIdx {
						colName = name
						break
					}
				}
				if filterFunc(rowIdx, colName, value) {
					keepRow = true
					filteredCols[colIdx].data = append(filteredCols[colIdx].data, value)
				} else {
					filteredCols[colIdx].data = append(filteredCols[colIdx].data, nil)
				}
			}
			if !keepRow {
				for _, col := range filteredCols {
					col.data = col.data[:len(col.data)-1]
				}
			}
		}

		newDt := &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          dt.rowNames,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}

// ==================== Filter Cols ====================

// FilterCols applies a custom filter function to each cell in every column and returns a
// new DataTable that only contains columns where the filter function returns true for at least
// one cell in that column.
//
// The filter function receives:
// - rowIndex: index of the row
// - rowName: name of the row (empty if none)
// - x: the cell value
func (dt *DataTable) FilterCols(filterFunc func(rowIndex int, rowName string, x any) bool) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		if len(dt.columns) == 0 {
			result = &DataTable{}
			return
		}

		numRows := 0
		if len(dt.columns) > 0 {
			numRows = len(dt.columns[0].data)
		}

		filteredCols := make([]*DataList, 0)

		for _, col := range dt.columns {
			keep := false
			for rowIdx := 0; rowIdx < numRows; rowIdx++ {
				var x any
				if rowIdx < len(col.data) {
					x = col.data[rowIdx]
				} else {
					x = nil
				}
				rowName, _ := dt.getRowNameByIndex(rowIdx)
				if filterFunc(rowIdx, rowName, x) {
					keep = true
					break
				}
			}
			if keep {
				newCol := &DataList{
					data:              make([]any, len(col.data)),
					name:              col.name,
					creationTimestamp: col.creationTimestamp,
				}
				copy(newCol.data, col.data)
				newCol.lastModifiedTimestamp.Store(col.lastModifiedTimestamp.Load())
				filteredCols = append(filteredCols, newCol)
			}
		}

		if len(filteredCols) == 0 {
			result = &DataTable{}
			return
		}

		newDt := &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          dt.rowNames,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}

// ==================== Filter Rows ====================

// FilterRows applies a custom filter function to each cell in the DataTable and keeps only rows
// where the filter function returns true for at least one cell.
// The filter function receives: colindex (column letter), colname (column name), and x (cell value).
func (dt *DataTable) FilterRows(filterFunc func(colIndex, colName string, x any) bool) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		filteredCols := make([]*DataList, len(dt.columns))
		for i := range dt.columns {
			filteredCols[i] = NewDataList()
		}

		// Build reverse index: column index to letter
		colIndexToLetter := make(map[int]string)
		for letter, idx := range dt.columnIndex {
			colIndexToLetter[idx] = letter
		}

		numRows := 0
		if len(dt.columns) > 0 {
			numRows = len(dt.columns[0].data)
		}

		for rowIdx := 0; rowIdx < numRows; rowIdx++ {
			keepRow := false
			rowData := make([]any, len(dt.columns))

			for colIdx, col := range dt.columns {
				value := col.data[rowIdx]
				colLetter := colIndexToLetter[colIdx]
				colName := col.name

				rowData[colIdx] = value

				// Apply filter function
				if filterFunc(colLetter, colName, value) {
					keepRow = true
				}
			}

			// Add the row if it passed the filter
			if keepRow {
				for colIdx, value := range rowData {
					filteredCols[colIdx].data = append(filteredCols[colIdx].data, value)
					filteredCols[colIdx].name = dt.columns[colIdx].name
				}
			}
		}

		newDt := &DataTable{
			columns:           filteredCols,
			columnIndex:       dt.columnIndex,
			rowNames:          dt.rowNames,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}
