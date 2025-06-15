package insyra

func (dt *DataTable) SetRowNameByIndex(index int, name string) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
	}()
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
}

func (dt *DataTable) ChangeRowName(oldName, newName string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	for srn, index := range dt.rowNames {
		if srn == oldName {
			dt.rowNames[newName] = index
			delete(dt.rowNames, srn)
			break
		}
	}
	dt.updateTimestamp()
	return dt
}

// GetRowNameByIndex returns the name of the row at the given index.
func (dt *DataTable) GetRowNameByIndex(index int) string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if rowName, exists := dt.getRowNameByIndex(index); exists {
		return rowName
	} else {
		return ""
	}
}

// RowNamesToFirstCol moves the row names to the first column of the DataTable.
func (dt *DataTable) RowNamesToFirstCol() *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

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

	return dt
}

func (dt *DataTable) DropRowNames() *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if len(dt.rowNames) == 0 {
		LogWarning("DataTable", "DropRowNames", "No row names to drop")
		return dt
	}

	dt.rowNames = make(map[string]int)

	return dt
}

func (dt *DataTable) RowNames() []string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	rowNames := make([]string, 0, len(dt.columns))
	for i := 0; i < dt.getMaxColLength(); i++ {
		if rowName, exists := dt.getRowNameByIndex(i); exists {
			rowNames = append(rowNames, rowName)
		} else {
			rowNames = append(rowNames, "")
		}
	}
	return rowNames
}
