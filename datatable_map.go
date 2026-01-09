package insyra

import "github.com/HazelnutParadise/insyra/internal/utils"

// Map applies a function to all elements in the DataTable and returns a new DataTable with the results.
// The mapFunc should take three parameters: row index (int), column index (string), and element (any),
// then return a transformed value of any type.
func (dt *DataTable) Map(mapFunc func(rowIndex int, colIndex string, element any) any) *DataTable {
	var newDt *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		numRows := dt.getMaxColLength()
		numCols := len(dt.columns)
		if numRows == 0 || numCols == 0 {
			dt.warn("Map", "DataTable is empty, returning empty DataTable.")
			newDt = NewDataTable()
			return
		}
		newDt = NewDataTable()

		for colPos := range numCols {
			originalCol := dt.columns[colPos]
			colIndex, _ := utils.CalcColIndex(colPos)

			newCol := NewDataList()
			newCol.SetName(originalCol.GetName())

			for rowIndex := range numRows {
				func() {
					defer func() {
						if r := recover(); r != nil {
							dt.warn("Map", "Error applying function to element at row %d, column %s: %v, keeping original value.", rowIndex, colIndex, r)
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
		for rowIndex := range numRows {
			if rowName, exists := dt.getRowNameByIndex(rowIndex); exists {
				newDt.SetRowNameByIndex(rowIndex, rowName)
			}
		}
	})
	return newDt
}
