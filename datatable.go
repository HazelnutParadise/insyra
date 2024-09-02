package insyra

import (
	"sync"
	"time"

	"github.com/HazelnutParadise/Go-Utils/sliceutil"
)

type DataTable struct {
	mu                    sync.Mutex
	columns               map[string]*DataList
	customIndex           []string
	creationTimestamp     int64
	lastModifiedTimestamp int64
}

type IDataTable interface {
	AppendColumns(columns ...*DataList)
	AppendRowsByIndex(rowsData ...map[string]interface{})
	AppendRowsByName(rowsData ...map[string]interface{})
	Data(useNamesAsKeys ...bool) map[string][]interface{}
	Size() (int, int)
	DropColumnsByIndex(columnIndices ...string)
	DropColumnsByName(columnNames ...string)
	DropRowsByIndex(rowIndices ...int)
	GetColumnByName(columnName string) *DataList
	GetRowByIndex(rowIndex int) *DataList
	updateTimestamp()
	updateColumnNames()
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	SetCustomIndex(index []string)
	getMaxColumnLength()
}

// NewDataTable creates a new empty DataTable or initializes it with provided DataLists as columns.
func NewDataTable(columns ...*DataList) *DataTable {
	newTable := &DataTable{
		columns:               make(map[string]*DataList),
		creationTimestamp:     time.Now().Unix(),
		lastModifiedTimestamp: time.Now().Unix(),
	}

	if len(columns) > 0 {
		// 依照順序生成列索引
		for i, column := range columns {
			columnName := generateColumnName(i)
			newTable.columns[columnName] = column
		}
	}

	return newTable
}

// ======================== Append ========================

// AddColumns adds columns to the DataTable and ensures that all columns have the same length.
func (dt *DataTable) AppendColumns(columns ...*DataList) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	// Find the maximum length of existing columns
	maxLength := 0
	for _, col := range dt.columns {
		if len(col.data) > maxLength {
			maxLength = len(col.data)
		}
	}

	// Add new columns and ensure their length matches the maximum length
	for i, column := range columns {
		if len(column.data) < maxLength {
			// Fill with nil to match the length
			column.data = append(column.data, make([]interface{}, maxLength-len(column.data))...)
		}
		columnName := generateColumnName(len(dt.columns) + i)
		dt.columns[columnName] = column
	}

	// Update other columns to match the new max length
	for _, col := range dt.columns {
		if len(col.data) < maxLength {
			col.data = append(col.data, make([]interface{}, maxLength-len(col.data))...)
		}
	}

	// Update custom index length if needed
	if len(dt.customIndex) < maxLength {
		dt.customIndex = append(dt.customIndex, make([]string, maxLength-len(dt.customIndex))...)
	}

}

// AppendRowsByIndex appends new rows to the DataTable based on the auto-generated column index.
func (dt *DataTable) AppendRowsByIndex(rowsData ...map[string]interface{}) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	// 先確定要新增的行數
	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		// 先檢查是否有新的 column，提前補齊舊的 column
		for colIndex := range rowData {
			if _, exists := dt.columns[colIndex]; !exists {
				dt.columns[colIndex] = newEmptyDataList(maxLength) // 創建新列
			}
		}

		// 插入新行
		for colIndex, value := range rowData {
			dt.columns[colIndex].data = append(dt.columns[colIndex].data, value)
		}

		// 將缺少資料的列補齊 nil
		for _, column := range dt.columns {
			if len(column.data) == maxLength {
				column.data = append(column.data, nil)
			}
		}
	}
}

// AppendRowsByName appends new rows to the DataTable based on the DataList name.
func (dt *DataTable) AppendRowsByName(rowsData ...map[string]interface{}) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		// 先檢查是否有新的 column，提前補齊舊的 column
		for colName := range rowData {
			columnFound := false
			for _, column := range dt.columns {
				if column.name == colName {
					columnFound = true
					break
				}
			}

			if !columnFound {
				newCol := newEmptyDataList(maxLength)
				newCol.name = colName
				dt.columns[generateColumnName(len(dt.columns))] = newCol
			}
		}

		// 插入新行
		for colName, value := range rowData {
			for _, column := range dt.columns {
				if column.name == colName {
					column.data = append(column.data, value)
				}
			}
		}

		// 將缺少資料的列補齊 nil
		for _, column := range dt.columns {
			if len(column.data) == maxLength {
				column.data = append(column.data, nil)
			}
		}
	}
}

// ======================== Data ========================

// Data 方法返回一個 map，可以選擇使用 DataList 的 name 作為鍵或使用自動生成的列索引。
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
			key = col.name + "(" + i + ")"
		} else {
			key = "(" + i + ")"
		}
		dataMap[key] = col.data
	}

	return dataMap
}

// Size returns the number of rows and columns in the DataTable.
func (dt *DataTable) Size() (int, int) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	numColumns := len(dt.columns)

	var numRows int
	for _, col := range dt.columns {
		numRows = len(col.data)
		break
	}

	return numRows, numColumns
}

// ======================== Drop ========================

// DropColumnsByIndex drops columns by their auto-generated index and then renames the remaining columns in alphabetical order.
func (dt *DataTable) DropColumnsByIndex(columnIndices ...string) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, delColumnIndex := range columnIndices {
		delete(dt.columns, delColumnIndex)
	}

	// 更新其餘列的名稱
	dt.updateColumnNames()
}

// DropColumnsByName drops columns by name and then renames the remaining columns in alphabetical order.
func (dt *DataTable) DropColumnsByName(columnNames ...string) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, columnName := range columnNames {
		for columnIndex, column := range dt.columns {
			if column.name == columnName {
				delete(dt.columns, columnIndex)
			}
		}
	}

	// 更新其餘列的名稱
	dt.updateColumnNames()
}

// DropRowsByIndex drops rows by index.
func (dt *DataTable) DropRowsByIndex(rowIndices ...int) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	if len(rowIndices) == 0 {
		LogWarning("DataTable.DropRowsByIndex(): no row indices provided, skipping.")
		return
	}

	for i, rowIndex := range rowIndices {
		if rowIndex < 0 {
			rowIndex = len(dt.columns) + rowIndex
		}
		if rowIndex >= dt.getMaxColumnLength() {
			LogWarning("DataTable.DropRowsByIndex(): row index out of range, skipping.")
			rowIndices, _ = sliceutil.Remove[int](rowIndices, i)
			i-- // 因為刪除了一個元素，所以 i 要減 1
			continue
		}

	}

	for _, rowIndex := range rowIndices {
		for _, column := range dt.columns {
			column.Drop(rowIndex)
		}
	}
}

// ======================== Get ========================

// GetColumnByName gets a column by name and return a DataList.
// Return nil if columnName is not found.
func (dt *DataTable) GetColumnByName(columnName string) *DataList {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	for _, column := range dt.columns {
		if column.name == columnName {
			dl := NewDataList(column.data)
			return dl
		}
	}
	LogWarning("DataTable.GetColumn(): column not found, returning nil.")
	return nil
}

// GetRowByIndex gets a row by index and return a DataList.
// Return nil if rowIndex is out of range.
func (dt *DataTable) GetRowByIndex(rowIndex int) *DataList {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if rowIndex < 0 {
		rowIndex += dt.getMaxColumnLength()
	}
	if rowIndex >= dt.getMaxColumnLength() {
		LogWarning("DataTable.GetRowByIndex(): row index out of range, returning nil.")
		return nil
	}

	dl := NewDataList()

	for _, column := range dt.columns {
		dl.Append(column.data[rowIndex])
	}

	return dl
}

// SetCustomIndex sets a custom index for the DataTable and ensures it matches the length of columns.
func (dt *DataTable) SetCustomIndex(index []string) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	maxLength := dt.getMaxColumnLength()

	if len(index) < maxLength {
		// Fill the custom index with empty strings to match the length
		index = append(index, make([]string, maxLength-len(index))...)
	}

	dt.customIndex = index[:maxLength]

}

// getMaxColumnLength returns the maximum length of the columns in the DataTable.
func (dt *DataTable) getMaxColumnLength() int {
	maxLength := 0
	for _, col := range dt.columns {
		if len(col.data) > maxLength {
			maxLength = len(col.data)
		}
	}
	return maxLength
}

// ======================== Timestamp ========================
// updateTimestamp updates the last modified timestamp of the DataTable.
func (dt *DataTable) updateTimestamp() {
	dt.lastModifiedTimestamp = time.Now().Unix()
}

func (dt *DataTable) GetCreationTimestamp() int64 {
	return dt.creationTimestamp
}

func (dt *DataTable) GetLastModifiedTimestamp() int64 {
	return dt.lastModifiedTimestamp
}

// ======================== Column Names ========================
// updateColumnNames updates the column names (keys in the map) to be in sequential order without skipping letters.
// Auto update timestamp.
func (dt *DataTable) updateColumnNames() {
	defer func() {
		go dt.updateTimestamp()
	}()
	updatedColumns := make(map[string]*DataList)
	i := 0
	for _, column := range dt.columns {
		columnName := generateColumnName(i)
		updatedColumns[columnName] = column
		i++
	}
	dt.columns = updatedColumns
}

func generateColumnName(index int) string {
	name := ""
	for index >= 0 {
		name = string('A'+(index%26)) + name
		index = index/26 - 1
	}
	return name
}

// ======================== functions ========================
// newEmptyDataList creates a new DataList with a specified number of empty rows (filled with nil).
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
