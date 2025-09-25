package insyra

import (
	"fmt"
	"maps"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HazelnutParadise/Go-Utils/asyncutil"
	"github.com/HazelnutParadise/Go-Utils/conv"
)

type DataTable struct {
	columns               []*DataList
	columnIndex           map[string]int // 儲存字母索引與切片中的索引對應
	rowNames              map[string]int
	name                  string // 新增 name 欄位
	creationTimestamp     int64
	lastModifiedTimestamp atomic.Int64

	// AtomicDo support
	cmdCh    chan func()
	initOnce sync.Once
	closed   atomic.Bool
}

func NewDataTable(columns ...*DataList) *DataTable {
	now := time.Now().Unix()
	newTable := &DataTable{
		columns:           []*DataList{},
		columnIndex:       make(map[string]int),
		rowNames:          make(map[string]int),
		creationTimestamp: now,
	}

	newTable.lastModifiedTimestamp.Store(now)

	if len(columns) > 0 {
		newTable.AppendCols(columns...)
	}

	return newTable
}

// ======================== Append ========================

// AppendCols appends columns to the DataTable, with each column represented by a DataList.
// If the columns are shorter than the existing columns, nil values will be appended to match the length.
// If the columns are longer than the existing columns, the existing columns will be extended with nil values.
func (dt *DataTable) AppendCols(columns ...*DataList) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		maxLength := dt.getMaxColLength()

		for _, col := range columns {
			column := NewDataList()
			column.data = col.data
			column.name = col.name
			columnName := generateColIndex(len(dt.columns)) // 修改這行確保按順序生成列名
			column.name = safeColName(dt, column.name)

			dt.columns = append(dt.columns, column)
			dt.columnIndex[columnName] = len(dt.columns) - 1
			if len(column.data) < maxLength {
				column.data = append(column.data, make([]any, maxLength-len(column.data))...)
			}
			LogDebug("DataTable", "AppendCols", "Added column %s at index %d", columnName, dt.columnIndex[columnName])
		}

		for _, col := range dt.columns {
			if len(col.data) < maxLength {
				col.data = append(col.data, make([]any, maxLength-len(col.data))...)
			}
		}
		go dt.updateTimestamp()
	})
	return dt
}

// AppendRowsFromDataList appends rows to the DataTable, with each row represented by a DataList.
// If the rows are shorter than the existing columns, nil values will be appended to match the length.
// If the rows are longer than the existing columns, the existing columns will be extended with nil values.
func (dt *DataTable) AppendRowsFromDataList(rowsData ...*DataList) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		for _, rowData := range rowsData {
			maxLength := dt.getMaxColLength()

			if rowData.name != "" {
				srn := safeRowName(dt, rowData.name)
				dt.rowNames[srn] = maxLength
			}

			if len(rowData.data) > len(dt.columns) {
				for i := len(dt.columns); i < len(rowData.data); i++ {
					newCol := newEmptyDataList(maxLength)
					columnName := generateColIndex(i)
					dt.columns = append(dt.columns, newCol)
					dt.columnIndex[columnName] = len(dt.columns) - 1
				}
			}

			for i, column := range dt.columns {
				if i < len(rowData.data) {
					column.data = append(column.data, rowData.data[i])
				} else {
					column.data = append(column.data, nil)
				}
			}

			for _, column := range dt.columns {
				if len(column.data) == maxLength {
					column.data = append(column.data, nil)
				}
			}
		}
		go dt.updateTimestamp()
	})
	return dt
}

// AppendRowsByIndex appends rows to the DataTable, with each row represented by a map of column index and value.
// If the rows are shorter than the existing columns, nil values will be appended to match the length.
// If the rows are longer than the existing columns, the existing columns will be extended with nil values.
func (dt *DataTable) AppendRowsByColIndex(rowsData ...map[string]any) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		upperCaseRowsData := make([]map[string]any, len(rowsData))
		for i, rowData := range rowsData {
			upperCaseRowData := make(map[string]any)
			for colIndex, value := range rowData {
				upperCaseRowData[strings.ToUpper(colIndex)] = value
			}
			upperCaseRowsData[i] = upperCaseRowData
		}
		rowsData = upperCaseRowsData

		for _, rowData := range rowsData {
			maxLength := dt.getMaxColLength()

			// 搜集所有要處理的欄位索引（確保無論是否存在都處理）
			allCols := make([]string, 0, len(rowData))
			for colIndex := range rowData {
				allCols = append(allCols, colIndex)
			}

			// 按照字母順序對欄位進行排序
			sort.Strings(allCols)

			// 按照排序順序處理每個欄位
			for _, colIndex := range allCols {
				value := rowData[colIndex]
				_, exists := dt.columnIndex[colIndex]
				LogDebug("DataTable", "AppendRowsByColIndex", "Handling column %s, exists: %t", colIndex, exists)

				if !exists {
					// 如果該欄位不存在，新增該欄位並插入字母順序位置
					newCol := newEmptyDataList(maxLength)
					dt.columns = append(dt.columns, newCol)
					dt.columnIndex[colIndex] = len(dt.columns) - 1
					LogDebug("DataTable", "AppendRowsByColIndex", "Added new column %s at index %d", colIndex, dt.columnIndex[colIndex])

					// 重新排序欄位以符合字母順序
					dt.sortColsByIndex()
				}

				dt.columns[dt.columnIndex[colIndex]].data = append(dt.columns[dt.columnIndex[colIndex]].data, value)
			}

			// 確保所有欄位的長度一致
			for _, column := range dt.columns {
				if len(column.data) <= maxLength {
					column.data = append(column.data, nil)
				}
			}
		}
		go dt.updateTimestamp()
	})
	return dt
}

// AppendRowsByName appends rows to the DataTable, with each row represented by a map of column name and value.
// If the rows are shorter than the existing columns, nil values will be appended to match the length.
// If the rows are longer than the existing columns, the existing columns will be extended with nil values.
func (dt *DataTable) AppendRowsByColName(rowsData ...map[string]any) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		for _, rowData := range rowsData {
			maxLength := dt.getMaxColLength()

			for colName, value := range rowData {
				found := false
				for i := 0; i < len(dt.columns); i++ {
					if dt.columns[i].name == colName {
						dt.columns[i].data = append(dt.columns[i].data, value)
						found = true
						LogDebug("DataTable", "AppendRowsByColName", "Found column %s at index %d", colName, i)
						break
					}
				}
				if !found {
					newCol := newEmptyDataList(maxLength)
					newCol.name = colName
					newCol.data = append(newCol.data, value)
					dt.columns = append(dt.columns, newCol)
					dt.columnIndex[generateColIndex(len(dt.columns)-1)] = len(dt.columns) - 1 // 更新 columnIndex
					LogDebug("DataTable", "AppendRowsByColName", "Added new column %s at index %d", colName, len(dt.columns)-1)
				}
			}

			for _, column := range dt.columns {
				if len(column.data) == maxLength {
					column.data = append(column.data, nil)
				}
			}
		}

		dt.regenerateColIndex()
		go dt.updateTimestamp()
	})
	return dt
}

// ======================== Get ========================

// GetElement returns the element at the given row and column index.
func (dt *DataTable) GetElement(rowIndex int, columnIndex string) any {
	var result any
	dt.AtomicDo(func(dt *DataTable) {
		columnIndex = strings.ToUpper(columnIndex)
		if colPos, exists := dt.columnIndex[columnIndex]; exists {
			if rowIndex < 0 {
				rowIndex = len(dt.columns[colPos].data) + rowIndex
			}
			if rowIndex < 0 || rowIndex >= len(dt.columns[colPos].data) {
				LogWarning("DataTable", "GetElement", "Row index is out of range, returning nil")
				result = nil
				return
			}
			result = dt.columns[colPos].data[rowIndex]
		} else {
			result = nil
		}
	})
	return result
}

func (dt *DataTable) GetElementByNumberIndex(rowIndex int, columnIndex int) any {
	var result any
	dt.AtomicDo(func(dt *DataTable) {
		if rowIndex < 0 {
			rowIndex = len(dt.columns[columnIndex].data) + rowIndex
		}
		if rowIndex < 0 || rowIndex >= len(dt.columns[columnIndex].data) {
			LogWarning("DataTable", "GetElementByNumberIndex", "Row index is out of range, returning nil")
			result = nil
			return
		}
		result = dt.columns[columnIndex].data[rowIndex]
	})
	return result
}

// GetCol returns a new DataList containing the data of the column with the given index.
func (dt *DataTable) GetCol(index string) *DataList {
	var result *DataList
	dt.AtomicDo(func(dt *DataTable) {
		index = strings.ToUpper(index)
		if colPos, exists := dt.columnIndex[index]; exists {
			// 初始化新的 DataList 並分配 data 切片的大小
			dl := NewDataList()
			dl.data = make([]any, len(dt.columns[colPos].data))

			// 拷貝數據到新的 DataList
			copy(dl.data, dt.columns[colPos].data)
			dl.name = dt.columns[colPos].name
			result = dl
			return
		}
		LogWarning("DataTable", "GetCol", "Column '%s' not found, returning empty DataList", index)
		result = NewDataList() // 返回空的 DataList 而不是 nil
	})
	return result
}

func (dt *DataTable) GetColByNumber(index int) *DataList {
	var result *DataList
	dt.AtomicDo(func(dt *DataTable) {
		if index < 0 {
			index = len(dt.columns) + index
		}

		if index < 0 || index >= len(dt.columns) {
			LogWarning("DataTable", "GetColByNumber", "Col index is out of range, returning empty DataList")
			result = NewDataList() // 返回空的 DataList 而不是 nil
			return
		}

		// 初始化新的 DataList 並分配 data 切片的大小
		dl := NewDataList()
		dl.data = make([]any, len(dt.columns[index].data))

		// 拷貝數據到新的 DataList
		copy(dl.data, dt.columns[index].data)
		dl.name = dt.columns[index].name
		result = dl
	})
	return result
}

func (dt *DataTable) GetColByName(name string) *DataList {
	var result *DataList
	dt.AtomicDo(func(dt *DataTable) {
		for _, column := range dt.columns {
			if column.name == name {
				// 初始化新的 DataList 並分配 data 切片的大小
				dl := NewDataList()
				dl.data = make([]any, len(column.data))

				// 拷貝數據到新的 DataList
				copy(dl.data, column.data)
				dl.name = column.name
				result = dl
				return
			}
		}
		LogWarning("DataTable", "GetColByName", "Column '%s' not found, returning nil", name)
		result = nil
	})
	return result
}

// GetRow returns a new DataList containing the data of the row with the given index.
func (dt *DataTable) GetRow(index int) *DataList {
	var result *DataList
	dt.AtomicDo(func(dt *DataTable) {
		if index < 0 {
			index = dt.getMaxColLength() + index
		}
		if index < 0 || index >= dt.getMaxColLength() {
			LogWarning("DataTable", "GetRow", "Row index is out of range, returning nil")
			result = nil
			return
		}

		// 初始化新的 DataList 並分配 data 切片的大小
		dl := NewDataList()
		dl.data = make([]any, len(dt.columns))

		// 拷貝數據到新的 DataList
		for i, column := range dt.columns {
			if index < len(column.data) {
				dl.data[i] = column.data[index]
			}
		}
		dl.name = dt.GetRowNameByIndex(index)
		result = dl
	})
	return result
}

func (dt *DataTable) GetRowByName(name string) *DataList {
	var result *DataList
	dt.AtomicDo(func(dt *DataTable) {
		if index, exists := dt.rowNames[name]; exists {
			// 初始化新的 DataList 並分配 data 切片的大小
			dl := NewDataList()
			// 拷貝數據到新的 DataList
			for _, column := range dt.columns {
				colData := column.data
				if index < column.Len() {
					dl.Append(colData[index])
				}
			}
			dl.name = name
			result = dl
			return
		}
		LogWarning("DataTable", "GetRowByName", "Row name '%s' not found, returning nil", name)
		result = nil
	})
	return result
}

// ======================== Update ========================

// UpdateElement updates the element at the given row and column index.
func (dt *DataTable) UpdateElement(rowIndex int, columnIndex string, value any) {
	dt.AtomicDo(func(dt *DataTable) {
		dt.regenerateColIndex()

		columnIndex = strings.ToUpper(columnIndex)
		if colPos, exists := dt.columnIndex[columnIndex]; exists {
			if rowIndex < 0 {
				rowIndex = len(dt.columns[colPos].data) + rowIndex
			}
			if rowIndex < 0 || rowIndex >= len(dt.columns[colPos].data) {
				LogWarning("DataTable", "UpdateElement", "Row index is out of range, returning")
				return
			}
			dt.columns[colPos].data[rowIndex] = value
		} else {
			LogWarning("DataTable", "UpdateElement", "Col index does not exist, returning")
		}
		go dt.updateTimestamp()
	})
}

// UpdateCol updates the column with the given index.
func (dt *DataTable) UpdateCol(index string, dl *DataList) {
	dt.AtomicDo(func(dt *DataTable) {
		dt.regenerateColIndex()

		index = strings.ToUpper(index)
		if colPos, exists := dt.columnIndex[index]; exists {
			dt.columns[colPos] = dl
		} else {
			LogWarning("DataTable", "UpdateCol", "Col index does not exist, returning")
		}
		go dt.updateTimestamp()
	})
}

// UpdateColByNumber updates the column at the given index.
func (dt *DataTable) UpdateColByNumber(index int, dl *DataList) {
	dt.AtomicDo(func(dt *DataTable) {
		if index < 0 {
			index = len(dt.columns) + index
		}

		if index < 0 || index >= len(dt.columns) {
			LogWarning("DataTable", "UpdateColByNumber", "Index out of bounds")
			return
		}

		dt.columns[index] = dl
		dt.columnIndex[generateColIndex(index)] = index
		go dt.updateTimestamp()
	})
}

// UpdateRow updates the row at the given index.
func (dt *DataTable) UpdateRow(index int, dl *DataList) {
	dt.AtomicDo(func(dt *DataTable) {
		if index < 0 || index >= dt.getMaxColLength() {
			LogWarning("DataTable", "UpdateRow", "Index out of bounds")
			return
		}

		if len(dl.data) > len(dt.columns) {
			LogWarning("DataTable", "UpdateRow", "DataList has more elements than DataTable columns, returning")
			return
		}

		// 更新 DataTable 中對應行的資料
		for i := 0; i < len(dl.data); i++ {
			dt.columns[i].data[index] = dl.data[i]
		}

		// 更新行名
		if dl.name != "" {
			for rowName, rowIndex := range dt.rowNames {
				if rowIndex == index {
					delete(dt.rowNames, rowName)
					break
				}
			}
			srn := safeRowName(dt, dl.name)
			dt.rowNames[srn] = index
		}

		go dt.updateTimestamp()
	})
}

// ======================== Set ========================

// SetColToRowNames sets the row names to the values of the specified column and drops the column.
func (dt *DataTable) SetColToRowNames(columnIndex string) *DataTable {
	columnIndex = strings.ToUpper(columnIndex)
	column := dt.GetCol(columnIndex)
	for i, value := range column.data {
		if value != nil {
			rowName := safeRowName(dt, conv.ToString(value))
			dt.rowNames[rowName] = i
		}
	}

	dt.DropColsByIndex(columnIndex)

	dt.regenerateColIndex()

	go dt.updateTimestamp()
	return dt
}

// SetRowToColNames sets the column names to the values of the specified row and drops the row.
func (dt *DataTable) SetRowToColNames(rowIndex int) *DataTable {
	row := dt.GetRow(rowIndex)
	dt.AtomicDo(func(dt *DataTable) {
		for i, value := range row.data {
			if value != nil {
				columnName := safeColName(dt, conv.ToString(value))
				dt.columns[i].name = columnName
			}
		}
	})
	dt.DropRowsByIndex(rowIndex)

	go dt.updateTimestamp()
	return dt
}

// ======================== Find ========================

// FindRowsIfContains returns the indices of rows that contain the given element.
func (dt *DataTable) FindRowsIfContains(value any) []int {
	var result []int
	dt.AtomicDo(func(dt *DataTable) {
		// 使用 map 來確保行索引唯一性
		indexMap := make(map[int]struct{})

		for _, column := range dt.columns {
			// 找到該列中包含 value 的所有行索引
			indexes := column.FindAll(value)
			for _, index := range indexes {
				indexMap[index] = struct{}{}
			}
		}

		// 將唯一的行索引轉換為 slice
		result = make([]int, 0, len(indexMap))
		for index := range indexMap {
			result = append(result, index)
		}

		// 排序結果以保證順序
		sort.Ints(result)
	})
	return result
}

// FindRowsIfContainsAll returns the indices of rows that contain all the given elements.
func (dt *DataTable) FindRowsIfContainsAll(values ...any) []int {
	var result []int
	dt.AtomicDo(func(dt *DataTable) {
		// 檢查每一行是否包含所有指定的值
		for rowIndex := 0; rowIndex < dt.getMaxColLength(); rowIndex++ {
			foundAll := true

			// 檢查該行中的所有列是否包含指定的值
			for _, value := range values {
				found := false
				for _, column := range dt.columns {
					if rowIndex < len(column.data) && column.data[rowIndex] == value {
						found = true
						break
					}
				}
				if !found {
					foundAll = false
					break
				}
			}

			// 如果該行包含所有指定的值，則將其索引添加到結果中
			if foundAll {
				result = append(result, rowIndex)
			}
		}
	})
	return result
}

// FindRowsIfAnyElementContainsSubstring returns the indices of rows that contain at least one element that contains the given substring.
func (dt *DataTable) FindRowsIfAnyElementContainsSubstring(substring string) []int {
	var matchingRows []int
	dt.AtomicDo(func(dt *DataTable) {
		for rowIndex := 0; rowIndex < dt.getMaxColLength(); rowIndex++ {
			for _, col := range dt.columns {
				if rowIndex < len(col.data) {
					if value, ok := col.data[rowIndex].(string); ok {
						if containsSubstring(value, substring) {
							matchingRows = append(matchingRows, rowIndex)
							break // 一旦找到匹配的元素，跳出內層循環檢查下一行
						}
					}
				}
			}
		}
	})
	return matchingRows
}

// FindRowsIfAllElementsContainSubstring returns the indices of rows that contain all elements that contain the given substring.
func (dt *DataTable) FindRowsIfAllElementsContainSubstring(substring string) []int {
	var matchingRows []int
	dt.AtomicDo(func(dt *DataTable) {
		for rowIndex := 0; rowIndex < dt.getMaxColLength(); rowIndex++ {
			foundAll := true

			for _, col := range dt.columns {
				if rowIndex < len(col.data) {
					if value, ok := col.data[rowIndex].(string); ok {
						if !containsSubstring(value, substring) {
							foundAll = false
							break
						}
					}
				}
			}

			if foundAll {
				matchingRows = append(matchingRows, rowIndex)
			}
		}
	})
	return matchingRows
}

// FindColsIfContains returns the indices of columns that contain the given element.
func (dt *DataTable) FindColsIfContains(value any) []string {
	var result []string
	dt.AtomicDo(func(dt *DataTable) {
		for colName, colPos := range dt.columnIndex {
			if dt.columns[colPos].FindFirst(value) != nil {
				result = append(result, colName)
			}
		}
	})
	return result
}

// FindColsIfContainsAll returns the indices of columns that contain all the given elements.
func (dt *DataTable) FindColsIfContainsAll(values ...any) []string {
	var result []string
	dt.AtomicDo(func(dt *DataTable) {
		for colName, colPos := range dt.columnIndex {
			foundAll := true

			for _, value := range values {
				if dt.columns[colPos].FindFirst(value) == nil {
					foundAll = false
					break
				}
			}

			if foundAll {
				result = append(result, colName)
			}
		}
	})
	return result
}

// FindColsIfAnyElementContainsSubstring returns the indices of columns that contain at least one element that contains the given substring.
func (dt *DataTable) FindColsIfAnyElementContainsSubstring(substring string) []string {
	var result []string
	dt.AtomicDo(func(dt *DataTable) {
		for colName, colPos := range dt.columnIndex {
			found := false

			for _, value := range dt.columns[colPos].data {
				if value != nil {
					if str, ok := value.(string); ok && containsSubstring(str, substring) {
						found = true
						break
					}
				}
			}

			if found {
				result = append(result, colName)
			}
		}
	})
	return result
}

// FindColsIfAllElementsContainSubstring returns the indices of columns that contain all elements that contain the given substring.
func (dt *DataTable) FindColsIfAllElementsContainSubstring(substring string) []string {
	var result []string
	dt.AtomicDo(func(dt *DataTable) {
		for colName, colPos := range dt.columnIndex {
			foundAll := true

			for _, value := range dt.columns[colPos].data {
				if value != nil {
					if str, ok := value.(string); ok && !containsSubstring(str, substring) {
						foundAll = false
						break
					}
				}
			}

			if foundAll {
				result = append(result, colName)
			}
		}
	})
	return result
}

// ======================== Drop ========================

// DropColsByName drops columns by their names.
func (dt *DataTable) DropColsByName(columnNames ...string) {
	dt.AtomicDo(func(dt *DataTable) {
		for _, name := range columnNames {
			for colName, colPos := range dt.columnIndex {
				if dt.columns[colPos].name == name {
					// 刪除對應的列
					dt.columns = append(dt.columns[:colPos], dt.columns[colPos+1:]...)
					delete(dt.columnIndex, colName)
					// 更新剩餘列的索引
					for i := colPos; i < len(dt.columns); i++ {
						newColName := generateColIndex(i)
						dt.columnIndex[newColName] = i
					}
					break
				}
			}
		}
		dt.regenerateColIndex()
		go dt.updateTimestamp()
	})
}

// DropColsByIndex drops columns by their index names.
func (dt *DataTable) DropColsByIndex(columnIndices ...string) {
	dt.AtomicDo(func(dt *DataTable) {
		for _, index := range columnIndices {
			index = strings.ToUpper(index)
			colPos, exists := dt.columnIndex[index]
			if exists {
				// 刪除對應的列
				dt.columns = append(dt.columns[:colPos], dt.columns[colPos+1:]...)
				delete(dt.columnIndex, index)
				// 更新剩餘列的索引
				for i := colPos; i < len(dt.columns); i++ {
					newColIndex := generateColIndex(i)
					dt.columnIndex[newColIndex] = i
				}
			}
		}

		dt.regenerateColIndex()
		go dt.updateTimestamp()
	})
}

// DropColsByNumber drops columns by their number.
func (dt *DataTable) DropColsByNumber(columnIndices ...int) {
	dt.AtomicDo(func(dt *DataTable) {
		// 從大到小排序，防止刪除後索引變動
		sort.Sort(sort.Reverse(sort.IntSlice(columnIndices)))

		for _, index := range columnIndices {
			if index >= 0 && index < len(dt.columns) {
				dt.columns = append(dt.columns[:index], dt.columns[index+1:]...)
				delete(dt.columnIndex, generateColIndex(index))
			}
		}

		dt.regenerateColIndex()
		go dt.updateTimestamp()
	})
}

// DropColsContainStringElements drops columns that contain string elements.
func (dt *DataTable) DropColsContainStringElements() {
	dt.AtomicDo(func(dt *DataTable) {
		columnsToDelete := make([]int, 0)

		// 找出包含字串元素的列索引
		for colIndex, column := range dt.columns {
			containsString := false

			for _, value := range column.data {
				if _, ok := value.(string); ok {
					containsString = true
					break
				}
			}

			if containsString {
				columnsToDelete = append(columnsToDelete, colIndex)
			}
		}

		// 反向刪除列，以避免索引錯誤
		for i := len(columnsToDelete) - 1; i >= 0; i-- {
			colIndex := columnsToDelete[i]
			dt.columns = append(dt.columns[:colIndex], dt.columns[colIndex+1:]...)
			delete(dt.columnIndex, generateColIndex(colIndex))
		}

		dt.regenerateColIndex()

		go dt.updateTimestamp()
	})
}

// DropColsContainNumbers drops columns that contain number elements.
func (dt *DataTable) DropColsContainNumbers() {
	dt.AtomicDo(func(dt *DataTable) {
		columnsToDelete := make([]int, 0)

		for colIndex, column := range dt.columns {
			containsNumber := false

			for _, value := range column.data {
				if _, isNumber := value.(int); isNumber {
					containsNumber = true
					break
				} else if _, isNumber := value.(float64); isNumber {
					containsNumber = true
					break
				}
			}

			if containsNumber {
				columnsToDelete = append(columnsToDelete, colIndex)
			}
		}

		for i := len(columnsToDelete) - 1; i >= 0; i-- {
			colIndex := columnsToDelete[i]
			dt.columns = append(dt.columns[:colIndex], dt.columns[colIndex+1:]...)
			delete(dt.columnIndex, generateColIndex(colIndex))
		}

		dt.regenerateColIndex()

		go dt.updateTimestamp()
	})
}

// DropColsContainNil drops columns that contain nil elements.
func (dt *DataTable) DropColsContainNil() {
	dt.AtomicDo(func(dt *DataTable) {
		columnsToDelete := make([]int, 0)

		for colIndex, column := range dt.columns {
			containsNil := false

			for _, value := range column.data {
				if value == nil {
					containsNil = true
					break
				}
			}

			if containsNil {
				columnsToDelete = append(columnsToDelete, colIndex)
			}
		}

		for i := len(columnsToDelete) - 1; i >= 0; i-- {
			colIndex := columnsToDelete[i]
			dt.columns = append(dt.columns[:colIndex], dt.columns[colIndex+1:]...)
			delete(dt.columnIndex, generateColIndex(colIndex))
		}

		dt.regenerateColIndex()

		go dt.updateTimestamp()
	})
}

// DropRowsByIndex drops rows by their indices.
func (dt *DataTable) DropRowsByIndex(rowIndices ...int) {
	dt.AtomicDo(func(dt *DataTable) {
		sort.Ints(rowIndices) // 確保從最小索引開始刪除

		for i, rowIndex := range rowIndices {
			if rowIndex < 0 {
				rowIndex = dt.getMaxColLength() + rowIndex
			}
			adjustedIndex := rowIndex - i // 因為每刪除一行，後續的行索引會變動
			for _, column := range dt.columns {
				if adjustedIndex >= 0 && adjustedIndex < len(column.data) {
					column.data = append(column.data[:adjustedIndex], column.data[adjustedIndex+1:]...)
				}
			}

			// 如果該行有名稱，也從 rowNames 中刪除
			for rowName, index := range dt.rowNames {
				if index == rowIndex {
					delete(dt.rowNames, rowName)
					break
				}
			}
		}
		go dt.updateTimestamp()
	})
}

// DropRowsByName drops rows by their names.
func (dt *DataTable) DropRowsByName(rowNames ...string) {
	dt.AtomicDo(func(dt *DataTable) {
		for _, rowName := range rowNames {
			rowIndex, exists := dt.rowNames[rowName]
			if !exists {
				LogWarning("DataTable", "DropRowsByName", "Row name '%s' does not exist", rowName)
				continue
			}

			// 移除所有列中對應行索引的資料
			for _, column := range dt.columns {
				if rowIndex < len(column.data) {
					column.data = append(column.data[:rowIndex], column.data[rowIndex+1:]...)
				}
			}

			// 移除行名索引
			delete(dt.rowNames, rowName)

			// 更新所有行名索引，以反映行被刪除後的變化
			for name, idx := range dt.rowNames {
				if idx > rowIndex {
					dt.rowNames[name] = idx - 1
				}
			}
		}

		go dt.updateTimestamp()
	})
}

// DropRowsContainStringElements drops rows that contain string elements.
func (dt *DataTable) DropRowsContainStringElements() {
	dt.AtomicDo(func(dt *DataTable) {
		rowsToDelete := make([]int, 0)

		// 找出包含字串元素的行索引
		for rowIndex := 0; rowIndex < dt.getMaxColLength(); rowIndex++ {
			containsString := false

			for _, col := range dt.columns {
				if rowIndex < len(col.data) {
					if _, ok := col.data[rowIndex].(string); ok {
						containsString = true
						break
					}
				}
			}

			if containsString {
				rowsToDelete = append(rowsToDelete, rowIndex)
			}
		}

		// 反向刪除行，以避免索引錯誤
		for i := len(rowsToDelete) - 1; i >= 0; i-- {
			rowIndex := rowsToDelete[i]
			for _, col := range dt.columns {
				if rowIndex < len(col.data) {
					col.data = append(col.data[:rowIndex], col.data[rowIndex+1:]...)
				}
			}

			// 刪除行名對應
			for rowName, idx := range dt.rowNames {
				if idx == rowIndex {
					delete(dt.rowNames, rowName)
				} else if idx > rowIndex {
					dt.rowNames[rowName] = idx - 1
				}
			}
		}
		go dt.updateTimestamp()
	})
}

// DropRowsContainNumbers drops rows that contain number elements.
func (dt *DataTable) DropRowsContainNumbers() {
	dt.AtomicDo(func(dt *DataTable) {
		maxLength := dt.getMaxColLength()
		rowsToKeep := make([]bool, maxLength)

		for rowIndex := 0; rowIndex < maxLength; rowIndex++ {
			keepRow := true
			for _, column := range dt.columns {
				if rowIndex < len(column.data) {
					if _, isNumber := column.data[rowIndex].(int); isNumber {
						keepRow = false
						break
					} else if _, isNumber := column.data[rowIndex].(float64); isNumber {
						keepRow = false
						break
					}
				}
			}
			rowsToKeep[rowIndex] = keepRow
		}

		for i := len(rowsToKeep) - 1; i >= 0; i-- {
			if !rowsToKeep[i] {
				for _, column := range dt.columns {
					if i < len(column.data) {
						column.data = append(column.data[:i], column.data[i+1:]...)
					}
				}
			}
		}

		// 更新 rowNames 索引
		newRowNames := make(map[string]int)
		newIndex := 0
		for name, oldIndex := range dt.rowNames {
			if rowsToKeep[oldIndex] {
				newRowNames[name] = newIndex
				newIndex++
			}
		}
		dt.rowNames = newRowNames

		go dt.updateTimestamp()
	})
}

// DropRowsContainNil drops rows that contain nil elements.
func (dt *DataTable) DropRowsContainNil() {
	dt.AtomicDo(func(dt *DataTable) {
		maxLength := dt.getMaxColLength()

		// 這個切片將存儲所有非nil的行的索引
		nonNilRowIndices := []int{}

		// 遍歷每一行
		for rowIndex := 0; rowIndex < maxLength; rowIndex++ {
			rowHasNil := false

			// 檢查該行是否包含 nil
			for _, column := range dt.columns {
				if rowIndex < len(column.data) && column.data[rowIndex] == nil {
					rowHasNil = true
					break
				}
			}

			// 如果該行不包含 nil，將其索引加入到 nonNilRowIndices 中
			if !rowHasNil {
				nonNilRowIndices = append(nonNilRowIndices, rowIndex)
			}
		}

		// 建立新的列資料，僅保留非nil的行
		for _, column := range dt.columns {
			newData := []any{}
			for _, rowIndex := range nonNilRowIndices {
				if rowIndex < len(column.data) {
					newData = append(newData, column.data[rowIndex])
				}
			}
			column.data = newData
		}

		// 更新 rowNames 映射，以移除被刪除的行
		for rowName, rowIndex := range dt.rowNames {
			if rowIndex >= len(nonNilRowIndices) || rowIndex != nonNilRowIndices[rowIndex] {
				delete(dt.rowNames, rowName)
			} else {
				dt.rowNames[rowName] = rowIndex
			}
		}
		go dt.updateTimestamp()
	})
}

// ======================== Data ========================

func (dt *DataTable) Data(useNamesAsKeys ...bool) map[string][]any {
	var result map[string][]any
	dt.AtomicDo(func(dt *DataTable) {
		dataMap := make(map[string][]any)

		useNamesAsKeysBool := true
		if len(useNamesAsKeys) == 1 {
			useNamesAsKeysBool = useNamesAsKeys[0]
		}
		if len(useNamesAsKeys) > 1 {
			LogWarning("DataTable", "Data", "Too many arguments, returning empty map")
			result = dataMap
			return
		}

		for i, col := range dt.columns {
			var key string
			if useNamesAsKeysBool && col.name != "" {
				key = col.name
			} else {
				key = generateColIndex(i)
			}
			dataMap[key] = col.data
		}

		result = dataMap
	})
	return result
}

// ======================== Statistics ========================

// Count returns the number of occurrences of the given value in the DataTable.
func (dt *DataTable) Count(value any) int {
	result := asyncutil.ParallelForEach(dt.columns, func(i int, column any) int {
		return dt.columns[i].Count(value)
	})
	count := NewDataList(result).Sum()
	return conv.ParseInt(count)
}

// Counter returns the number of occurrences of the given value in the DataTable.
// Return a map[any]int
func (dt *DataTable) Counter() map[any]int {
	var result map[any]int
	dt.AtomicDo(func(dt *DataTable) {
		result = make(map[any]int)
		for _, column := range dt.columns {
			for _, value := range column.data {
				result[value] += 1
			}
		}
	})
	return result
}

// Size returns the number of rows and columns in the DataTable.
func (dt *DataTable) Size() (numRows int, numCols int) {
	var rows, cols int
	dt.AtomicDo(func(dt *DataTable) {
		rows = dt.getMaxColLength()
		cols = len(dt.columns)
	})
	return rows, cols
}

// Mean returns the mean of the DataTable.
func (dt *DataTable) Mean() any {
	var result any
	dt.AtomicDo(func(dt *DataTable) {
		var totalSum float64
		rowNum, colNum := dt.getMaxColLength(), len(dt.columns)
		totalCount := rowNum * colNum
		for _, column := range dt.columns {
			totalSum += column.Sum()
		}
		result = totalSum / float64(totalCount)
	})
	return result
}

// ======================== Conversion ========================

// Transpose transposes the DataTable, converting rows into columns and vice versa.
func (dt *DataTable) Transpose() *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		dls := make([]*DataList, 0)
		dls = append(dls, dt.columns...)

		oldRowNames := dt.rowNames
		dt.rowNames = make(map[string]int)
		newDt := &DataTable{
			columns:           make([]*DataList, 0),
			rowNames:          make(map[string]int),
			columnIndex:       make(map[string]int),
			creationTimestamp: dt.GetCreationTimestamp(),
		}

		newDt.lastModifiedTimestamp.Store(dt.GetLastModifiedTimestamp())

		for i, col := range dls {

			newDt.AppendRowsFromDataList(col)
			for rowName, rowIndex := range oldRowNames {
				if rowIndex == i {
					newDt.columns[i].name = rowName
					newDt.columnIndex[generateColIndex(i)] = i
				}
			}
		}

		dt.columns = newDt.columns
		dt.rowNames = newDt.rowNames
		dt.columnIndex = newDt.columnIndex

		go dt.updateTimestamp()
		result = dt
	})
	return result
}

// ======================== Utilities ========================

func (dt *DataTable) getRowNameByIndex(index int) (string, bool) {
	for rowName, rowIndex := range dt.rowNames {
		if rowIndex == index {
			return rowName, true
		}
	}
	return "", false
}

func (dt *DataTable) getMaxColLength() int {
	maxLength := 0
	for _, col := range dt.columns {
		if len(col.data) > maxLength {
			maxLength = len(col.data)
		}
	}
	return maxLength
}

func (dt *DataTable) regenerateColIndex() {
	dt.columnIndex = make(map[string]int)
	for i := range dt.columns {
		dt.columnIndex[generateColIndex(i)] = i
	}
}

// 新增一個方法來根據字母順序重新排序 columns 及更新 columnIndex
func (dt *DataTable) sortColsByIndex() {
	// 取得所有欄位名稱並排序
	keys := make([]string, 0, len(dt.columnIndex))
	for key := range dt.columnIndex {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// 根據排序的欄位名稱重建 columns 和 columnIndex
	newCols := make([]*DataList, len(keys))
	for i, key := range keys {
		newCols[i] = dt.columns[dt.columnIndex[key]]
		dt.columnIndex[key] = i // 更新對應的 index
	}
	dt.columns = newCols
}

func generateColIndex(index int) string {
	name := ""
	for index >= 0 {
		name = fmt.Sprintf("%c%s", 'A'+(index%26), name)
		index = index/26 - 1
	}
	return name
}

func newEmptyDataList(rowCount int) *DataList {
	data := make([]any, rowCount)
	for i := 0; i < rowCount; i++ {
		data[i] = nil
	}

	now := time.Now().Unix()
	dl := &DataList{
		data:              data,
		creationTimestamp: now,
	}
	dl.lastModifiedTimestamp.Store(now)

	return dl
}

func safeRowName(dt *DataTable, name string) string {
	if name == "" {
		return ""
	}

	originalName := name
	counter := 1

	for {
		// 檢查是否已經存在該行名
		if _, exists := dt.rowNames[name]; !exists {
			break // 如果行名不存在，跳出循環
		}

		// 如果行名存在，則生成新的行名並繼續檢查
		name = fmt.Sprintf("%s_%d", originalName, counter)
		counter++
	}

	return name
}

func safeColName(dt *DataTable, name string) string {
	if name == "" {
		return ""
	}

	originalName := name
	counter := 1

	for {
		// 檢查是否已經存在該列名
		found := false
		for _, col := range dt.columns {
			if col.name == name {
				found = true
			}
		}

		if !found {
			break // 如果列名不存在，跳出循環
		}

		// 如果列名存在，則生成新的列名並繼續檢查
		name = fmt.Sprintf("%s_%d", originalName, counter)
		counter++
	}

	return name
}

// containsSubstring 是一個輔助函數，用來檢查一個字符串是否包含子字符串
func containsSubstring(value string, substring string) bool {
	return len(value) >= len(substring) && (value == substring || len(value) > len(substring) && (value[:len(substring)] == substring || containsSubstring(value[1:], substring)))
}

func (dt *DataTable) updateTimestamp() {
	now := time.Now().Unix()
	oldTimestamp := dt.lastModifiedTimestamp.Load()
	if oldTimestamp < now {
		dt.lastModifiedTimestamp.Store(now)
	}
}

func (dt *DataTable) GetCreationTimestamp() int64 {
	return dt.creationTimestamp
}

func (dt *DataTable) GetLastModifiedTimestamp() int64 {
	return dt.lastModifiedTimestamp.Load()
}

// Clone creates a deep copy of the DataTable.
// It copies all columns, column indices, row names, and metadata,
// ensuring that modifications to the original DataTable do not affect the clone.
// The cloned DataTable has a new creation timestamp and is fully independent.
func (dt *DataTable) Clone() *DataTable {
	var newDT *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		// Clone columns
		clonedColumns := make([]*DataList, len(dt.columns))
		for i, col := range dt.columns {
			clonedColumns[i] = col.Clone()
		}

		// Clone columnIndex map
		clonedColumnIndex := make(map[string]int)
		maps.Copy(clonedColumnIndex, dt.columnIndex)

		// Clone rowNames map
		clonedRowNames := make(map[string]int)
		maps.Copy(clonedRowNames, dt.rowNames)

		// Create new DataTable with cloned data
		now := time.Now().Unix()
		newDT = &DataTable{
			columns:           clonedColumns,
			columnIndex:       clonedColumnIndex,
			rowNames:          clonedRowNames,
			name:              dt.name,
			creationTimestamp: time.Now().Unix(),
		}
		newDT.lastModifiedTimestamp.Store(now)

		// Initialize atomic fields
		newDT.cmdCh = make(chan func())
		newDT.initOnce = sync.Once{}
		newDT.closed = atomic.Bool{}
	})
	return newDT
}
