package insyra

func (dt *DataTable) ChangeColName(oldName, newName string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	for _, col := range dt.columns {
		if col.name == oldName {
			col.name = newName
			break
		}
	}
	dt.updateTimestamp()
	return dt
}

func (dt *DataTable) ChangeColNameByNumber(numberIndex int, newName string) *DataTable {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if numberIndex < 0 {
		numberIndex += len(dt.columns)
	}
	if numberIndex < 0 || numberIndex >= len(dt.columns) {
		LogWarning("DataTable", "ChangeColNameByNumber", "index out of range")
		return dt
	}
	dt.columns[numberIndex].name = newName
	// 更新時間戳
	dt.updateTimestamp()
	return dt
}

func (dt *DataTable) GetColNameByNumber(index int) string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if index < 0 {
		index += len(dt.columns)
	}
	if index < 0 || index >= len(dt.columns) {
		LogWarning("DataTable", "GetColNameByNumber", "index out of range")
		return ""
	}
	return dt.columns[index].name
}
