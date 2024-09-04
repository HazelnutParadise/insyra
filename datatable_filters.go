package insyra

import (
	"strings"
	"time"
)

// ==================== Column Index ====================

// FilterByColumnIndexGreaterThan filters columns with index greater than the specified column.
func (dt *DataTable) FilterByColumnIndexGreaterThan(columnLetter string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx, exists := dt.columnIndex[columnLetter]
	if !exists {
		return &DataTable{}
	}

	filteredColumns := dt.columns[colIdx+1:]

	return &DataTable{
		columns:               filteredColumns,
		columnIndex:           dt.columnIndex,
		rowNames:              dt.rowNames,
		creationTimestamp:     dt.creationTimestamp,
		lastModifiedTimestamp: dt.lastModifiedTimestamp,
	}
}

// FilterByColumnIndexGreaterThanOrEqualTo filters columns with index greater than or equal to the specified column.
func (dt *DataTable) FilterByColumnIndexGreaterThanOrEqualTo(columnLetter string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx, exists := dt.columnIndex[columnLetter]
	if !exists {
		return &DataTable{}
	}

	filteredColumns := dt.columns[colIdx:]

	return &DataTable{
		columns:               filteredColumns,
		columnIndex:           dt.columnIndex,
		rowNames:              dt.rowNames,
		creationTimestamp:     dt.creationTimestamp,
		lastModifiedTimestamp: dt.lastModifiedTimestamp,
	}
}

// FilterByColumnIndexEqualTo filters to only keep the column with the specified index.
func (dt *DataTable) FilterByColumnIndexEqualTo(columnLetter string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx, exists := dt.columnIndex[columnLetter]
	if !exists {
		return &DataTable{}
	}

	filteredColumns := []*DataList{dt.columns[colIdx]}

	return &DataTable{
		columns:               filteredColumns,
		columnIndex:           dt.columnIndex,
		rowNames:              dt.rowNames,
		creationTimestamp:     dt.creationTimestamp,
		lastModifiedTimestamp: dt.lastModifiedTimestamp,
	}
}

// FilterByColumnIndexLessThan filters columns with index less than the specified column.
func (dt *DataTable) FilterByColumnIndexLessThan(columnLetter string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx, exists := dt.columnIndex[columnLetter]
	if !exists {
		return &DataTable{}
	}

	filteredColumns := dt.columns[:colIdx]

	return &DataTable{
		columns:               filteredColumns,
		columnIndex:           dt.columnIndex,
		rowNames:              dt.rowNames,
		creationTimestamp:     dt.creationTimestamp,
		lastModifiedTimestamp: dt.lastModifiedTimestamp,
	}
}

// FilterByColumnIndexLessThanOrEqualTo filters columns with index less than or equal to the specified column.
func (dt *DataTable) FilterByColumnIndexLessThanOrEqualTo(columnLetter string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	colIdx, exists := dt.columnIndex[columnLetter]
	if !exists {
		return &DataTable{}
	}

	filteredColumns := dt.columns[:colIdx+1]

	return &DataTable{
		columns:               filteredColumns,
		columnIndex:           dt.columnIndex,
		rowNames:              dt.rowNames,
		creationTimestamp:     dt.creationTimestamp,
		lastModifiedTimestamp: dt.lastModifiedTimestamp,
	}
}

// ==================== Column Name ====================

// FilterByColumnNameEqualTo filters to only keep the column with the specified name.
func (dt *DataTable) FilterByColumnNameEqualTo(columnName string) *DataTable {
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

	filteredColumns := []*DataList{dt.columns[colIdx]}

	return &DataTable{
		columns:               filteredColumns,
		columnIndex:           dt.columnIndex,
		rowNames:              dt.rowNames,
		creationTimestamp:     dt.creationTimestamp,
		lastModifiedTimestamp: dt.lastModifiedTimestamp,
	}
}

// FilterByColumnNameContains filters columns whose name contains the specified substring.
func (dt *DataTable) FilterByColumnNameContains(substring string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	var filteredColumns []*DataList
	for _, col := range dt.columns {
		if strings.Contains(col.name, substring) {
			filteredColumns = append(filteredColumns, col)
		}
	}

	return &DataTable{
		columns:               filteredColumns,
		columnIndex:           dt.columnIndex,
		rowNames:              dt.rowNames,
		creationTimestamp:     dt.creationTimestamp,
		lastModifiedTimestamp: dt.lastModifiedTimestamp,
	}
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
	filteredColumns := make([]*DataList, len(dt.columns))
	for i := range dt.columns {
		filteredColumns[i] = &DataList{
			data:                  make([]interface{}, 0, len(filteredRowIndices)),
			name:                  dt.columns[i].name,
			creationTimestamp:     dt.columns[i].creationTimestamp,
			lastModifiedTimestamp: dt.columns[i].lastModifiedTimestamp,
		}
		for _, rowIndex := range filteredRowIndices {
			if rowIndex < len(dt.columns[i].data) {
				filteredColumns[i].data = append(filteredColumns[i].data, dt.columns[i].data[rowIndex])
			}
		}
	}

	return &DataTable{
		columns:               filteredColumns,
		columnIndex:           dt.columnIndex,
		rowNames:              filterRowNames(dt.rowNames, filteredRowIndices),
		creationTimestamp:     dt.creationTimestamp,
		lastModifiedTimestamp: time.Now().Unix(),
	}
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

	filteredColumns := make([]*DataList, len(dt.columns))
	for i := range dt.columns {
		filteredColumns[i] = &DataList{}
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
				filteredColumns[colIdx].data = append(filteredColumns[colIdx].data, value)
			} else {
				filteredColumns[colIdx].data = append(filteredColumns[colIdx].data, nil)
			}
		}
		if !keepRow {
			for _, col := range filteredColumns {
				col.data = col.data[:len(col.data)-1]
			}
		}
	}

	return &DataTable{
		columns:               filteredColumns,
		columnIndex:           dt.columnIndex,
		rowNames:              dt.rowNames,
		creationTimestamp:     dt.creationTimestamp,
		lastModifiedTimestamp: dt.lastModifiedTimestamp,
	}
}
