package insyra

func (dt *DataTable) SetColNameByIndex(index string, name string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		nIndex := ParseColIndex(index)
		name = safeColName(dt, name)

		if nIndex < 0 || nIndex >= len(dt.columns) {
			LogWarning("DataTable", "SetColNameByIndex", "Index out of bounds")
			result = dt
			return
		}

		dt.columns[nIndex].name = name
		go dt.updateTimestamp()
		result = dt
	})
	return result
}

func (dt *DataTable) SetColNameByNumber(numberIndex int, name string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		if numberIndex < 0 {
			numberIndex += len(dt.columns)
		}

		if numberIndex < 0 || numberIndex >= len(dt.columns) {
			LogWarning("DataTable", "SetColNameByNumber", "Index out of bounds")
			result = dt
			return
		}

		name = safeColName(dt, name)
		dt.columns[numberIndex].name = name
		go dt.updateTimestamp()
		result = dt
	})
	return result
}

func (dt *DataTable) ChangeColName(oldName, newName string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		for _, col := range dt.columns {
			if col.name == oldName {
				col.name = newName
				break
			}
		}
		dt.updateTimestamp()
		result = dt
	})
	return result
}

func (dt *DataTable) GetColNameByNumber(index int) string {
	var result string
	dt.AtomicDo(func(dt *DataTable) {
		if index < 0 {
			index += len(dt.columns)
		}
		if index < 0 || index >= len(dt.columns) {
			LogWarning("DataTable", "GetColNameByNumber", "index out of range")
			result = ""
			return
		}
		result = dt.columns[index].name
	})
	return result
}

// GetColNameByIndex gets the column name by its Excel-style index (A, B, C, ..., Z, AA, AB, ...).
func (dt *DataTable) GetColNameByIndex(index string) string {
	var result string
	dt.AtomicDo(func(dt *DataTable) {
		nIndex := ParseColIndex(index)
		result = dt.GetColNameByNumber(nIndex)
	})
	return result
}

func (dt *DataTable) GetColNumberByName(name string) int {
	var result = -1
	dt.AtomicDo(func(dt *DataTable) {
		for i, col := range dt.columns {
			if col.name == name {
				result = i
				return
			}
		}
	})
	if result == -1 {
		LogWarning("DataTable", "GetColIndexByName", "Column name not found: %s, returning -1", name)
	}
	return result
}

func (dt *DataTable) ColNamesToFirstRow() *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		if len(dt.columns) == 0 {
			LogWarning("DataTable", "ColNamesToFirstRow", "No columns to set names")
			result = dt
			return
		}

		for _, col := range dt.columns {
			col.data = append([]any{col.name}, col.data...)
			col.name = "" // Clear the name after moving it to the first row
		}
		go dt.updateTimestamp()
		result = dt
	})
	return result
}

func (dt *DataTable) DropColNames() *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		if len(dt.columns) == 0 {
			LogWarning("DataTable", "DropColNames", "No columns to drop names")
			result = dt
			return
		}

		for _, col := range dt.columns {
			col.name = ""
		}
		go dt.updateTimestamp()
		result = dt
	})
	return result
}

func (dt *DataTable) ColNames() []string {
	var result []string
	dt.AtomicDo(func(dt *DataTable) {
		names := make([]string, len(dt.columns))
		for i, col := range dt.columns {
			names[i] = col.name
		}
		result = names
	})
	return result
}

// Headers is an alias for ColNames, returning the column names of the DataTable.
func (dt *DataTable) Headers() []string {
	return dt.ColNames()
}

func (dt *DataTable) SetColNames(colNames []string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		for i, colName := range colNames {
			if i < len(dt.columns) {
				dt.columns[i].SetName(safeColName(dt, colName))
			} else {
				// Add new column
				newCol := NewDataList()
				newCol.SetName(safeColName(dt, colName))
				dt.AppendCols(newCol)
			}
		}
		// Set excess columns to empty name
		for i := len(colNames); i < len(dt.columns); i++ {
			dt.columns[i].SetName("")
		}
		go dt.updateTimestamp()
		result = dt
	})
	return result
}

// SetHeaders is an alias for SetColNames, setting the column names of the DataTable.
func (dt *DataTable) SetHeaders(headers []string) *DataTable {
	return dt.SetColNames(headers)
}
