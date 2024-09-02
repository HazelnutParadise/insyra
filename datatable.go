package insyra

import (
	"fmt"
	"sort"
	"sync"
	"time"
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
	Data(useNamesAsKeys ...bool) map[string][]interface{}
	Show()
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	getSortedColumnNames() []string
	getRowNameByIndex(index int) (string, bool)
	getMaxColumnLength() int
	updateTimestamp()
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

func (dt *DataTable) AppendColumns(columns ...*DataList) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	maxLength := dt.getMaxColumnLength()

	for i, column := range columns {
		columnName := generateColumnName(len(dt.columns) + i)
		dt.columns = append(dt.columns, column)
		dt.columnIndex[columnName] = len(dt.columns) - 1
		if len(column.data) < maxLength {
			column.data = append(column.data, make([]interface{}, maxLength-len(column.data))...)
		}
	}

	for _, col := range dt.columns {
		if len(col.data) < maxLength {
			col.data = append(col.data, make([]interface{}, maxLength-len(col.data))...)
		}
	}

	dt.updateTimestamp()
}

func (dt *DataTable) AppendRowsFromDataList(rowsData ...*DataList) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		if rowData.name != "" {
			dt.rowNames[rowData.name] = maxLength
		}

		if len(rowData.data) > len(dt.columns) {
			for i := len(dt.columns); i < len(rowData.data); i++ {
				newCol := newEmptyDataList(maxLength)
				columnName := generateColumnName(i)
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

	dt.updateTimestamp()
}

func (dt *DataTable) AppendRowsByIndex(rowsData ...map[string]interface{}) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		for colIndex, value := range rowData {
			// 使用 columnIndex 查找對應的切片索引
			colPos, exists := dt.columnIndex[colIndex]
			if !exists {
				// 如果列不存在，創建新列
				newCol := newEmptyDataList(maxLength)
				dt.columns = append(dt.columns, newCol)
				colPos = len(dt.columns) - 1
				dt.columnIndex[colIndex] = colPos // 確保 columnIndex map 正確更新
			}
			dt.columns[colPos].data = append(dt.columns[colPos].data, value)
		}

		// 填補空白資料
		for _, column := range dt.columns {
			if len(column.data) <= maxLength {
				column.data = append(column.data, nil)
			}
		}
	}

	dt.updateTimestamp()
}

func (dt *DataTable) AppendRowsByName(rowsData ...map[string]interface{}) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		for colName, value := range rowData {
			found := false
			for i := 0; i < len(dt.columns); i++ {
				if dt.columns[i].name == colName {
					dt.columns[i].data = append(dt.columns[i].data, value)
					found = true
					break
				}
			}
			if !found {
				// 創建新列並插入資料
				newCol := newEmptyDataList(maxLength)
				newCol.name = colName
				newCol.data = append(newCol.data, value)
				dt.columns = append(dt.columns, newCol)
				// 更新字母索引
				dt.columnIndex[generateColumnName(len(dt.columns)-1)] = len(dt.columns) - 1 // 確保 columnIndex map 正確更新
			}
		}

		// 填充其他缺少資料的列
		for _, column := range dt.columns {
			if len(column.data) == maxLength {
				column.data = append(column.data, nil)
			}
		}
	}

	dt.updateTimestamp()
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
			key = fmt.Sprintf("%s(%s)", generateColumnName(i), col.name)
		} else {
			key = generateColumnName(i)
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
		key := generateColumnName(i)
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

	// 計算 RowNames 的最大寬度
	rowNames := make([]string, dt.getMaxColumnLength())
	maxRowNameWidth := len("RowNames")
	for i := range rowNames {
		if rowName, exists := dt.getRowNameByIndex(i); exists {
			rowNames[i] = rowName
		} else {
			rowNames[i] = fmt.Sprintf("%d", i)
		}
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

func generateColumnName(index int) string {
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

func (dt *DataTable) updateTimestamp() {
	dt.lastModifiedTimestamp = time.Now().Unix()
}

func (dt *DataTable) GetCreationTimestamp() int64 {
	return dt.creationTimestamp
}

func (dt *DataTable) GetLastModifiedTimestamp() int64 {
	return dt.lastModifiedTimestamp
}
