package insyra

import "github.com/HazelnutParadise/insyra/internal/core"

// SetRowNameByIndex sets the name of the row at the given index.
// Parameters:
//   - index: The row index (0-based). Supports negative indices (e.g., -1 for the last row).
//   - name: The name to set for the row. If empty, the row name is removed.
//
// Returns:
//   - *DataTable: The DataTable instance for chaining.
//
// Example:
//
//	dt.SetRowNameByIndex(0, "FirstRow")
func (dt *DataTable) SetRowNameByIndex(index int, name string) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		if dt.rowNames == nil {
			dt.rowNames = core.NewBiIndex(0)
		}
		originalIndex := index
		if index < 0 {
			index = dt.getMaxColLength() + index
		}
		if index < 0 || index >= dt.getMaxColLength() {
			dt.warn("SetRowNameByIndex", "Row index %d is out of range, returning", originalIndex)
			return
		}

		if name == "" {
			_, _ = dt.rowNames.DeleteByID(index)
			go dt.updateTimestamp()
			return
		}

		srn := safeRowName(dt, name)
		_, _ = dt.rowNames.Set(index, srn)
		go dt.updateTimestamp()
	})
	return dt
}

// GetRowNameByIndex returns the name of the row at the given index.
// Parameters:
//   - index: The row index (0-based). Supports negative indices (e.g., -1 for the last row).
//
// Returns:
//   - string: The name of the row. Returns empty string if no name is set for this row.
//   - bool: true if a row name exists for this index, false otherwise.
//
// Example:
//
//	name, exists := dt.GetRowNameByIndex(0)
//	if exists {
//	    fmt.Println("Row name:", name)
//	}
func (dt *DataTable) GetRowNameByIndex(index int) (string, bool) {
	var result string
	var exists bool
	dt.AtomicDo(func(dt *DataTable) {
		if index < 0 {
			index = dt.getMaxColLength() + index
		}
		result, exists = dt.getRowNameByIndex(index)
	})
	return result, exists
}

// GetRowIndexByName returns the index of the row with the given name.
// Parameters:
//   - name: The name of the row to find.
//
// Returns:
//   - int: The row index (0-based). Returns -1 if the row name does not exist.
//   - bool: true if the row name exists, false otherwise.
//
// Note:
//
//	Since Insyra's Get methods usually support -1 as an index (representing the last element),
//	always check the boolean return value to distinguish between "name not found" and "last row".
//
// Example:
//
//	index, exists := dt.GetRowIndexByName("MyRow")
//	if exists {
//	    row := dt.GetRow(index)
//	}
func (dt *DataTable) GetRowIndexByName(name string) (int, bool) {
	var index = -1
	var exists bool
	dt.AtomicDo(func(dt *DataTable) {
		var idx int
		idx, exists = dt.rowNames.Index(name)
		if exists {
			index = idx
		}
	})
	if !exists {
		dt.warn("GetRowIndexByName", "Row name not found: %s, returning -1", name)
	}
	return index, exists
}

// ChangeRowName changes the name of a row from oldName to newName.
// Parameters:
//   - oldName: The current name of the row.
//   - newName: The new name to set for the row.
//
// Returns:
//   - *DataTable: The DataTable instance for chaining.
func (dt *DataTable) ChangeRowName(oldName, newName string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		if dt.rowNames == nil {
			dt.rowNames = core.NewBiIndex(0)
		}
		if index, exists := dt.rowNames.Index(oldName); exists {
			if newName == "" {
				_, _ = dt.rowNames.DeleteByID(index)
			} else {
				_, _ = dt.rowNames.Set(index, newName)
			}
		}
		dt.updateTimestamp()
		result = dt
	})
	return result
}

// RowNamesToFirstCol moves the row names to the first column of the DataTable.
// This will clear the row names map and insert a new column at the beginning.
// Returns:
//   - *DataTable: The DataTable instance for chaining.
func (dt *DataTable) RowNamesToFirstCol() *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		if dt.rowNames == nil {
			dt.rowNames = core.NewBiIndex(0)
		}
		rowNames := NewDataList()
		for range dt.columns {
			rowNames.Append("")
		}
		for _, index := range dt.rowNames.IDs() {
			name, ok := dt.rowNames.Get(index)
			if !ok || index < 0 || index >= len(dt.columns) {
				continue
			}
			rowNames.data[index] = name
		}
		dt.columns = append([]*DataList{rowNames}, dt.columns...)

		dt.rowNames = core.NewBiIndex(0)
		go dt.updateTimestamp()
	})
	return dt
}

// DropRowNames removes all row names from the DataTable.
// Returns:
//   - *DataTable: The DataTable instance for chaining.
func (dt *DataTable) DropRowNames() *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		if dt.rowNames == nil || dt.rowNames.Len() == 0 {
			dt.warn("DropRowNames", "No row names to drop")
			return
		}

		dt.rowNames = core.NewBiIndex(0)
		go dt.updateTimestamp()
	})
	return dt
}

// RowNames returns a slice of all row names in the DataTable.
// Rows without names will have an empty string.
// Returns:
//   - []string: A slice containing the names of all rows.
func (dt *DataTable) RowNames() []string {
	var rowNames []string
	dt.AtomicDo(func(dt *DataTable) {
		rowNames = make([]string, 0, len(dt.columns))
		for i := 0; i < dt.getMaxColLength(); i++ {
			if rowName, exists := dt.getRowNameByIndex(i); exists {
				rowNames = append(rowNames, rowName)
			} else {
				rowNames = append(rowNames, "")
			}
		}
	})
	return rowNames
}

// SetRowNames sets the row names of the DataTable.
// Different from SetColNames, it only sets names for existing rows.
func (dt *DataTable) SetRowNames(rowNames []string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		if dt.rowNames == nil {
			dt.rowNames = core.NewBiIndex(0)
		}
		maxRows := dt.getMaxColLength()
		for i, name := range rowNames {
			if i < maxRows {
				dt.SetRowNameByIndex(i, name)
			}
		}
		// Set excess rows to empty name
		for i := len(rowNames); i < maxRows; i++ {
			_, _ = dt.rowNames.DeleteByID(i)
		}
		result = dt
	})
	return result
}
