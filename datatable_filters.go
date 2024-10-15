package insyra

import (
	"strings"
	"time"
)

// ==================== Col Index ====================

// FilterByColIndexGreaterThan filters columns with index greater than the specified column.
func (dt *DataTable) FilterByColIndexGreaterThan(columnLetter string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx, exists := dt.columnIndex[columnLetter]
	if !exists {
		return &DataTable{}
	}

	filteredCols := dt.columns[colIdx+1:]

	newDt := &DataTable{
		columns:           filteredCols,
		columnIndex:       dt.columnIndex,
		rowNames:          dt.rowNames,
		creationTimestamp: dt.creationTimestamp,
	}

	newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())

	return newDt
}

// FilterByColIndexGreaterThanOrEqualTo filters columns with index greater than or equal to the specified column.
func (dt *DataTable) FilterByColIndexGreaterThanOrEqualTo(columnLetter string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx, exists := dt.columnIndex[columnLetter]
	if !exists {
		return &DataTable{}
	}

	filteredCols := dt.columns[colIdx:]

	newDt := &DataTable{
		columns:           filteredCols,
		columnIndex:       dt.columnIndex,
		rowNames:          dt.rowNames,
		creationTimestamp: dt.creationTimestamp,
	}

	newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())

	return newDt
}

// FilterByColIndexEqualTo filters to only keep the column with the specified index.
func (dt *DataTable) FilterByColIndexEqualTo(columnLetter string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx, exists := dt.columnIndex[columnLetter]
	if !exists {
		return &DataTable{}
	}

	filteredCols := []*DataList{dt.columns[colIdx]}

	newDt := &DataTable{
		columns:           filteredCols,
		columnIndex:       dt.columnIndex,
		rowNames:          dt.rowNames,
		creationTimestamp: dt.creationTimestamp,
	}

	newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())

	return newDt
}

// FilterByColIndexLessThan filters columns with index less than the specified column.
func (dt *DataTable) FilterByColIndexLessThan(columnLetter string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx, exists := dt.columnIndex[columnLetter]
	if !exists {
		return &DataTable{}
	}

	filteredCols := dt.columns[:colIdx]

	newDt := &DataTable{
		columns:           filteredCols,
		columnIndex:       dt.columnIndex,
		rowNames:          dt.rowNames,
		creationTimestamp: dt.creationTimestamp,
	}

	newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())

	return newDt
}

// FilterByColIndexLessThanOrEqualTo filters columns with index less than or equal to the specified column.
func (dt *DataTable) FilterByColIndexLessThanOrEqualTo(columnLetter string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx, exists := dt.columnIndex[columnLetter]
	if !exists {
		return &DataTable{}
	}

	filteredCols := dt.columns[:colIdx+1]

	newDt := &DataTable{
		columns:           filteredCols,
		columnIndex:       dt.columnIndex,
		rowNames:          dt.rowNames,
		creationTimestamp: dt.creationTimestamp,
	}

	newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())

	return newDt
}

// ==================== Col Name ====================

// FilterByColNameEqualTo filters to only keep the column with the specified name.
func (dt *DataTable) FilterByColNameEqualTo(columnName string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx := -1
	for i, col := range dt.columns {
		if col.name == columnName {
			colIdx = i
			break
		}
	}
	if colIdx == -1 {
		return &DataTable{}
	}

	filteredCols := []*DataList{dt.columns[colIdx]}

	newDt := &DataTable{
		columns:           filteredCols,
		columnIndex:       dt.columnIndex,
		rowNames:          dt.rowNames,
		creationTimestamp: dt.creationTimestamp,
	}

	newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())

	return newDt
}

// FilterByColNameContains filters columns whose name contains the specified substring.
func (dt *DataTable) FilterByColNameContains(substring string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

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

	return newDt
}

// ==================== Row Index ====================

// FilterByRowIndexGreaterThan filters rows with index greater than the specified threshold.
func (dt *DataTable) FilterByRowIndexGreaterThan(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value interface{}) bool {
		return rowIndex > threshold
	})
}

// FilterByRowIndexGreaterThanOrEqualTo filters rows with index greater than or equal to the specified threshold.
func (dt *DataTable) FilterByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value interface{}) bool {
		return rowIndex >= threshold
	})
}

// FilterByRowIndexEqualTo filters to only keep the row with the specified index.
func (dt *DataTable) FilterByRowIndexEqualTo(index int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value interface{}) bool {
		return rowIndex == index
	})
}

// FilterByRowIndexLessThan filters rows with index less than the specified threshold.
func (dt *DataTable) FilterByRowIndexLessThan(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value interface{}) bool {
		return rowIndex < threshold
	})
}

// FilterByRowIndexLessThanOrEqualTo filters rows with index less than or equal to the specified threshold.
func (dt *DataTable) FilterByRowIndexLessThanOrEqualTo(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value interface{}) bool {
		return rowIndex <= threshold
	})
}

// ==================== Row Name ====================

// FilterByRowNameEqualTo filters to only keep the row with the specified name.
func (dt *DataTable) FilterByRowNameEqualTo(rowName string) *DataTable {
	newdt := dt.FilterByRowNameContains(rowName)
	for name, index := range newdt.rowNames {
		if name == rowName {
			return newdt.FilterByRowIndexEqualTo(index)
		}
	}
	return &DataTable{}
}

// FilterByRowNameContains filters rows whose name contains the specified substring.
func (dt *DataTable) FilterByRowNameContains(substring string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// 找出符合條件的行索引
	var filteredRowIndices []int
	for name, rowIndex := range dt.rowNames {
		if strings.Contains(name, substring) {
			filteredRowIndices = append(filteredRowIndices, rowIndex)
		}
	}

	// 如果沒有符合條件的行，返回空的 DataTable
	if len(filteredRowIndices) == 0 {
		return &DataTable{}
	}

	// 構建新的 DataTable，只包含符合條件的行
	filteredCols := make([]*DataList, len(dt.columns))
	for i := range dt.columns {
		filteredCols[i] = &DataList{
			data:              make([]interface{}, 0, len(filteredRowIndices)),
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

	return newDt
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
func (dt *DataTable) FilterByCustomElement(filterFunc func(value interface{}) bool) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value interface{}) bool {
		return filterFunc(value)
	})
}

// ==================== Custom Filter ====================

// FilterFunc is a custom filter function that takes the row index, column name, and value as input and returns a boolean.
type FilterFunc func(rowIndex int, columnIndex string, value interface{}) bool

// Filter applies a custom filter function to the DataTable and returns the filtered DataTable.
func (dt *DataTable) Filter(filterFunc FilterFunc) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

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

	return newDt
}
