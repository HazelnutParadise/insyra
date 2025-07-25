package insyra

import "strings"

// ================ Swap Columns ================

// SwapColsByName swaps two columns by their names.
func (dt *DataTable) SwapColsByName(columnName1 string, columnName2 string) *DataTable {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	var index1, index2 = -1, -1

	for i, col := range dt.columns {
		if col.name == columnName1 {
			index1 = i
		}
		if col.name == columnName2 {
			index2 = i
		}
		if index1 != -1 && index2 != -1 {
			break
		}
	}

	if index1 == -1 {
		LogWarning("DataTable", "SwapColsByName", "Column '%s' not found", columnName1)
		return dt
	}
	if index2 == -1 {
		LogWarning("DataTable", "SwapColsByName", "Column '%s' not found", columnName2)
		return dt
	}

	return dt.swapColsByNumber_NoLock(index1, index2)
}

// SwapColsByIndex swaps two columns by their letter indices.
func (dt *DataTable) SwapColsByIndex(columnIndex1 string, columnIndex2 string) *DataTable {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	idx1, ok1 := dt.columnIndex[strings.ToUpper(columnIndex1)]
	if !ok1 {
		LogWarning("DataTable", "SwapColsByIndex", "Column index '%s' not found", columnIndex1)
		return dt
	}
	idx2, ok2 := dt.columnIndex[strings.ToUpper(columnIndex2)]
	if !ok2 {
		LogWarning("DataTable", "SwapColsByIndex", "Column index '%s' not found", columnIndex2)
		return dt
	}

	return dt.swapColsByNumber_NoLock(idx1, idx2)
}

// SwapColsByNumber swaps two columns by their numerical indices.
func (dt *DataTable) SwapColsByNumber(columnNumber1 int, columnNumber2 int) *DataTable {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	if columnNumber1 < 0 {
		columnNumber1 += len(dt.columns) // Wrap around if negative
	}
	if columnNumber2 < 0 {
		columnNumber2 += len(dt.columns) // Wrap around if negative
	}

	if columnNumber1 < 0 || columnNumber1 >= len(dt.columns) {
		LogWarning("DataTable", "SwapColsByNumber", "Column number %d is out of range", columnNumber1)
		return dt
	}
	if columnNumber2 < 0 || columnNumber2 >= len(dt.columns) {
		LogWarning("DataTable", "SwapColsByNumber", "Column number %d is out of range", columnNumber2)
		return dt
	}

	return dt.swapColsByNumber_NoLock(columnNumber1, columnNumber2)
}

func (dt *DataTable) swapColsByNumber_NoLock(columnNumber1 int, columnNumber2 int) *DataTable {
	if columnNumber1 == columnNumber2 {
		return dt // No need to swap if they are the same
	}

	// Swap columns
	dt.columns[columnNumber1], dt.columns[columnNumber2] = dt.columns[columnNumber2], dt.columns[columnNumber1]

	// Update columnIndex map
	// We need to find the letter indices corresponding to the original numerical indices
	// and then update them to point to the new numerical indices.
	// A simpler way is to regenerate the column index.
	dt.regenerateColIndex()

	return dt
}

// ================ Swap rows ================

// SwapRowsByIndex swaps two rows by their numerical indices.
func (dt *DataTable) SwapRowsByIndex(rowIndex1 int, rowIndex2 int) *DataTable {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	maxColLen := dt.getMaxColLength()

	if rowIndex1 < 0 {
		rowIndex1 += maxColLen
	}
	if rowIndex2 < 0 {
		rowIndex2 += maxColLen
	}

	if rowIndex1 < 0 || rowIndex1 >= maxColLen {
		LogWarning("DataTable", "SwapRowsByIndex", "Row index %d is out of range", rowIndex1)
		return dt
	}
	if rowIndex2 < 0 || rowIndex2 >= maxColLen {
		LogWarning("DataTable", "SwapRowsByIndex", "Row index %d is out of range", rowIndex2)
		return dt
	}

	// Swap rows
	dt.swapRowsByIndex_NoLock(rowIndex1, rowIndex2)

	return dt
}

// SwapRowsByName swaps two rows by their names.
func (dt *DataTable) SwapRowsByName(rowName1 string, rowName2 string) *DataTable {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	index1, ok1 := dt.rowNames[rowName1]
	if !ok1 {
		LogWarning("DataTable", "SwapRowsByName", "Row name '%s' not found", rowName1)
		return dt
	}
	index2, ok2 := dt.rowNames[rowName2]
	if !ok2 {
		LogWarning("DataTable", "SwapRowsByName", "Row name '%s' not found", rowName2)
		return dt
	}

	return dt.swapRowsByIndex_NoLock(index1, index2)
}

func (dt *DataTable) swapRowsByIndex_NoLock(rowIndex1 int, rowIndex2 int) *DataTable {
	for _, col := range dt.columns {
		col.data[rowIndex1], col.data[rowIndex2] = col.data[rowIndex2], col.data[rowIndex1]
	}
	newRowName1, _ := dt.getRowNameByIndex(rowIndex2)
	newRowName2, _ := dt.getRowNameByIndex(rowIndex1)
	dt.rowNames[newRowName1] = rowIndex1
	dt.rowNames[newRowName2] = rowIndex2
	return dt
}
