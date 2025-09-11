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

// GetRowNameByIndex returns the name of the row at the given index.
func (dt *DataTable) GetRowNameByIndex(index int) string {
	var result string
	dt.AtomicDo(func(dt *DataTable) {
		if rowName, exists := dt.getRowNameByIndex(index); exists {
			result = rowName
		} else {
			result = ""
		}
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
