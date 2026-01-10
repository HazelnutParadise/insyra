package insyra

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HazelnutParadise/Go-Utils/asyncutil"
	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra/internal/core"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/HazelnutParadise/insyra/parallel"
)

// DataTable is the core data structure of Insyra for handling structured data.
// It provides rich data manipulation functionality including reading, writing,
// filtering, statistical analysis, and transformation operations.
//
// DataTable uses a columnar storage format where each column is represented
// by a DataList. It supports both alphabetical column indexing (A, B, C...)
// and named columns, as well as row naming capabilities.
//
// Key features:
// - Thread-safe operations via AtomicDo
// - Flexible data types using any
// - Column and row indexing/naming
// - Comprehensive data manipulation methods
// - CSV/JSON/SQL import/export capabilities
type DataTable struct {
	columns               []*DataList
	rowNames              *core.BiIndex
	name                  string
	creationTimestamp     int64
	lastModifiedTimestamp atomic.Int64

	// Cached column name -> index map for O(1) name lookup (lazy built, invalidated on changes)
	colIndexMap map[string]int

	// AtomicDo support
	cmdCh    chan func()
	initOnce sync.Once
	closed   atomic.Bool

	// Instance-level error tracking for chained operations
	lastError *ErrorInfo
}

func NewDataTable(columns ...*DataList) *DataTable {
	now := time.Now().Unix()
	newTable := &DataTable{
		columns:           []*DataList{},
		rowNames:          core.NewBiIndex(0),
		creationTimestamp: now,
		colIndexMap:       nil,
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
			if len(col.data) > maxLength {
				maxLength = len(col.data)
			}
		}

		for _, col := range columns {
			column := NewDataList()
			column.data = col.data
			column.name = col.name
			column.name = safeColName(dt, column.name)

			dt.columns = append(dt.columns, column)
			if len(column.data) < maxLength {
				column.data = append(column.data, make([]any, maxLength-len(column.data))...)
			}
		}

		for _, col := range dt.columns {
			if len(col.data) < maxLength {
				col.data = append(col.data, make([]any, maxLength-len(col.data))...)
			}
		}

		// Invalidate column cache because schema changed
		dt.colIndexMap = nil

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
				_, _ = dt.rowNames.Set(maxLength, srn)
			}

			if len(rowData.data) > len(dt.columns) {
				for i := len(dt.columns); i < len(rowData.data); i++ {
					newCol := newEmptyDataList(maxLength)
					dt.columns = append(dt.columns, newCol)
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
				colPos, ok := utils.ParseColIndex(colIndex)
				LogDebug("DataTable", "AppendRowsByColIndex", "Handling column %s, colPos: %d, ok: %t", colIndex, colPos, ok)

				if !ok || colPos < 0 || colPos >= len(dt.columns) {
					// 如果該欄位不存在，新增該欄位
					newCol := newEmptyDataList(maxLength)
					dt.columns = append(dt.columns, newCol)
					LogDebug("DataTable", "AppendRowsByColIndex", "Added new column %s at index %d", colIndex, len(dt.columns)-1)
				}

				colPos, _ = utils.ParseColIndex(colIndex)
				if colPos >= 0 && colPos < len(dt.columns) {
					dt.columns[colPos].data = append(dt.columns[colPos].data, value)
				}
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
					LogDebug("DataTable", "AppendRowsByColName", "Added new column %s at index %d", colName, len(dt.columns)-1)
					// Invalidate column cache due to schema change
					dt.colIndexMap = nil
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

// ======================== Get ========================

// GetElement returns the element at the given row and column index.
func (dt *DataTable) GetElement(rowIndex int, columnIndex string) any {
	var result any
	dt.AtomicDo(func(dt *DataTable) {
		columnIndex = strings.ToUpper(columnIndex)
		colPos, ok := utils.ParseColIndex(columnIndex)
		if ok && colPos >= 0 && colPos < len(dt.columns) {
			if rowIndex < 0 {
				rowIndex = len(dt.columns[colPos].data) + rowIndex
			}
			if rowIndex < 0 || rowIndex >= len(dt.columns[colPos].data) {
				dt.warn("GetElement", "Row index is out of range, returning nil")
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
			dt.warn("GetElementByNumberIndex", "Row index is out of range, returning nil")
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
		colPos, ok := utils.ParseColIndex(index)
		if ok && colPos >= 0 && colPos < len(dt.columns) {
			result = dt.columns[colPos].Clone()
			return
		}

		// Try name-based lookup using cache-aware GetColByName
		if res := dt.GetColByName(index); res != nil {
			result = res
			return
		}

		dt.warn("GetCol", "Column '%s' not found, returning nil", index)
		result = nil
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
			dt.warn("GetColByNumber", "Col index is out of range, returning nil")
			result = nil
			return
		}

		result = dt.columns[index].Clone()
	})
	return result
}

func (dt *DataTable) GetColByName(name string) *DataList {
	var result *DataList
	dt.AtomicDo(func(dt *DataTable) {
		// Try cache first
		if dt.colIndexMap != nil {
			if pos, ok := dt.colIndexMap[name]; ok && pos >= 0 && pos < len(dt.columns) {
				result = dt.columns[pos].Clone()
				return
			}
		}

		// Fallback to linear scan and populate cache lazily
		for i, column := range dt.columns {
			if column.name == name {
				// initialize cache if nil
				if dt.colIndexMap == nil {
					dt.colIndexMap = make(map[string]int, len(dt.columns))
				}
				dt.colIndexMap[column.name] = i
				result = column.Clone()
				return
			}
		}
		dt.warn("GetColByName", "Column '%s' not found, returning nil", name)
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
			dt.warn("GetRow", "Row index is out of range, returning nil")
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
		dl.name, _ = dt.GetRowNameByIndex(index)
		result = dl
	})
	return result
}

func (dt *DataTable) GetRowByName(name string) *DataList {
	var result *DataList
	dt.AtomicDo(func(dt *DataTable) {
		if index, exists := dt.rowNames.Index(name); exists {
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
		dt.warn("GetRowByName", "Row name '%s' not found, returning nil", name)
		result = nil
	})
	return result
}

// ======================== Update ========================

// UpdateElement updates the element at the given row and column index.
func (dt *DataTable) UpdateElement(rowIndex int, columnIndex string, value any) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		columnIndex = strings.ToUpper(columnIndex)
		colPos, ok := utils.ParseColIndex(columnIndex)
		if ok && colPos >= 0 && colPos < len(dt.columns) {
			if rowIndex < 0 {
				rowIndex = len(dt.columns[colPos].data) + rowIndex
			}
			if rowIndex < 0 || rowIndex >= len(dt.columns[colPos].data) {
				dt.warn("UpdateElement", "Row index is out of range, returning")
				return
			}
			dt.columns[colPos].data[rowIndex] = value
		} else {
			dt.warn("UpdateElement", "Col index does not exist, returning")
		}
		go dt.updateTimestamp()
	})
	return dt
}

// UpdateCol updates the column with the given index.
func (dt *DataTable) UpdateCol(index string, dl *DataList) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		index = strings.ToUpper(index)
		colPos, ok := utils.ParseColIndex(index)
		if ok && colPos >= 0 && colPos < len(dt.columns) {
			dt.columns[colPos] = dl
		} else {
			dt.warn("UpdateCol", "Col index does not exist, returning")
		}
		go dt.updateTimestamp()
	})
	return dt
}

// UpdateColByNumber updates the column at the given index.
func (dt *DataTable) UpdateColByNumber(index int, dl *DataList) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		if index < 0 {
			index = len(dt.columns) + index
		}

		if index < 0 || index >= len(dt.columns) {
			dt.warn("UpdateColByNumber", "Index out of bounds")
			return
		}

		dt.columns[index] = dl
		go dt.updateTimestamp()
	})
	return dt
}

// UpdateRow updates the row at the given index.
func (dt *DataTable) UpdateRow(index int, dl *DataList) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		if index < 0 || index >= dt.getMaxColLength() {
			dt.warn("UpdateRow", "Index out of bounds")
			return
		}

		if len(dl.data) > len(dt.columns) {
			dt.warn("UpdateRow", "DataList has more elements than DataTable columns, returning")
			return
		}

		// 更新 DataTable 中對應行的資料
		for i := 0; i < len(dl.data); i++ {
			dt.columns[i].data[index] = dl.data[i]
		}

		// 更新行名
		if dl.name != "" {
			srn := safeRowName(dt, dl.name)
			_, _ = dt.rowNames.Set(index, srn)
		}

		go dt.updateTimestamp()
	})
	return dt
}

// ======================== Set ========================

// SetColToRowNames sets the row names to the values of the specified column and drops the column.
func (dt *DataTable) SetColToRowNames(columnIndex string) *DataTable {
	columnIndex = strings.ToUpper(columnIndex)
	dt.AtomicDo(func(dt *DataTable) {
		column := dt.GetCol(columnIndex)
		for i, value := range column.data {
			if value != nil {
				rowName := safeRowName(dt, conv.ToString(value))
				_, _ = dt.rowNames.Set(i, rowName)
			}
		}

		dt.DropColsByIndex(columnIndex)

		go dt.updateTimestamp()
	})
	return dt
}

// SetRowToColNames sets the column names to the values of the specified row and drops the row.
func (dt *DataTable) SetRowToColNames(rowIndex int) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		row := dt.GetRow(rowIndex)
		for i, value := range row.data {
			if value != nil {
				columnName := safeColName(dt, conv.ToString(value))
				dt.columns[i].name = columnName
			}
		}

		dt.DropRowsByIndex(rowIndex)

		go dt.updateTimestamp()
	})
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
		for i := range dt.columns {
			if dt.columns[i].FindFirst(value) != nil {
				if colName, ok := utils.CalcColIndex(i); ok {
					result = append(result, colName)
				}
			}
		}
	})
	return result
}

// FindColsIfContainsAll returns the indices of columns that contain all the given elements.
func (dt *DataTable) FindColsIfContainsAll(values ...any) []string {
	var result []string
	dt.AtomicDo(func(dt *DataTable) {
		for i := range dt.columns {
			foundAll := true

			for _, value := range values {
				if dt.columns[i].FindFirst(value) == nil {
					foundAll = false
					break
				}
			}

			if foundAll {
				if colName, ok := utils.CalcColIndex(i); ok {
					result = append(result, colName)
				}
			}
		}
	})
	return result
}

// FindColsIfAnyElementContainsSubstring returns the indices of columns that contain at least one element that contains the given substring.
func (dt *DataTable) FindColsIfAnyElementContainsSubstring(substring string) []string {
	var result []string
	dt.AtomicDo(func(dt *DataTable) {
		for i := range dt.columns {
			found := false

			for _, value := range dt.columns[i].data {
				if value != nil {
					if str, ok := value.(string); ok && containsSubstring(str, substring) {
						found = true
						break
					}
				}
			}

			if found {
				if colName, ok := utils.CalcColIndex(i); ok {
					result = append(result, colName)
				}
			}
		}
	})
	return result
}

// FindColsIfAllElementsContainSubstring returns the indices of columns that contain all elements that contain the given substring.
func (dt *DataTable) FindColsIfAllElementsContainSubstring(substring string) []string {
	var result []string
	dt.AtomicDo(func(dt *DataTable) {
		for i := range dt.columns {
			foundAll := true

			for _, value := range dt.columns[i].data {
				if value != nil {
					if str, ok := value.(string); ok && !containsSubstring(str, substring) {
						foundAll = false
						break
					}
				}
			}

			if foundAll {
				if colName, ok := utils.CalcColIndex(i); ok {
					result = append(result, colName)
				}
			}
		}
	})
	return result
}

// ======================== Drop ========================

// DropColsByName drops columns by their names.
func (dt *DataTable) DropColsByName(columnNames ...string) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		for _, name := range columnNames {
			for i, col := range dt.columns {
				if col.name == name {
					dt.columns = append(dt.columns[:i], dt.columns[i+1:]...)
					break
				}
			}
		}
		go dt.updateTimestamp()
	})
	return dt
}

// DropColsByIndex drops columns by their index names.
func (dt *DataTable) DropColsByIndex(columnIndices ...string) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		colsToDelete := make([]int, 0)
		for _, index := range columnIndices {
			index = strings.ToUpper(index)
			if colPos, ok := utils.ParseColIndex(index); ok && colPos >= 0 && colPos < len(dt.columns) {
				colsToDelete = append(colsToDelete, colPos)
			}
		}
		sort.Sort(sort.Reverse(sort.IntSlice(colsToDelete)))
		for _, colPos := range colsToDelete {
			dt.columns = append(dt.columns[:colPos], dt.columns[colPos+1:]...)
		}
		go dt.updateTimestamp()
	})
	return dt
}

// DropColsByNumber drops columns by their number.
func (dt *DataTable) DropColsByNumber(columnIndices ...int) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		sort.Sort(sort.Reverse(sort.IntSlice(columnIndices)))

		for _, index := range columnIndices {
			if index >= 0 && index < len(dt.columns) {
				dt.columns = append(dt.columns[:index], dt.columns[index+1:]...)
			}
		}

		go dt.updateTimestamp()
	})
	return dt
}

// DropColsContainString drops columns that contain string elements.
func (dt *DataTable) DropColsContainString() *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		columnsToDelete := make([]int, 0)

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

		for i := len(columnsToDelete) - 1; i >= 0; i-- {
			colIndex := columnsToDelete[i]
			dt.columns = append(dt.columns[:colIndex], dt.columns[colIndex+1:]...)
		}

		go dt.updateTimestamp()
	})
	return dt
}

// DropColsContainNumber drops columns that contain number elements.
func (dt *DataTable) DropColsContainNumber() *DataTable {
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
		}

		go dt.updateTimestamp()
	})
	return dt
}

// DropColsContainNil drops columns that contain nil elements.
func (dt *DataTable) DropColsContainNil() *DataTable {
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
		}

		go dt.updateTimestamp()
	})
	return dt
}

// DropColsContainNaN drops columns that contain NaN (Not a Number) elements.
func (dt *DataTable) DropColsContainNaN() *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		columnsToDelete := make([]int, 0)

		for colIndex, column := range dt.columns {
			containsNaN := false

			for _, value := range column.data {
				if v, ok := value.(float64); ok && math.IsNaN(v) {
					containsNaN = true
					break
				}
			}

			if containsNaN {
				columnsToDelete = append(columnsToDelete, colIndex)
			}
		}

		for i := len(columnsToDelete) - 1; i >= 0; i-- {
			colIndex := columnsToDelete[i]
			dt.columns = append(dt.columns[:colIndex], dt.columns[colIndex+1:]...)
		}

		go dt.updateTimestamp()
	})
	return dt
}

// DropColsContain drops columns that contain the specified value.
func (dt *DataTable) DropColsContain(value ...any) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		columnsToDelete := make([]int, 0)
		for colIndex, column := range dt.columns {
			containsValue := false
			for _, v := range value {
				if slices.Contains(column.data, v) {
					containsValue = true
					break
				}
				if vFloat, ok := v.(float64); ok {
					for _, dataValue := range column.data {
						if dataFloat, ok := dataValue.(float64); ok && math.IsNaN(vFloat) && math.IsNaN(dataFloat) {
							containsValue = true
							break
						}
					}
				}
			}
			if containsValue {
				columnsToDelete = append(columnsToDelete, colIndex)
			}
		}

		for i := len(columnsToDelete) - 1; i >= 0; i-- {
			colIndex := columnsToDelete[i]
			dt.columns = append(dt.columns[:colIndex], dt.columns[colIndex+1:]...)
		}
		go dt.updateTimestamp()
	})
	return dt
}

// DropColsContainExcelNA drops columns that contain Excel NA values ("#N/A").
func (dt *DataTable) DropColsContainExcelNA() *DataTable {
	dt.DropColsContain("#N/A")
	return dt
}

// DropRowsByIndex drops rows by their indices.
func (dt *DataTable) DropRowsByIndex(rowIndices ...int) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		sort.Ints(rowIndices) // 確保從最小索引開始刪除

		for i, rowIndex := range rowIndices {
			if rowIndex < 0 {
				rowIndex = dt.getMaxColLength() + rowIndex
			}
			adjustedIndex := rowIndex - i // 因為每刪除一行，後續的行索引會變動
			if adjustedIndex < 0 || adjustedIndex >= dt.getMaxColLength() {
				continue
			}
			for _, column := range dt.columns {
				if adjustedIndex >= 0 && adjustedIndex < len(column.data) {
					column.data = append(column.data[:adjustedIndex], column.data[adjustedIndex+1:]...)
				}
			}

			dt.reindexRowNamesAfterRemoval(adjustedIndex)
		}
		go dt.updateTimestamp()
	})
	return dt
}

// DropRowsByName drops rows by their names.
func (dt *DataTable) DropRowsByName(rowNames ...string) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		for _, rowName := range rowNames {
			rowIndex, exists := dt.rowNames.Index(rowName)
			if !exists {
				dt.warn("DropRowsByName", "Row name '%s' does not exist", rowName)
				continue
			}

			// 移除所有列中對應行索引的資料
			for _, column := range dt.columns {
				if rowIndex < len(column.data) {
					column.data = append(column.data[:rowIndex], column.data[rowIndex+1:]...)
				}
			}

			dt.reindexRowNamesAfterRemoval(rowIndex)
		}

		go dt.updateTimestamp()
	})
	return dt
}

// DropRowsContainString drops rows that contain string elements.
func (dt *DataTable) DropRowsContainString() *DataTable {
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

			dt.reindexRowNamesAfterRemoval(rowIndex)
		}
		go dt.updateTimestamp()
	})
	return dt
}

// DropRowsContainNumber drops rows that contain number elements.
func (dt *DataTable) DropRowsContainNumber() *DataTable {
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

		dt.remapRowNames(rowsToKeep)

		go dt.updateTimestamp()
	})
	return dt
}

// DropRowsContainNil drops rows that contain nil elements.
func (dt *DataTable) DropRowsContainNil() *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		maxLength := dt.getMaxColLength()

		// 這個切片將存儲所有非nil的行的索引
		nonNilRowIndices := []int{}

		// 遍歷每一行
		for rowIndex := range maxLength {
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
		dt.remapRowNamesByIndices(nonNilRowIndices)
		go dt.updateTimestamp()
	})
	return dt
}

// DropRowsContainNaN drops rows that contain NaN (Not a Number) elements.
func (dt *DataTable) DropRowsContainNaN() *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		maxLength := dt.getMaxColLength()

		// 這個切片將存儲所有不含NaN的行的索引
		nonNaNRowIndices := []int{}

		// 遍歷每一行
		for rowIndex := range maxLength {
			rowHasNaN := false

			// 檢查該行是否包含 NaN
			for _, column := range dt.columns {
				if rowIndex < len(column.data) {
					if v, ok := column.data[rowIndex].(float64); ok && math.IsNaN(v) {
						rowHasNaN = true
						break
					}
				}
			}

			// 如果該行不包含 NaN，將其索引加入到 nonNaNRowIndices 中
			if !rowHasNaN {
				nonNaNRowIndices = append(nonNaNRowIndices, rowIndex)
			}
		}

		// 建立新的列資料，僅保留不含NaN的行
		for _, column := range dt.columns {
			newData := []any{}
			for _, rowIndex := range nonNaNRowIndices {
				if rowIndex < len(column.data) {
					newData = append(newData, column.data[rowIndex])
				}
			}
			column.data = newData
		}

		// 更新 rowNames 映射，以移除被刪除的行
		dt.remapRowNamesByIndices(nonNaNRowIndices)
		go dt.updateTimestamp()
	})
	return dt
}

// DropRowsContain drops rows that contain the specified value.
func (dt *DataTable) DropRowsContain(value ...any) *DataTable {
	dt.AtomicDo(func(dt *DataTable) {
		maxLength := dt.getMaxColLength()
		rowsToKeep := make([]bool, maxLength)

		hasNaNInValue := false
		for _, v := range value {
			if f, ok := v.(float64); ok && math.IsNaN(f) {
				hasNaNInValue = true
				break
			}
		}

		for rowIndex := range maxLength {
			keepRow := true
			for _, column := range dt.columns {
				if rowIndex < len(column.data) && slices.Contains(value, column.data[rowIndex]) {
					keepRow = false
					break
				}
				if hasNaNInValue {
					if rowIndex < len(column.data) {
						if dataFloat, ok := column.data[rowIndex].(float64); ok && math.IsNaN(dataFloat) {
							keepRow = false
							break
						}
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
		dt.remapRowNames(rowsToKeep)
		go dt.updateTimestamp()
	})
	return dt
}

// DropRowsContainExcelNA drops rows that contain Excel NA values ("#N/A").
func (dt *DataTable) DropRowsContainExcelNA() *DataTable {
	dt.DropRowsContain("#N/A")
	return dt
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
			dt.warn("Data", "Too many arguments, returning empty map")
			result = dataMap
			return
		}

		for i, col := range dt.columns {
			var key string
			if useNamesAsKeysBool && col.name != "" {
				key = col.name
			} else {
				key, _ = utils.CalcColIndex(i)
			}
			dataMap[key] = col.data
		}

		result = dataMap
	})
	return result
}

// ToMap is the alias for Data().
// It returns a map[string][]any representation of the DataTable.
func (dt *DataTable) ToMap(useNamesAsKeys ...bool) map[string][]any {
	return dt.Data(useNamesAsKeys...)
}

// ======================== Statistics ========================

// Count returns the number of occurrences of the given value in the DataTable.
func (dt *DataTable) Count(value any) int {
	var count float64
	dt.AtomicDo(func(dt *DataTable) {
		result := asyncutil.ParallelForEach(dt.columns, func(i int, column any) int {
			return dt.columns[i].Count(value)
		})
		count = NewDataList(result).Sum()
	})
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

func (dt *DataTable) NumRows() int {
	var numRows int
	dt.AtomicDo(func(dt *DataTable) {
		numRows = dt.getMaxColLength()
	})
	return numRows
}

func (dt *DataTable) NumCols() int {
	var numCols int
	dt.AtomicDo(func(dt *DataTable) {
		numCols = len(dt.columns)
	})
	return numCols
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

		oldRowNames := dt.rowNames.Clone()
		dt.rowNames = core.NewBiIndex(0)
		nameByIndex := make(map[int]string)
		for _, rowIndex := range oldRowNames.IDs() {
			if name, ok := oldRowNames.Get(rowIndex); ok && name != "" {
				nameByIndex[rowIndex] = name
			}
		}

		newDt := &DataTable{
			columns:           make([]*DataList, 0),
			rowNames:          core.NewBiIndex(0),
			creationTimestamp: dt.GetCreationTimestamp(),
		}

		newDt.lastModifiedTimestamp.Store(dt.GetLastModifiedTimestamp())

		for i, col := range dls {
			newDt.AppendRowsFromDataList(col)
			if name, ok := nameByIndex[i]; ok {
				newDt.columns[i].name = name
			}
		}

		dt.columns = newDt.columns
		dt.rowNames = newDt.rowNames

		go dt.updateTimestamp()
		result = dt
	})
	return result
}

// Clone creates a deep copy of the DataTable.
// It copies all columns, column indices, row names, and metadata,
// ensuring that modifications to the original DataTable do not affect the clone.
// The cloned DataTable has a new creation timestamp and is fully independent.
func (dt *DataTable) Clone() *DataTable {
	var newDT *DataTable
	now := time.Now().Unix()
	dt.AtomicDo(func(dt *DataTable) {
		clonedColumns := make([]*DataList, len(dt.columns))
		var clonedRowNames *core.BiIndex
		parallel.GroupUp(func() {
			for i, col := range dt.columns {
				clonedColumns[i] = col.Clone()
			}
		}, func() {
			clonedRowNames = dt.rowNames.Clone()
		}).Run().AwaitResult()

		newDT = &DataTable{
			columns:           clonedColumns,
			rowNames:          clonedRowNames,
			name:              dt.name,
			creationTimestamp: now,
		}
	})
	newDT.lastModifiedTimestamp.Store(now)
	return newDT
}

// To2DSlice converts the DataTable to a 2D slice of any.
// Each row in the DataTable becomes a slice in the outer slice,
// and each column's element at that row becomes an element in the inner slice.
// If a column is shorter than the maximum row count, nil values are used to fill.
func (dt *DataTable) To2DSlice() [][]any {
	var result [][]any
	dt.AtomicDo(func(dt *DataTable) {
		maxRows := dt.getMaxColLength()
		result = make([][]any, maxRows)
		for i := range maxRows {
			result[i] = make([]any, len(dt.columns))
			for j, col := range dt.columns {
				if i < len(col.data) {
					result[i][j] = col.data[i]
				} else {
					result[i][j] = nil
				}
			}
		}
	})
	return result
}

// ======================== Utilities ========================

func (dt *DataTable) getRowNameByIndex(index int) (string, bool) {
	if dt.rowNames == nil {
		return "", false
	}
	return dt.rowNames.Get(index)
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

	if dt.rowNames == nil {
		dt.rowNames = core.NewBiIndex(0)
	}

	for dt.rowNames.Has(name) {
		name = fmt.Sprintf("%s_%d", originalName, counter)
		counter++
	}

	return name
}

func (dt *DataTable) reindexRowNamesAfterRemoval(removedIndex int) {
	if dt.rowNames == nil || dt.rowNames.Len() == 0 {
		return
	}
	_, _, _ = dt.rowNames.DeleteAndShift(removedIndex)
}

func (dt *DataTable) remapRowNames(rowsToKeep []bool) {
	if dt.rowNames == nil || dt.rowNames.Len() == 0 {
		dt.rowNames = core.NewBiIndex(0)
		return
	}
	indexMap := make([]int, len(rowsToKeep))
	newPos := 0
	for i, keep := range rowsToKeep {
		if keep {
			indexMap[i] = newPos
			newPos++
		} else {
			indexMap[i] = -1
		}
	}
	remapped := core.NewBiIndex(dt.rowNames.Len())
	for _, id := range dt.rowNames.IDs() {
		if id < 0 || id >= len(indexMap) {
			continue
		}
		target := indexMap[id]
		if target < 0 {
			continue
		}
		name, ok := dt.rowNames.Get(id)
		if !ok || name == "" {
			continue
		}
		_, _ = remapped.Set(target, name)
	}
	dt.rowNames = remapped
}

func (dt *DataTable) remapRowNamesByIndices(keptIndices []int) {
	if dt.rowNames == nil || dt.rowNames.Len() == 0 {
		dt.rowNames = core.NewBiIndex(0)
		return
	}
	indexMap := make(map[int]int, len(keptIndices))
	for newIdx, oldIdx := range keptIndices {
		indexMap[oldIdx] = newIdx
	}
	remapped := core.NewBiIndex(dt.rowNames.Len())
	for _, id := range dt.rowNames.IDs() {
		name, ok := dt.rowNames.Get(id)
		if !ok || name == "" {
			continue
		}
		if target, exists := indexMap[id]; exists {
			_, _ = remapped.Set(target, name)
		}
	}
	dt.rowNames = remapped
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

// Err returns the last error that occurred during a chained operation.
// Returns nil if no error occurred.
// This method allows for error checking after chained calls without breaking the chain.
//
// Example usage:
//
//	dt.Replace(old, new).ReplaceNaNsWith(0).SortBy(config)
//	if err := dt.Err(); err != nil {
//	    // handle error
//	}
func (dt *DataTable) Err() *ErrorInfo {
	return dt.lastError
}

// ClearErr clears the last error stored in the DataTable.
// Returns the DataTable to support chaining.
func (dt *DataTable) ClearErr() *DataTable {
	dt.lastError = nil
	return dt
}

// setError is an internal method to record an error on the DataTable instance.
func (dt *DataTable) setError(level LogLevel, packageName, funcName, message string) {
	dt.lastError = &ErrorInfo{
		Level:       level,
		PackageName: packageName,
		FuncName:    funcName,
		Message:     message,
		Timestamp:   time.Now(),
	}
}

// warn logs a warning and sets the error on the DataTable instance.
// This is a convenience method that combines LogWarning and setError.
func (dt *DataTable) warn(funcName, msg string, args ...any) {
	fullMsg := fmt.Sprintf(msg, args...)
	LogWarning("DataTable", funcName, "%s", fullMsg)
	dt.setError(LogLevelWarning, "DataTable", funcName, fullMsg)
}
