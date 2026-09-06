package insyra

import (
	"strings"

	"github.com/HazelnutParadise/insyra/internal/core"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

// ==================== Col Index ====================

// FilterColsByColIndexGreaterThan filters columns with index greater than the specified column.
func (dt *DataTable) FilterColsByColIndexGreaterThan(columnIndexLetter string) *DataTable {
	var newDt *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		columnIndexLetter = strings.ToUpper(columnIndexLetter)
		colIdx, ok := utils.ParseColIndex(columnIndexLetter)
		if !ok || colIdx < 0 || colIdx >= len(dt.columns)-1 {
			newDt = NewDataTable()
			return
		}

		filteredCols := dt.columns[colIdx+1:]

		newDt = &DataTable{
			columns:           cloneColumns(filteredCols),
			rowNames:          cloneRowNames(dt.rowNames),
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
		colIdx, ok := utils.ParseColIndex(columnIndexLetter)
		if !ok || colIdx < 0 || colIdx >= len(dt.columns) {
			result = NewDataTable()
			return
		}

		filteredCols := dt.columns[colIdx:]

		newDt := &DataTable{
			columns:           cloneColumns(filteredCols),
			rowNames:          cloneRowNames(dt.rowNames),
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
		colIdx, ok := utils.ParseColIndex(columnIndexLetter)
		if !ok || colIdx < 0 || colIdx >= len(dt.columns) {
			result = NewDataTable()
			return
		}

		filteredCols := []*DataList{dt.columns[colIdx]}

		newDt := &DataTable{
			columns:           cloneColumns(filteredCols),
			rowNames:          cloneRowNames(dt.rowNames),
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
		colIdx, ok := utils.ParseColIndex(columnIndexLetter)
		if !ok || colIdx <= 0 {
			result = NewDataTable()
			return
		}

		filteredCols := dt.columns[:colIdx]

		newDt := &DataTable{
			columns:           cloneColumns(filteredCols),
			rowNames:          cloneRowNames(dt.rowNames),
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
		colIdx, ok := utils.ParseColIndex(columnIndexLetter)
		if !ok || colIdx < 0 {
			result = NewDataTable()
			return
		}

		filteredCols := dt.columns[:colIdx+1]

		newDt := &DataTable{
			columns:           cloneColumns(filteredCols),
			rowNames:          cloneRowNames(dt.rowNames),
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
			result = NewDataTable()
			return
		}

		filteredCols := []*DataList{dt.columns[colIdx]}

		newDt := &DataTable{
			columns:           cloneColumns(filteredCols),
			rowNames:          cloneRowNames(dt.rowNames),
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
			columns:           cloneColumns(filteredCols),
			rowNames:          cloneRowNames(dt.rowNames),
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
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		if dt.rowNames == nil {
			result = NewDataTable()
			return
		}
		id, ok := dt.rowNames.Index(rowName)
		if !ok || id < 0 || id >= dt.getMaxColLength() {
			result = NewDataTable()
			return
		}
		result = dt.FilterRowsByRowIndexEqualTo(id)
	})
	return result
}

// FilterRowsByRowNameContains filters rows whose name contains the specified substring.
func (dt *DataTable) FilterRowsByRowNameContains(substring string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		maxRows := dt.getMaxColLength()
		// 找出符合條件的行索引
		var filteredRowIndices []int
		for i := 0; i < maxRows; i++ {
			name, ok := dt.getRowNameByIndex(i)
			if ok && name != "" && strings.Contains(name, substring) {
				filteredRowIndices = append(filteredRowIndices, i)
			}
		}

		// 如果沒有符合條件的行，返回空的 DataTable
		if len(filteredRowIndices) == 0 {
			result = NewDataTable()
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
				filteredCols[i].data = append(filteredCols[i].data, dt.columns[i].data[rowIndex])
			}
		}

		newDt := &DataTable{
			columns:           filteredCols,
			rowNames:          filterRowNames(dt.rowNames, filteredRowIndices),
			name:              dt.name,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}

// filterRowNames remaps row names to match filtered row indices.
func filterRowNames(originalRowNames *core.BiIndex, filteredIndices []int) *core.BiIndex {
	if originalRowNames == nil {
		return nil
	}
	if originalRowNames.Len() == 0 {
		return core.NewBiIndex(0)
	}
	indexMap := make(map[int]int, len(filteredIndices))
	for newIndex, filteredIndex := range filteredIndices {
		indexMap[filteredIndex] = newIndex
	}
	remapped := core.NewBiIndex(originalRowNames.Len())
	for _, id := range originalRowNames.IDs() {
		if target, exists := indexMap[id]; exists {
			name, ok := originalRowNames.Get(id)
			if !ok || name == "" {
				continue
			}
			_, _ = remapped.Set(target, name)
		}
	}
	return remapped
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
			// Preserve the original column name (and metadata) in the result;
			// a bare &DataList{} would silently drop every column name.
			filteredCols[i] = &DataList{name: dt.columns[i].name}
		}

		var filteredRowIndices []int
		// Guard against an empty table (no columns): dt.columns[0] would panic.
		if len(dt.columns) > 0 {
			cellAt := func(col *DataList, rowIdx int) any {
				if rowIdx < len(col.data) {
					return col.data[rowIdx]
				}
				return nil
			}
			for rowIdx := range dt.columns[0].data {
				// A row is kept if the predicate matches ANY cell in it.
				keepRow := false
				for colIdx, col := range dt.columns {
					colName, _ := utils.CalcColIndex(colIdx)
					if filterFunc(rowIdx, colName, cellAt(col, rowIdx)) {
						keepRow = true
						break
					}
				}
				if !keepRow {
					continue
				}
				// Preserve every cell's ORIGINAL value in a kept row; do not mask
				// non-matching cells to nil (that silently dropped data).
				for colIdx, col := range dt.columns {
					filteredCols[colIdx].data = append(filteredCols[colIdx].data, cellAt(col, rowIdx))
				}
				filteredRowIndices = append(filteredRowIndices, rowIdx)
			}
		}

		newDt := &DataTable{
			columns:           filteredCols,
			rowNames:          filterRowNames(dt.rowNames, filteredRowIndices),
			name:              dt.name,
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
			result = NewDataTable()
			return
		}

		numRows := dt.getMaxColLength()

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
			result = NewDataTable()
			return
		}

		newDt := &DataTable{
			columns:           cloneColumns(filteredCols),
			rowNames:          cloneRowNames(dt.rowNames),
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

		numRows := dt.getMaxColLength()

		var filteredRowIndices []int
		for rowIdx := 0; rowIdx < numRows; rowIdx++ {
			keepRow := false
			rowData := make([]any, len(dt.columns))

			for colIdx, col := range dt.columns {
				var value any
				if rowIdx < len(col.data) {
					value = col.data[rowIdx]
				}
				colLetter, _ := utils.CalcColIndex(colIdx)
				colName := col.name

				rowData[colIdx] = value

				if filterFunc(colLetter, colName, value) {
					keepRow = true
				}
			}

			if keepRow {
				filteredRowIndices = append(filteredRowIndices, rowIdx)
				for colIdx, value := range rowData {
					filteredCols[colIdx].data = append(filteredCols[colIdx].data, value)
					filteredCols[colIdx].name = dt.columns[colIdx].name
				}
			}
		}

		newDt := &DataTable{
			columns:           filteredCols,
			rowNames:          filterRowNames(dt.rowNames, filteredRowIndices),
			name:              dt.name,
			creationTimestamp: dt.creationTimestamp,
		}

		newDt.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
		result = newDt
	})
	return result
}

// cloneColumns deep-copies the selected columns so a filtered table owns its
// storage. Sharing the *DataList pointers let a filtered table and its source
// mutate each other through two independent actor locks.
func cloneColumns(cols []*DataList) []*DataList {
	out := make([]*DataList, len(cols))
	for i, col := range cols {
		c := &DataList{name: col.name, creationTimestamp: col.creationTimestamp}
		c.data = append([]any(nil), col.data...)
		c.lastModifiedTimestamp.Store(col.lastModifiedTimestamp.Load())
		out[i] = c
	}
	return out
}

// cloneRowNames copies a row-name index, yielding an empty index for nil so a
// result table always has a usable rowNames.
func cloneRowNames(idx *core.BiIndex) *core.BiIndex {
	if idx == nil {
		return core.NewBiIndex(0)
	}
	return idx.Clone()
}
