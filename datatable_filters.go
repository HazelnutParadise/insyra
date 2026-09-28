package insyra

import (
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/internal/core"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

// ==================== Slices ====================

// SliceRows returns rows from through to-1 as a new DataTable, the way
// s[from:to] slices a Go slice. The bounds are 0-based positions with
// 0 <= from <= to <= NumRows(); from == to gives the table's columns with no
// rows. The result keeps the table's name, the column names and the names of
// the rows it holds, and owns its data. A bound outside that range records an
// error on the table and returns an empty DataTable.
func (dt *DataTable) SliceRows(from, to int) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		numRows := dt.getMaxColLength()
		if from < 0 || from > to || to > numRows {
			dt.fail("SliceRows", "bounds [%d:%d] out of range for %d rows; want 0 <= from <= to <= %d", from, to, numRows, numRows)
			return
		}
		cols := make([]*DataList, len(dt.columns))
		for i, col := range dt.columns {
			data := make([]any, to-from)
			for r := from; r < to && r < len(col.data); r++ {
				data[r-from] = col.data[r]
			}
			cols[i] = &DataList{data: data, name: col.name, creationTimestamp: col.creationTimestamp}
			cols[i].lastModifiedTimestamp.Store(col.lastModifiedTimestamp.Load())
		}
		rows := make([]int, to-from)
		for i := range rows {
			rows[i] = from + i
		}
		result = &DataTable{
			columns:           cols,
			rowNames:          filterRowNames(dt.rowNames, rows),
			name:              dt.name,
			creationTimestamp: dt.creationTimestamp,
		}
		result.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
	})
	if result == nil {
		return NewDataTable()
	}
	return result
}

// SliceCols returns the columns from from up to, but not including, to as a
// new DataTable, the way s[from:to] slices a Go slice.
//
// Each bound is a column selector: an Excel-style letter ("B"), a Name
// (Name("price")) or an int position. The three spellings of one column are
// the same bound, so SliceCols("B", "D"), SliceCols(1, 3) and, where those
// columns are named b and d, SliceCols(Name("b"), Name("d")) all give columns
// B and C. An int may be negative, counting from the end, and may be
// NumCols(), one past the last column. nil is the first column for from and
// one past the last for to, so SliceCols("C", nil) runs to the end. from == to
// gives a table with no columns.
//
// The result keeps every row, the row names and the table's name, and owns
// its data. A bound that picks no column, lies outside the table, or comes
// after the other records an error on the table and returns an empty
// DataTable.
func (dt *DataTable) SliceCols(from, to any) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		numCols := len(dt.columns)
		start, ok := dt.sliceColBound(from, 0)
		if !ok {
			return
		}
		end, ok := dt.sliceColBound(to, numCols)
		if !ok {
			return
		}
		if start > end {
			dt.fail("SliceCols", "from %v (column %d) comes after to %v (column %d)", from, start, to, end)
			return
		}
		rowNames := core.NewBiIndex(0)
		if start < end {
			rowNames = cloneRowNames(dt.rowNames)
		}
		result = &DataTable{
			columns:           cloneColumns(dt.columns[start:end]),
			rowNames:          rowNames,
			name:              dt.name,
			creationTimestamp: dt.creationTimestamp,
		}
		result.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
	})
	if result == nil {
		return NewDataTable()
	}
	return result
}

// sliceColBound turns one bound of SliceCols into a position from 0 to the
// column count. nil gives ifNil. An int is a slice bound, so the column count
// itself is allowed; a letter or a Name picks a column through the one
// selector rule. The table must already be locked.
func (dt *DataTable) sliceColBound(bound any, ifNil int) (int, bool) {
	numCols := len(dt.columns)
	switch v := bound.(type) {
	case nil:
		return ifNil, true
	case int:
		position := v
		if position < 0 {
			position += numCols
		}
		if position < 0 || position > numCols {
			dt.fail("SliceCols", "bound %d is out of range for %d columns; an int bound runs from %d to %d", v, numCols, -numCols, numCols)
			return 0, false
		}
		return position, true
	}
	return dt.resolveColSelector("SliceCols", bound)
}

// ==================== Col Index ====================

// FilterColsByColIndexGreaterThan keeps the columns after the one at
// columnIndexLetter. An unreadable letter, the last column or one past it
// gives an empty DataTable.
//
// Deprecated: use SliceCols(i+1, nil), where i is the column's position;
// ParseColIndex turns a letter into one. SliceCols reports a bound past the
// last column as an error instead of returning an empty table.
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

// FilterColsByColIndexGreaterThanOrEqualTo keeps the column at
// columnIndexLetter and every column after it. An unreadable letter, or one
// past the last column, gives an empty DataTable.
//
// Deprecated: use SliceCols(columnIndexLetter, nil). SliceCols reports a
// letter past the last column as an error instead of returning an empty
// table.
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

// FilterColsByColIndexEqualTo keeps only the column at columnIndexLetter. An
// unreadable letter, or one past the last column, gives an empty DataTable.
//
// Deprecated: use SliceCols(columnIndexLetter, i+1), where i is the column's
// position; ParseColIndex turns a letter into one. To pick a column as a
// DataList, use GetCol.
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

// FilterColsByColIndexLessThan keeps the columns before the one at
// columnIndexLetter; a letter past the last column keeps them all. An
// unreadable letter, or column A, gives an empty DataTable.
//
// Deprecated: use SliceCols(nil, columnIndexLetter). SliceCols reports a
// letter past the last column as an error.
func (dt *DataTable) FilterColsByColIndexLessThan(columnIndexLetter string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		columnIndexLetter = strings.ToUpper(columnIndexLetter)
		colIdx, ok := utils.ParseColIndex(columnIndexLetter)
		if !ok || colIdx <= 0 {
			result = NewDataTable()
			return
		}

		filteredCols := dt.columns[:min(colIdx, len(dt.columns))]

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

// FilterColsByColIndexLessThanOrEqualTo keeps the column at
// columnIndexLetter and every column before it; a letter past the last column
// keeps them all. An unreadable letter gives an empty DataTable.
//
// Deprecated: use SliceCols(nil, i+1), where i is the column's position;
// ParseColIndex turns a letter into one. SliceCols reports a bound past the
// last column as an error.
func (dt *DataTable) FilterColsByColIndexLessThanOrEqualTo(columnIndexLetter string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		columnIndexLetter = strings.ToUpper(columnIndexLetter)
		colIdx, ok := utils.ParseColIndex(columnIndexLetter)
		if !ok || colIdx < 0 {
			result = NewDataTable()
			return
		}

		filteredCols := dt.columns[:min(colIdx+1, len(dt.columns))]

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

// FilterRowsByRowIndexGreaterThan keeps the rows after row threshold.
//
// Deprecated: use SliceRows(threshold+1, dt.NumRows()). SliceRows reports a
// bound outside the table as an error; this method clamps it.
func (dt *DataTable) FilterRowsByRowIndexGreaterThan(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return rowIndex > threshold
	})
}

// FilterRowsByRowIndexGreaterThanOrEqualTo keeps row threshold and the rows
// after it.
//
// Deprecated: use SliceRows(threshold, dt.NumRows()). SliceRows reports a
// bound outside the table as an error; this method clamps it.
func (dt *DataTable) FilterRowsByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return rowIndex >= threshold
	})
}

// FilterRowsByRowIndexEqualTo keeps only row index.
//
// Deprecated: use SliceRows(index, index+1). SliceRows reports a row outside
// the table as an error; this method returns no rows. To read one row as a
// DataList, use GetRow.
func (dt *DataTable) FilterRowsByRowIndexEqualTo(index int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return rowIndex == index
	})
}

// FilterRowsByRowIndexLessThan keeps the rows before row threshold.
//
// Deprecated: use SliceRows(0, threshold). SliceRows reports a bound outside
// the table as an error; this method clamps it.
func (dt *DataTable) FilterRowsByRowIndexLessThan(threshold int) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return rowIndex < threshold
	})
}

// FilterRowsByRowIndexLessThanOrEqualTo keeps row threshold and the rows
// before it.
//
// Deprecated: use SliceRows(0, threshold+1). SliceRows reports a bound
// outside the table as an error; this method clamps it.
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

// FilterByCustomElement keeps the rows in which filterFunc returns true for at
// least one cell. It is Filter without the row and column arguments.
//
// Deprecated: use Filter, which does the same thing:
//
//	dt.Filter(func(_ int, _ string, value any) bool { return filterFunc(value) })
//
// To judge a whole row at once, use FilterRowsWhere.
func (dt *DataTable) FilterByCustomElement(filterFunc func(value any) bool) *DataTable {
	return dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return filterFunc(value)
	})
}

// ==================== Custom Filter ====================

// Filter keeps a row when filterFunc returns true for any one of its cells,
// and returns the kept rows as a new DataTable. filterFunc is called cell by
// cell with the row's 0-based index, the column's Excel-style letter ("A",
// "B", ...) and the cell's value; a row is kept at the first cell that passes.
// Because each call sees one cell, Filter cannot compare cells of the same
// row; use FilterRowsWhere for that.
func (dt *DataTable) Filter(filterFunc func(rowIndex int, columnIndex string, value any) bool) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		// The predicate runs inside the table's lock and may call the
		// table's own methods; work from the columns as they were when the
		// call began, so a column added meanwhile cannot run past the result.
		cols := dt.columns
		filteredCols := make([]*DataList, len(cols))
		for i := range cols {
			// Preserve the original column name (and metadata) in the result;
			// a bare &DataList{} would silently drop every column name.
			filteredCols[i] = &DataList{name: cols[i].name}
		}

		var filteredRowIndices []int
		// Guard against an empty table (no columns): cols[0] would panic.
		if len(cols) > 0 {
			cellAt := func(col *DataList, rowIdx int) any {
				if rowIdx < len(col.data) {
					return col.data[rowIdx]
				}
				return nil
			}
			for rowIdx := range cols[0].data {
				// A row is kept if the predicate matches ANY cell in it.
				keepRow := false
				for colIdx, col := range cols {
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
				for colIdx, col := range cols {
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

// ==================== Filter Rows Where ====================

// FilterRowsWhere keeps the rows for which keep returns true, and returns
// them as a new DataTable with the table's name, its column names and the
// kept rows' names.
//
// keep is called once per row with the whole row: a DataList holding the
// row's cells in column order and named with the row's name. That lets a
// condition compare cells of the same row, which Filter and FilterRows, calling
// their function one cell at a time, cannot:
//
//	cheap := dt.FilterRowsWhere(func(row *insyra.DataList) bool {
//		return insyra.ToFloat64(row.Get(0)) < insyra.ToFloat64(row.Get(1))
//	})
//
// The row is a copy: changing it changes neither the table nor the result. A
// nil keep records an error and returns an empty DataTable.
func (dt *DataTable) FilterRowsWhere(keep func(row *DataList) bool) *DataTable {
	if keep == nil {
		dt.fail("FilterRowsWhere", "keep is nil")
		return NewDataTable()
	}
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		// keep runs inside the table's lock and may call the table's own
		// methods; the columns as they were when the call began are what
		// the result is built from, so a column added meanwhile cannot run
		// past filteredCols.
		cols := dt.columns
		filteredCols := make([]*DataList, len(cols))
		for i, col := range cols {
			filteredCols[i] = &DataList{name: col.name, creationTimestamp: col.creationTimestamp}
			filteredCols[i].lastModifiedTimestamp.Store(col.lastModifiedTimestamp.Load())
		}
		now := time.Now().Unix()
		numRows := dt.getMaxColLength()
		var kept []int
		for rowIdx := 0; rowIdx < numRows; rowIdx++ {
			cells := make([]any, len(cols))
			for colIdx, col := range cols {
				if rowIdx < len(col.data) {
					cells[colIdx] = col.data[rowIdx]
				}
			}
			// Built by hand rather than with NewDataList, which would split a
			// slice cell into several.
			row := &DataList{data: cells, creationTimestamp: now}
			row.lastModifiedTimestamp.Store(now)
			row.name, _ = dt.getRowNameByIndex(rowIdx)
			if !keep(row) {
				continue
			}
			// Copy from the table, not from row, which keep may have changed.
			for colIdx, col := range cols {
				var v any
				if rowIdx < len(col.data) {
					v = col.data[rowIdx]
				}
				filteredCols[colIdx].data = append(filteredCols[colIdx].data, v)
			}
			kept = append(kept, rowIdx)
		}

		result = &DataTable{
			columns:           filteredCols,
			rowNames:          filterRowNames(dt.rowNames, kept),
			name:              dt.name,
			creationTimestamp: dt.creationTimestamp,
		}
		result.lastModifiedTimestamp.Store(dt.lastModifiedTimestamp.Load())
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

// FilterRows keeps a row when filterFunc returns true for any one of its
// cells, like Filter. filterFunc receives the column's Excel-style letter, the
// column's name and the cell's value. To judge a whole row at once, use
// FilterRowsWhere.
func (dt *DataTable) FilterRows(filterFunc func(colIndex, colName string, x any) bool) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		// The predicate runs inside the table's lock and may call the
		// table's own methods; work from the columns as they were when the
		// call began, so a column added meanwhile cannot run past the result.
		cols := dt.columns
		filteredCols := make([]*DataList, len(cols))
		for i := range cols {
			filteredCols[i] = NewDataList()
		}

		numRows := dt.getMaxColLength()

		var filteredRowIndices []int
		for rowIdx := 0; rowIdx < numRows; rowIdx++ {
			keepRow := false
			rowData := make([]any, len(cols))

			for colIdx, col := range cols {
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
					filteredCols[colIdx].name = cols[colIdx].name
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
