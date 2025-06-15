package insyra

func (dt *DataTable) SetColNameByIndex(index string, name string) *DataTable {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()
	nIndex := ParseColIndex(index)
	name = safeColName(dt, name)

	if nIndex < 0 || nIndex >= len(dt.columns) {
		LogWarning("DataTable", "SetColNameByIndex", "Index out of bounds")
		return dt
	}

	dt.columns[nIndex].name = name

	return dt
}

func (dt *DataTable) SetColNameByNumber(numberIndex int, name string) *DataTable {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()
	if numberIndex < 0 {
		numberIndex += len(dt.columns)
	}

	if numberIndex < 0 || numberIndex >= len(dt.columns) {
		LogWarning("DataTable", "SetColNameByNumber", "Index out of bounds")
		return dt
	}

	name = safeColName(dt, name)
	dt.columns[numberIndex].name = name

	return dt
}

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

func (dt *DataTable) ColNamesToFirstRow() *DataTable {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	if len(dt.columns) == 0 {
		LogWarning("DataTable", "ColNamesToFirstRow", "No columns to set names")
		return dt
	}

	for _, col := range dt.columns {
		col.data = append([]any{col.name}, col.data...)
		col.name = "" // Clear the name after moving it to the first row
	}

	return dt
}

func (dt *DataTable) DropColNames() *DataTable {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	if len(dt.columns) == 0 {
		LogWarning("DataTable", "DropColNames", "No columns to drop names")
		return dt
	}

	for _, col := range dt.columns {
		col.name = ""
	}

	return dt
}

func (dt *DataTable) ColNames() []string {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	names := make([]string, len(dt.columns))
	for i, col := range dt.columns {
		names[i] = col.name
	}
	return names
}
