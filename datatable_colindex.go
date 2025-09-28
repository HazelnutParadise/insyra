package insyra

// GetColIndexByName returns the column index (A, B, C, ...) by its name.
func (dt *DataTable) GetColIndexByName(name string) string {
	var result = ""
	dt.AtomicDo(func(dt *DataTable) {
		colNumber := dt.GetColNumberByName(name)
		if colNumber != -1 {
			result = dt.GetColIndexByNumber(colNumber)
		}
	})
	if result == "" {
		LogWarning("DataTable", "GetColIndexByName", "Column name not found: %s, returning empty string", name)
	}
	return result
}

// GetColIndexByNumber returns the column index (A, B, C, ...) by its number (0, 1, 2, ...).
func (dt *DataTable) GetColIndexByNumber(number int) string {
	var result = ""
	dt.AtomicDo(func(dt *DataTable) {
		if number < 0 {
			number += len(dt.columns)
		}
		for i, num := range dt.columnIndex {
			if num == number {
				result = i
				return
			}
		}
	})
	if result == "" {
		LogWarning("DataTable", "GetColIndexByNumber", "Column number not found: %d, returning empty string", number)
	}
	return result
}
