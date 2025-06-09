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
