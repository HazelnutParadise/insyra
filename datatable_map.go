package insyra

// Map applies a function to all elements in the DataTable and returns a new DataTable with the results.
// The mapFunc should take three parameters: row index (int), column index (string), and element (any),
// then return a transformed value of any type.
func (dt *DataTable) Map(mapFunc func(rowIndex int, colIndex string, element any) any) *DataTable {
	defer func() {
		dt.mu.Unlock()
	}()
	dt.mu.Lock()

	// 直接計算大小以避免死鎖
	numRows := dt.getMaxColLength()
	numCols := len(dt.columns)
	if numRows == 0 || numCols == 0 {
		LogWarning("DataTable", "Map", "DataTable is empty, returning empty DataTable.")
		return NewDataTable()
	} // 創建新的 DataTable
	newDt := NewDataTable()

	// 直接按順序處理每個列
	for colPos := 0; colPos < numCols; colPos++ {
		originalCol := dt.columns[colPos]
		// 生成列索引（A, B, C...）
		colIndex := generateColIndex(colPos)

		newCol := NewDataList()
		newCol.SetName(originalCol.GetName()) // 保持原來的列名

		for rowIndex := 0; rowIndex < numRows; rowIndex++ {
			func() {
				defer func() {
					if r := recover(); r != nil {
						LogWarning("DataTable", "Map", "Error applying function to element at row %d, column %s: %v, keeping original value.", rowIndex, colIndex, r)
						// 保留原始值
						if rowIndex < originalCol.Len() {
							newCol.Append(originalCol.Get(rowIndex))
						} else {
							newCol.Append(nil)
						}
					}
				}()

				var originalValue any
				if rowIndex < originalCol.Len() {
					originalValue = originalCol.Get(rowIndex)
				} else {
					originalValue = nil
				}

				transformedValue := mapFunc(rowIndex, colIndex, originalValue)
				newCol.Append(transformedValue)
			}()
		}

		newDt.AppendCols(newCol)
	}

	// 直接複製行名稱以避免死鎖
	for rowIndex := 0; rowIndex < numRows; rowIndex++ {
		if rowName, exists := dt.getRowNameByIndex(rowIndex); exists {
			newDt.SetRowNameByIndex(rowIndex, rowName)
		}
	}

	return newDt
}
