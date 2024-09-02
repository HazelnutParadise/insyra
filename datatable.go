package insyra

import (
	"sync"
	"time"
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
	AppendRows(rowsData ...map[string]interface{})
	Data(useNamesAsKeys ...bool) map[string][]interface{}
	Size() (int, int)
	updateTimestamp()
	updateColumnNames()
	SetCustomIndex(index []string)
	getMaxColumnLength() int
}

// NewDataTable creates a new DataTable with a specified number of columns and rows.
func NewDataTable(rowCount, columnCount int) *DataTable {
	newTable := &DataTable{
		columns:               make(map[string]*DataList),
		creationTimestamp:     time.Now().Unix(),
		lastModifiedTimestamp: time.Now().Unix(),
	}

	for i := 0; i < columnCount; i++ {
		columnName := generateColumnName(i)
		newTable.columns[columnName] = newEmptyDataList(rowCount)
	}

	return newTable
}

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

// AppendRow appends new rows to the DataTable.
func (dt *DataTable) AppendRows(rowsData ...map[string]interface{}) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	// 檢查是否需要新增自訂索引
	if len(dt.customIndex) < dt.getMaxColumnLength()+1 {
		dt.customIndex = append(dt.customIndex, generateColumnName(len(dt.customIndex)))
	}

	// 將新的資料插入到對應的 DataList 中
	for _, rowData := range rowsData {
		for colName, value := range rowData {
			if _, exists := dt.columns[colName]; !exists {
				LogWarning("column %s does not exist in the DataTable, skipping.", colName)
				continue
			}
			dt.columns[colName].data = append(dt.columns[colName].data, value)
		}

		// 如果某一個 column 沒有對應的新數據，則插入 nil 來保持每列的長度一致
		for colName, column := range dt.columns {
			if _, exists := rowData[colName]; !exists {
				column.data = append(column.data, nil)
			}
		}
	}
}

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
			key = col.name
		} else {
			key = "[" + i + "]"
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

// updateTimestamp updates the last modified timestamp of the DataTable.
func (dt *DataTable) updateTimestamp() {
	dt.lastModifiedTimestamp = time.Now().Unix()
}

// updateColumnNames updates the column names (keys in the map) to be in sequential order.
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
