package insyra

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/HazelnutParadise/Go-Utils/asyncutil"
	"github.com/HazelnutParadise/Go-Utils/conv"
)

type DataTable struct {
	mu                    sync.Mutex
	columns               []*DataList
	columnIndex           map[string]int // 儲存字母索引與切片中的索引對應
	rowNames              map[string]int
	creationTimestamp     int64
	lastModifiedTimestamp int64
}

type IDataTable interface {
	AppendColumns(columns ...*DataList)
	AppendRowsFromDataList(rowsData ...*DataList)
	AppendRowsByIndex(rowsData ...map[string]interface{})
	AppendRowsByName(rowsData ...map[string]interface{})
	GetElement(rowIndex int, columnIndex string) interface{}
	GetColumn(index string) *DataList
	GetColumnByNumber(index int) *DataList
	GetRow(index int) *DataList
	UpdateElement(rowIndex int, columnIndex string, value interface{})
	UpdateColumn(index string, dl *DataList)
	UpdateRow(index int, dl *DataList)
	FindRowsIfContains(value interface{}) []int
	FindRowsIfContainsAll(values ...interface{}) []int
	FindRowsIfAnyElementContainsSubstring(substring string) []int
	FindRowsIfAllElementsContainSubstring(substring string) []int
	FindColumnsIfContains(value interface{}) []string
	FindColumnsIfContainsAll(values ...interface{}) []string
	FindColumnsIfAnyElementContainsSubstring(substring string) []string
	FindColumnsIfAllElementsContainSubstring(substring string) []string
	DropColumnsByName(columnNames ...string)
	DropColumnsByIndex(columnIndices ...string)
	DropColumnsContainStringElements()
	DropColumnsContainNumbers()
	DropColumnsContainNil()
	DropRowsByIndex(rowIndices ...int)
	DropRowsByName(rowNames ...string)
	DropRowsContainStringElements()
	DropRowsContainNumbers()
	DropRowsContainNil()
	Data(useNamesAsKeys ...bool) map[string][]interface{}
	Show()
	ShowTypes()
	GetRowNameByIndex(index int) string
	SetRowNameByIndex(index int, name string)
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	getSortedColumnNames() []string
	getRowNameByIndex(index int) (string, bool)
	getMaxColumnLength() int
	updateTimestamp()

	// Statistics
	Size() (int, int)
	Count(value interface{}) int

	// Filters
	Filter(filterFunc FilterFunc) *DataTable
	FilterByCustomElement(f func(value interface{}) bool) *DataTable
	FilterByColumnIndexGreaterThan(threshold string) *DataTable
	FilterByColumnIndexGreaterThanOrEqualTo(threshold string) *DataTable
	FilterByColumnIndexLessThan(threshold string) *DataTable
	FilterByColumnIndexLessThanOrEqualTo(threshold string) *DataTable
	FilterByColumnIndexEqualTo(index string) *DataTable
	FilterByColumnNameEqualTo(name string) *DataTable
	FilterByColumnNameContains(substring string) *DataTable
	FilterByRowNameEqualTo(name string) *DataTable
	FilterByRowNameContains(substring string) *DataTable
	FilterByRowIndexGreaterThan(threshold int) *DataTable
	FilterByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable
	FilterByRowIndexLessThan(threshold int) *DataTable
	FilterByRowIndexLessThanOrEqualTo(threshold int) *DataTable
	FilterByRowIndexEqualTo(index int) *DataTable

	// CSV
	ToCSV(filePath string, setRowNamesToFirstColumn bool, setColumnNamesToFirstRow bool) error
	LoadFromCSV(filePath string, setFirstColumnToRowNames bool, setFirstRowToColumnNames bool) error
}

func NewDataTable(columns ...*DataList) *DataTable {
	newTable := &DataTable{
		columns:               []*DataList{},
		columnIndex:           make(map[string]int),
		rowNames:              make(map[string]int),
		creationTimestamp:     time.Now().Unix(),
		lastModifiedTimestamp: time.Now().Unix(),
	}

	if len(columns) > 0 {
		newTable.AppendColumns(columns...)
	}

	return newTable
}

// ======================== Append ========================

// AppendColumns appends columns to the DataTable, with each column represented by a DataList.
// If the columns are shorter than the existing columns, nil values will be appended to match the length.
// If the columns are longer than the existing columns, the existing columns will be extended with nil values.
func (dt *DataTable) AppendColumns(columns ...*DataList) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	maxLength := dt.getMaxColumnLength()

	for _, column := range columns {
		columnName := generateColumnIndex(len(dt.columns)) // 修改這行確保按順序生成列名
		column.name = safeColumnName(dt, column.name)

		dt.columns = append(dt.columns, column)
		dt.columnIndex[columnName] = len(dt.columns) - 1
		if len(column.data) < maxLength {
			column.data = append(column.data, make([]interface{}, maxLength-len(column.data))...)
		}
		LogDebug(fmt.Sprintf("AppendColumns: Added column %s at index %d", columnName, dt.columnIndex[columnName]))
	}

	for _, col := range dt.columns {
		if len(col.data) < maxLength {
			col.data = append(col.data, make([]interface{}, maxLength-len(col.data))...)
		}
	}
}

// AppendRowsFromDataList appends rows to the DataTable, with each row represented by a DataList.
// If the rows are shorter than the existing columns, nil values will be appended to match the length.
// If the rows are longer than the existing columns, the existing columns will be extended with nil values.
func (dt *DataTable) AppendRowsFromDataList(rowsData ...*DataList) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		if rowData.name != "" {
			srn := safeRowName(dt, rowData.name)
			dt.rowNames[srn] = maxLength
		}

		if len(rowData.data) > len(dt.columns) {
			for i := len(dt.columns); i < len(rowData.data); i++ {
				newCol := newEmptyDataList(maxLength)
				columnName := generateColumnIndex(i)
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
}

// AppendRowsByIndex appends rows to the DataTable, with each row represented by a map of column index and value.
// If the rows are shorter than the existing columns, nil values will be appended to match the length.
// If the rows are longer than the existing columns, the existing columns will be extended with nil values.
func (dt *DataTable) AppendRowsByIndex(rowsData ...map[string]interface{}) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		for colIndex, value := range rowData {
			colPos, exists := dt.columnIndex[colIndex]
			LogDebug(fmt.Sprintf("AppendRowsByIndex: Handling column %s, exists: %t", colIndex, exists))
			if !exists {
				newCol := newEmptyDataList(maxLength)
				dt.columns = append(dt.columns, newCol)
				colPos = len(dt.columns) - 1
				dt.columnIndex[colIndex] = colPos // 更新 columnIndex
				LogDebug(fmt.Sprintf("AppendRowsByIndex: Added new column %s at index %d", colIndex, colPos))
			}
			dt.columns[colPos].data = append(dt.columns[colPos].data, value)
		}

		for _, column := range dt.columns {
			if len(column.data) <= maxLength {
				column.data = append(column.data, nil)
			}
		}
	}
}

// AppendRowsByName appends rows to the DataTable, with each row represented by a map of column name and value.
// If the rows are shorter than the existing columns, nil values will be appended to match the length.
// If the rows are longer than the existing columns, the existing columns will be extended with nil values.
func (dt *DataTable) AppendRowsByName(rowsData ...map[string]interface{}) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		for colName, value := range rowData {
			found := false
			for i := 0; i < len(dt.columns); i++ {
				if dt.columns[i].name == colName {
					dt.columns[i].data = append(dt.columns[i].data, value)
					found = true
					LogDebug(fmt.Sprintf("AppendRowsByName: Found column %s at index %d", colName, i))
					break
				}
			}
			if !found {
				newCol := newEmptyDataList(maxLength)
				newCol.name = colName
				newCol.data = append(newCol.data, value)
				dt.columns = append(dt.columns, newCol)
				dt.columnIndex[generateColumnIndex(len(dt.columns)-1)] = len(dt.columns) - 1 // 更新 columnIndex
				LogDebug(fmt.Sprintf("AppendRowsByName: Added new column %s at index %d", colName, len(dt.columns)-1))
			}
		}

		for _, column := range dt.columns {
			if len(column.data) == maxLength {
				column.data = append(column.data, nil)
			}
		}
	}
}

// ======================== Get ========================

// GetElement returns the element at the given row and column index.
func (dt *DataTable) GetElement(rowIndex int, columnIndex string) interface{} {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if colPos, exists := dt.columnIndex[columnIndex]; exists {
		if rowIndex < 0 {
			rowIndex = len(dt.columns[colPos].data) + rowIndex
		}
		if rowIndex < 0 || rowIndex >= len(dt.columns[colPos].data) {
			LogWarning("DataTable.GetElement(): Row index is out of range, returning nil.")
			return nil
		}
		return dt.columns[colPos].data[rowIndex]
	}
	return nil

}

// GetColumn returns a new DataList containing the data of the column with the given index.
func (dt *DataTable) GetColumn(index string) *DataList {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if colPos, exists := dt.columnIndex[index]; exists {
		// 初始化新的 DataList 並分配 data 切片的大小
		dl := NewDataList()
		dl.data = make([]interface{}, len(dt.columns[colPos].data))

		// 拷貝數據到新的 DataList
		copy(dl.data, dt.columns[colPos].data)
		dl.name = dt.columns[colPos].name
		return dl
	}
	return nil
}

func (dt *DataTable) GetColumnByNumber(index int) *DataList {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if index < 0 {
		index = len(dt.columns) + index
	}

	if index < 0 || index >= len(dt.columns) {
		LogWarning("DataTable.GetColumnByNumber(): Column index is out of range, returning nil.")
		return nil
	}

	// 初始化新的 DataList 並分配 data 切片的大小
	dl := NewDataList()
	dl.data = make([]interface{}, len(dt.columns[index].data))

	// 拷貝數據到新的 DataList
	copy(dl.data, dt.columns[index].data)
	dl.name = dt.columns[index].name
	return dl
}

// GetRow returns a new DataList containing the data of the row with the given index.
func (dt *DataTable) GetRow(index int) *DataList {
	dt.mu.Lock()
	if index < 0 {
		index = dt.getMaxColumnLength() + index
	}
	if index < 0 || index >= dt.getMaxColumnLength() {
		LogWarning("DataTable.GetRow(): Row index is out of range, returning nil.")
		return nil
	}

	// 初始化新的 DataList 並分配 data 切片的大小
	dl := NewDataList()
	dl.data = make([]interface{}, len(dt.columns))

	// 拷貝數據到新的 DataList
	for i, column := range dt.columns {
		if index < len(column.data) {
			dl.data[i] = column.data[index]
		}
	}
	dt.mu.Unlock()
	dl.name = dt.GetRowNameByIndex(index)
	return dl
}

// ======================== Update ========================

// UpdateElement updates the element at the given row and column index.
func (dt *DataTable) UpdateElement(rowIndex int, columnIndex string, value interface{}) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	if colPos, exists := dt.columnIndex[columnIndex]; exists {
		if rowIndex < 0 {
			rowIndex = len(dt.columns[colPos].data) + rowIndex
		}
		if rowIndex < 0 || rowIndex >= len(dt.columns[colPos].data) {
			LogWarning("DataTable.UpdateElement(): Row index is out of range, returning.")
			return
		}
		dt.columns[colPos].data[rowIndex] = value
	} else {
		LogWarning("DataTable.UpdateElement(): Column index does not exist, returning.")
	}
}

// UpdateColumn updates the column with the given index.
func (dt *DataTable) UpdateColumn(index string, dl *DataList) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	if colPos, exists := dt.columnIndex[index]; exists {
		dt.columns[colPos] = dl
	} else {
		LogWarning("DataTable.UpdateColumn(): Column index does not exist, returning.")
	}
}

// UpdateRow updates the row at the given index.
func (dt *DataTable) UpdateRow(index int, dl *DataList) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if index < 0 || index >= dt.getMaxColumnLength() {
		LogWarning("DataTable.UpdateRow(): Index out of bounds")
		return
	}

	if len(dl.data) > len(dt.columns) {
		LogWarning("DataTable.UpdateRow(): DataList has more elements than DataTable columns, returning.")
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
}

// ======================== Find ========================

// FindRowsIfContains returns the indices of rows that contain the given element.
func (dt *DataTable) FindRowsIfContains(value interface{}) []int {
	dt.mu.Lock()
	defer dt.mu.Unlock()

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
	var result []int
	for index := range indexMap {
		result = append(result, index)
	}

	// 排序結果以保證順序
	sort.Ints(result)

	return result
}

// FindRowsIfContainsAll returns the indices of rows that contain all the given elements.
func (dt *DataTable) FindRowsIfContainsAll(values ...interface{}) []int {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	var result []int

	// 檢查每一行是否包含所有指定的值
	for rowIndex := 0; rowIndex < dt.getMaxColumnLength(); rowIndex++ {
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

	return result
}

// FindRowsIfAnyElementContainsSubstring returns the indices of rows that contain at least one element that contains the given substring.
func (dt *DataTable) FindRowsIfAnyElementContainsSubstring(substring string) []int {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	var matchingRows []int

	for rowIndex := 0; rowIndex < dt.getMaxColumnLength(); rowIndex++ {
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

	return matchingRows
}

// FindRowsIfAllElementsContainSubstring returns the indices of rows that contain all elements that contain the given substring.
func (dt *DataTable) FindRowsIfAllElementsContainSubstring(substring string) []int {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	var matchingRows []int

	for rowIndex := 0; rowIndex < dt.getMaxColumnLength(); rowIndex++ {
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

	return matchingRows
}

// FindColumnsIfContains returns the indices of columns that contain the given element.
func (dt *DataTable) FindColumnsIfContains(value interface{}) []string {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	var result []string

	for colName, colPos := range dt.columnIndex {
		if dt.columns[colPos].FindFirst(value) != nil {
			result = append(result, colName)
		}
	}

	return result
}

// FindColumnsIfContainsAll returns the indices of columns that contain all the given elements.
func (dt *DataTable) FindColumnsIfContainsAll(values ...interface{}) []string {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	var result []string

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

	return result
}

// FindColumnsIfAnyElementContainsSubstring returns the indices of columns that contain at least one element that contains the given substring.
func (dt *DataTable) FindColumnsIfAnyElementContainsSubstring(substring string) []string {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	var result []string

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

	return result
}

// FindColumnsIfAllElementsContainSubstring returns the indices of columns that contain all elements that contain the given substring.
func (dt *DataTable) FindColumnsIfAllElementsContainSubstring(substring string) []string {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	var result []string

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

	return result
}

// ======================== Drop ========================

// DropColumnsByName drops columns by their names.
func (dt *DataTable) DropColumnsByName(columnNames ...string) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, name := range columnNames {
		for colName, colPos := range dt.columnIndex {
			if dt.columns[colPos].name == name {
				// 刪除對應的列
				dt.columns = append(dt.columns[:colPos], dt.columns[colPos+1:]...)
				delete(dt.columnIndex, colName)
				// 更新剩餘列的索引
				for i := colPos; i < len(dt.columns); i++ {
					newColName := generateColumnIndex(i)
					dt.columnIndex[newColName] = i
				}
				break
			}
		}
	}
}

// DropColumnsByIndex drops columns by their index names.
func (dt *DataTable) DropColumnsByIndex(columnIndices ...string) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, index := range columnIndices {
		colPos, exists := dt.columnIndex[index]
		if exists {
			// 刪除對應的列
			dt.columns = append(dt.columns[:colPos], dt.columns[colPos+1:]...)
			delete(dt.columnIndex, index)
			// 更新剩餘列的索引
			for i := colPos; i < len(dt.columns); i++ {
				newColName := generateColumnIndex(i)
				dt.columnIndex[newColName] = i
			}
		}
	}
}

// DropColumnsContainStringElements drops columns that contain string elements.
func (dt *DataTable) DropColumnsContainStringElements() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

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
		delete(dt.columnIndex, generateColumnIndex(colIndex))
	}

	// 更新 columnIndex 映射，以反映列被刪除後的變化
	for i, _ := range dt.columns {
		newColName := generateColumnIndex(i)
		dt.columnIndex[newColName] = i
	}

	go dt.updateTimestamp()
}

// DropColumnsContainNumbers drops columns that contain number elements.
func (dt *DataTable) DropColumnsContainNumbers() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

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
		delete(dt.columnIndex, generateColumnIndex(colIndex))
	}

	for i, _ := range dt.columns {
		newColName := generateColumnIndex(i)
		dt.columnIndex[newColName] = i
	}

	go dt.updateTimestamp()
}

// DropColumnsContainNil drops columns that contain nil elements.
func (dt *DataTable) DropColumnsContainNil() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

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
		delete(dt.columnIndex, generateColumnIndex(colIndex))
	}

	for i, _ := range dt.columns {
		newColName := generateColumnIndex(i)
		dt.columnIndex[newColName] = i
	}

	go dt.updateTimestamp()
}

// DropRowsByIndex drops rows by their indices.
func (dt *DataTable) DropRowsByIndex(rowIndices ...int) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	sort.Ints(rowIndices) // 確保從最小索引開始刪除

	for i, rowIndex := range rowIndices {
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
}

// DropRowsByName drops rows by their names.
func (dt *DataTable) DropRowsByName(rowNames ...string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	for _, rowName := range rowNames {
		rowIndex, exists := dt.rowNames[rowName]
		if !exists {
			LogWarning(fmt.Sprintf("Row name '%s' does not exist.", rowName))
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
}

// DropRowsContainStringElements drops rows that contain string elements.
func (dt *DataTable) DropRowsContainStringElements() {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	rowsToDelete := make([]int, 0)

	// 找出包含字串元素的行索引
	for rowIndex := 0; rowIndex < dt.getMaxColumnLength(); rowIndex++ {
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
}

// DropRowsContainNumbers drops rows that contain number elements.
func (dt *DataTable) DropRowsContainNumbers() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	maxLength := dt.getMaxColumnLength()
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
}

// DropRowsContainNil drops rows that contain nil elements.
func (dt *DataTable) DropRowsContainNil() {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	maxLength := dt.getMaxColumnLength()

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
		newData := []interface{}{}
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
}

// ======================== Data ========================

func (dt *DataTable) Data(useNamesAsKeys ...bool) map[string][]interface{} {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dataMap := make(map[string][]interface{})

	useNamesAsKeysBool := true
	if len(useNamesAsKeys) == 1 {
		useNamesAsKeysBool = useNamesAsKeys[0]
	}
	if len(useNamesAsKeys) > 1 {
		LogWarning("DataTable.Data(): too many arguments, returning empty map.")
		return dataMap
	}

	for i, col := range dt.columns {
		var key string
		if useNamesAsKeysBool && col.name != "" {
			key = fmt.Sprintf("%s(%s)", generateColumnIndex(i), col.name)
		} else {
			key = generateColumnIndex(i)
		}
		dataMap[key] = col.data
	}

	return dataMap
}

// ======================== Show ========================

func (dt *DataTable) Show() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// 構建資料地圖，但不使用 Data() 方法以避免死鎖
	dataMap := make(map[string][]interface{})
	for i, col := range dt.columns {
		key := generateColumnIndex(i)
		if col.name != "" {
			key += fmt.Sprintf("(%s)", col.name)
		}
		dataMap[key] = col.data
	}

	// 取得所有的列索引並排序
	var colIndices []string
	for colIndex := range dataMap {
		colIndices = append(colIndices, colIndex)
	}
	sort.Strings(colIndices)

	// 計算每一列的最大寬度
	colWidths := make(map[string]int)
	for _, colIndex := range colIndices {
		colWidths[colIndex] = len(colIndex)
		for _, value := range dataMap[colIndex] {
			valueStr := fmt.Sprintf("%v", value)
			if len(valueStr) > colWidths[colIndex] {
				colWidths[colIndex] = len(valueStr)
			}
		}
	}

	// 計算 RowNames 的最大寬度，並顯示 RowIndex
	rowNames := make([]string, dt.getMaxColumnLength())
	maxRowNameWidth := len("RowNames")
	for i := range rowNames {
		if rowName, exists := dt.getRowNameByIndex(i); exists {
			rowNames[i] = rowName
		} else {
			rowNames[i] = "" // 如果沒有名字則顯示為空
		}
		rowNames[i] = fmt.Sprintf("%d: %s", i, rowNames[i]) // 加上 RowIndex
		if len(rowNames[i]) > maxRowNameWidth {
			maxRowNameWidth = len(rowNames[i])
		}
	}

	// 打印列名
	fmt.Printf("%-*s", maxRowNameWidth+2, "RowNames") // +2 是為了讓其更清晰
	for _, colIndex := range colIndices {
		fmt.Printf("%-*s", colWidths[colIndex]+2, colIndex)
	}
	fmt.Println()

	// 打印行資料
	for rowIndex := 0; rowIndex < dt.getMaxColumnLength(); rowIndex++ {
		fmt.Printf("%-*s", maxRowNameWidth+2, rowNames[rowIndex])

		for _, colIndex := range colIndices {
			value := "nil"
			if rowIndex < len(dataMap[colIndex]) && dataMap[colIndex][rowIndex] != nil {
				value = fmt.Sprintf("%v", dataMap[colIndex][rowIndex])
			}
			fmt.Printf("%-*s", colWidths[colIndex]+2, value)
		}
		fmt.Println()
	}
}

func (dt *DataTable) ShowTypes() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// 構建資料地圖，但不使用 Data() 方法以避免死鎖
	dataMap := make(map[string][]interface{})
	for i, col := range dt.columns {
		key := generateColumnIndex(i)
		if col.name != "" {
			key += fmt.Sprintf("(%s)", col.name)
		}
		dataMap[key] = col.data
	}

	// 取得所有的列索引並排序
	var colIndices []string
	for colIndex := range dataMap {
		colIndices = append(colIndices, colIndex)
	}
	sort.Strings(colIndices)

	// 計算每一列的最大寬度
	colWidths := make(map[string]int)
	for _, colIndex := range colIndices {
		colWidths[colIndex] = len(colIndex)
		for _, value := range dataMap[colIndex] {
			valueStr := fmt.Sprintf("%v", value)
			if len(valueStr) > colWidths[colIndex] {
				colWidths[colIndex] = len(valueStr)
			}
		}
	}

	// 計算 RowNames 的最大寬度，並顯示 RowIndex
	rowNames := make([]string, dt.getMaxColumnLength())
	maxRowNameWidth := len("RowNames")
	for i := range rowNames {
		if rowName, exists := dt.getRowNameByIndex(i); exists {
			rowNames[i] = rowName
		} else {
			rowNames[i] = "" // 如果沒有名字則顯示為空
		}
		rowNames[i] = fmt.Sprintf("%d: %s", i, rowNames[i]) // 加上 RowIndex
		if len(rowNames[i]) > maxRowNameWidth {
			maxRowNameWidth = len(rowNames[i])
		}
	}

	// 打印列名
	fmt.Printf("%-*s", maxRowNameWidth+2, "RowNames") // +2 是為了讓其更清晰
	for _, colIndex := range colIndices {
		fmt.Printf("%-*s", colWidths[colIndex]+8, colIndex)
	}
	fmt.Println()

	// 打印行資料
	for rowIndex := 0; rowIndex < dt.getMaxColumnLength(); rowIndex++ {
		fmt.Printf("%-*s", maxRowNameWidth+2, rowNames[rowIndex])

		for _, colIndex := range colIndices {
			value := "nil"
			if rowIndex < len(dataMap[colIndex]) && dataMap[colIndex][rowIndex] != nil {
				value = fmt.Sprintf("%T", dataMap[colIndex][rowIndex])
			}
			fmt.Printf("%-*s", colWidths[colIndex]+8, value)
		}
		fmt.Println()
	}
}

// ======================== RowName ========================

// GetRowNameByIndex returns the name of the row at the given index.
func (dt *DataTable) GetRowNameByIndex(index int) string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if rowName, exists := dt.getRowNameByIndex(index); exists {
		return rowName
	} else {
		// LogWarning("DataTable.GetRowNameByIndex(): Row index %d does not have a name.", index)
		return ""
	}
}

func (dt *DataTable) SetRowNameByIndex(index int, name string) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
	}()
	originalIndex := index
	if index < 0 {
		index = dt.getMaxColumnLength() + index
	}
	if index < 0 || index >= dt.getMaxColumnLength() {
		LogWarning("DataTable.SetRowNameByIndex(): Row index %d is out of range, returning.", originalIndex)
		return
	}
	srn := safeRowName(dt, name)
	dt.rowNames[srn] = index
	go dt.updateTimestamp()
}

// ======================== Statistics ========================

// Count returns the number of occurrences of the given value in the DataTable.
func (dt *DataTable) Count(value interface{}) int {
	result := asyncutil.ParallelForEach(dt.columns, func(i int, column interface{}) int {
		return dt.columns[i].Count(value)
	})
	count := NewDataList(result).Sum()
	return conv.ParseInt(count)
}

// Size returns the number of rows and columns in the DataTable.
func (dt *DataTable) Size() (int, int) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.getMaxColumnLength(), len(dt.columns)
}

// ======================== Utilities ========================

func (dt *DataTable) getSortedColumnNames() []string {
	colNames := make([]string, 0, len(dt.columnIndex))
	for colName := range dt.columnIndex {
		colNames = append(colNames, colName)
	}
	sort.Strings(colNames)
	return colNames
}

func (dt *DataTable) getRowNameByIndex(index int) (string, bool) {
	for rowName, rowIndex := range dt.rowNames {
		if rowIndex == index {
			return rowName, true
		}
	}
	return "", false
}

func (dt *DataTable) getMaxColumnLength() int {
	maxLength := 0
	for _, col := range dt.columns {
		if len(col.data) > maxLength {
			maxLength = len(col.data)
		}
	}
	return maxLength
}

func generateColumnIndex(index int) string {
	name := ""
	for index >= 0 {
		name = string('A'+(index%26)) + name
		index = index/26 - 1
	}
	return name
}

func newEmptyDataList(rowCount int) *DataList {
	data := make([]interface{}, rowCount)
	for i := 0; i < rowCount; i++ {
		data[i] = nil
	}
	return &DataList{
		data:                  data,
		creationTimestamp:     time.Now().Unix(),
		lastModifiedTimestamp: time.Now().Unix(),
	}
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

func safeColumnName(dt *DataTable, name string) string {
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
	dt.lastModifiedTimestamp = time.Now().Unix()
	LogDebug(fmt.Sprintf("Timestamp updated: %d", dt.lastModifiedTimestamp))
}

func (dt *DataTable) GetCreationTimestamp() int64 {
	return dt.creationTimestamp
}

func (dt *DataTable) GetLastModifiedTimestamp() int64 {
	return dt.lastModifiedTimestamp
}
