package insyra

func (dt *DataTable) SetRowNameByIndex(index int, name string) {
	dt.AtomicDo(func(dt *DataTable) {
		originalIndex := index
		if index < 0 {
			index = dt.getMaxColLength() + index
		}
		if index < 0 || index >= dt.getMaxColLength() {
			LogWarning("DataTable", "SetRowNameByIndex", "Row index %d is out of range, returning", originalIndex)
			return
		}
		srn := safeRowName(dt, name)
		dt.rowNames[srn] = index
		go dt.updateTimestamp()
	})
}

// GetRowNameByIndex returns the name of the row at the given index.
// Parameters:
//   - index: The row index (0-based). Negative indices are not supported for row names.
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
		idx, exists = dt.rowNames[name]
		if exists {
			index = idx
		}
	})
	if !exists {
		LogWarning("DataTable", "GetRowIndexByName", "Row name not found: %s, returning -1", name)
	}
	return index, exists
}

func (dt *DataTable) ChangeRowName(oldName, newName string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		for srn, index := range dt.rowNames {
			if srn == oldName {
				dt.rowNames[newName] = index
				delete(dt.rowNames, srn)
				break
			}
		}
		dt.updateTimestamp()
		result = dt
	})
	return result
}

// RowNamesToFirstCol moves the row names to the first column of the DataTable.
func (dt *DataTable) RowNamesToFirstCol() *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		rowNames := NewDataList()
		for range dt.columns {
			rowNames.Append("")
		}
		for name, index := range dt.rowNames {
			if index < 0 || index >= len(dt.columns) {
				continue
			}
			rowNames.data[index] = name
		}
		dt.columns = append([]*DataList{rowNames}, dt.columns...)

		dt.rowNames = make(map[string]int)
		go dt.updateTimestamp()
	})
	return dt
}

func (dt *DataTable) DropRowNames() *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		if len(dt.rowNames) == 0 {
			LogWarning("DataTable", "DropRowNames", "No row names to drop")
			return
		}

		dt.rowNames = make(map[string]int)
		go dt.updateTimestamp()
	})
	return dt
}

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
		maxRows := dt.getMaxColLength()
		for i, name := range rowNames {
			if i < maxRows {
				dt.SetRowNameByIndex(i, name)
			}
		}
		// Set excess rows to empty name
		for i := len(rowNames); i < maxRows; i++ {
			if _, exists := dt.getRowNameByIndex(i); exists {
				// Remove the name by deleting from map
				for name, idx := range dt.rowNames {
					if idx == i {
						delete(dt.rowNames, name)
						break
					}
				}
			}
		}
		result = dt
	})
	return result
}
