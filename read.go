package insyra

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	csvInternal "github.com/HazelnutParadise/insyra/internal/csv"
	json "github.com/goccy/go-json"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// Alias for Slice2DToDataTable
// Converts a 2D slice into a DataTable.
// Supports various types like [][]any, [][]int, [][]float64, [][]string, etc.
var ReadSlice2D = Slice2DToDataTable

// Slice2DToDataTable converts a 2D slice of any type into a DataTable.
// It supports various 2D array types such as [][]any, [][]int, [][]float64, [][]string, etc.
func Slice2DToDataTable(data any) (*DataTable, error) {
	if data == nil {
		return nil, fmt.Errorf("input data cannot be nil")
	}

	rv := reflect.ValueOf(data)

	// 驗證輸入是否為切片
	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("input data must be a 2D slice, got %s", rv.Kind())
	}

	// 如果是空切片
	if rv.Len() == 0 {
		return nil, fmt.Errorf("input data cannot be empty")
	}

	// 檢查第一個元素是否為切片（確保是二維陣列）
	firstRow := rv.Index(0)
	if firstRow.Kind() != reflect.Slice && firstRow.Kind() != reflect.Array {
		return nil, fmt.Errorf("input data must be a 2D slice, first element is %s", firstRow.Kind())
	}

	numCols := firstRow.Len()
	if numCols == 0 {
		return nil, fmt.Errorf("first row cannot be empty")
	}

	dls := make([]*DataList, numCols)
	// 初始化列
	for i := range dls {
		dls[i] = NewDataList()
	}

	// 遍歷所有行
	for rowIdx := 0; rowIdx < rv.Len(); rowIdx++ {
		row := rv.Index(rowIdx)

		// 處理行作為切片或陣列
		if row.Kind() != reflect.Slice && row.Kind() != reflect.Array {
			return nil, fmt.Errorf("row %d is not a slice or array, got %s", rowIdx, row.Kind())
		}

		rowLen := row.Len()

		// 遍歷所有列
		for colIdx := range numCols {
			if colIdx < rowLen {
				// 將值轉換為 interface{}
				dls[colIdx].Append(row.Index(colIdx).Interface())
			} else {
				// 不足的列填 nil
				dls[colIdx].Append(nil)
			}
		}
	}

	dt := NewDataTable(dls...)
	return dt, nil
}

// ----- csv -----

// ReadCSV_File loads a CSV file into a DataTable, with options to set the first column as row names
// and the first row as column names.
func ReadCSV_File(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool, encoding ...string) (*DataTable, error) {
	dt := NewDataTable()

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	if len(encoding) == 0 {
		encoding = []string{"auto"}
	}
	useEncoding := strings.ToLower(encoding[0])
	if useEncoding == "auto" {
		detected, err := DetectEncoding(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-detect encoding for %s: %v", filePath, err)
		}
		useEncoding = detected
		LogInfo("csvxl", "ReadCSV_File", "Auto-detected encoding %s for file %s", useEncoding, filePath)
	}

	// Use internal CSV reader with encoding support
	csvString, err := csvInternal.ReadCSVWithEncoding(file, useEncoding)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file %s: %w", filePath, err)
	}

	reader := csv.NewReader(strings.NewReader(csvString))
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	dt.columns = []*DataList{}
	dt.columnIndex = make(map[string]int)
	dt.rowNames = make(map[string]int)

	if len(rows) == 0 {
		return dt, nil // 空的CSV
	}

	// 處理第一行是否為欄名
	startRow := 0
	if setFirstRowToColNames {
		for i, colName := range rows[0] {
			if setFirstColToRowNames && i == 0 {
				// 第一欄是行名，不作為列名處理
				continue
			}
			column := &DataList{name: safeColName(dt, colName)}
			dt.columns = append(dt.columns, column)
			dt.columnIndex[generateColIndex(len(dt.columns)-1)] = len(dt.columns) - 1
		}
		startRow = 1
	} else {
		// 如果沒有指定第一行作為列名，則動態生成列名
		for i := range rows[0] {
			if setFirstColToRowNames && i == 0 {
				continue
			}
			column := &DataList{}
			dt.columns = append(dt.columns, column)
			dt.columnIndex[generateColIndex(len(dt.columns)-1)] = len(dt.columns) - 1
		}
	}

	// 處理資料行和是否將第一欄作為行名
	for rowIndex, row := range rows[startRow:] {
		if setFirstColToRowNames {
			rowName := row[0]
			dt.rowNames[safeRowName(dt, rowName)] = rowIndex
			row = row[1:] // 移除第一欄作為行名
		}

		for colIndex, cell := range row {
			if colIndex >= len(dt.columns) {
				continue
			}
			column := dt.columns[colIndex]
			if num, err := strconv.ParseFloat(cell, 64); err == nil {
				column.data = append(column.data, num)
			} else {
				column.data = append(column.data, cell)
			}
		}
	}

	return dt, nil
}

func ReadCSV_String(csvString string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error) {
	dt := NewDataTable()

	reader := csv.NewReader(strings.NewReader(csvString))
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	dt.columns = []*DataList{}
	dt.columnIndex = make(map[string]int)
	dt.rowNames = make(map[string]int)

	if len(rows) == 0 {
		return dt, nil // 空的CSV
	}

	// 處理第一行是否為欄名
	startRow := 0
	if setFirstRowToColNames {
		for i, colName := range rows[0] {
			if setFirstColToRowNames && i == 0 {
				continue
			}
			column := &DataList{name: safeColName(dt, colName)}
			dt.columns = append(dt.columns, column)
			dt.columnIndex[generateColIndex(len(dt.columns)-1)] = len(dt.columns) - 1
		}
		startRow = 1
	} else {
		for i := range rows[0] {
			if setFirstColToRowNames && i == 0 {
				continue
			}
			column := &DataList{}
			dt.columns = append(dt.columns, column)
			dt.columnIndex[generateColIndex(len(dt.columns)-1)] = len(dt.columns) - 1
		}
	}

	for rowIndex, row := range rows[startRow:] {
		if setFirstColToRowNames {
			rowName := row[0]
			dt.rowNames[safeRowName(dt, rowName)] = rowIndex
			row = row[1:] // 移除第一欄作為行名
		}

		for colIndex, cell := range row {
			if colIndex >= len(dt.columns) {
				continue
			}
			column := dt.columns[colIndex]
			if num, err := strconv.ParseFloat(cell, 64); err == nil {
				column.data = append(column.data, num)
			} else {
				column.data = append(column.data, cell)
			}
		}
	}

	return dt, nil
}

// ----- excel -----

// ReadExcelSheet reads a specific sheet from an Excel file and loads it into a DataTable.
func ReadExcelSheet(filePath string, sheetName string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file %s: %v", filePath, err)
	}
	defer func() { _ = f.Close() }()
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from sheet %s: %v", sheetName, err)
	}
	dt, err := ReadSlice2D(rows)
	if err != nil {
		return nil, fmt.Errorf("failed when converting sheet %s to DataTable: %v", sheetName, err)
	}
	if setFirstColToRowNames {
		dt.SetColToRowNames("A")
	}
	if setFirstRowToColNames {
		dt.SetRowToColNames(0)
	}
	return dt, nil
}

// ----- json -----

// ReadJSON_File reads a JSON file and loads the data into a DataTable and returns it.
func ReadJSON_File(filePath string) (*DataTable, error) {
	dt := NewDataTable()

	// 讀取檔案
	buf, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// 解析 JSON
	var rows []map[string]any
	err = json.Unmarshal(buf, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// 將資料加入 DataTable
	for _, row := range rows {
		dt.AppendRowsByColName(row)
	}

	return dt, nil
}

func ReadJSON(data any) (*DataTable, error) {
	dt := NewDataTable()

	if data == nil {
		return nil, fmt.Errorf("input data cannot be nil")
	}

	var rows []map[string]any

	switch v := data.(type) {
	case []map[string]any:
		rows = v
	case map[string]any:
		rows = append(rows, v)
	case []byte:
		if err := json.Unmarshal(v, &rows); err != nil {
			// try single object
			var single map[string]any
			if err2 := json.Unmarshal(v, &single); err2 == nil {
				rows = append(rows, single)
			} else {
				return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
			}
		}
	case string:
		b := []byte(v)
		if err := json.Unmarshal(b, &rows); err != nil {
			var single map[string]any
			if err2 := json.Unmarshal(b, &single); err2 == nil {
				rows = append(rows, single)
			} else {
				return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
			}
		}
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				rows = append(rows, m)
				continue
			}
			b, err := json.Marshal(item)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal array element: %w", err)
			}
			var m map[string]any
			if err := json.Unmarshal(b, &m); err != nil {
				return nil, fmt.Errorf("failed to unmarshal array element: %w", err)
			}
			rows = append(rows, m)
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input: %w", err)
		}
		if err := json.Unmarshal(b, &rows); err != nil {
			var single map[string]any
			if err2 := json.Unmarshal(b, &single); err2 == nil {
				rows = append(rows, single)
			} else {
				return nil, fmt.Errorf("failed to marshal/unmarshal input as JSON: %w", err)
			}
		}
	}

	// 將資料加入 DataTable
	for _, row := range rows {
		dt.AppendRowsByColName(row)
	}

	return dt, nil
}

// ----- sql -----

type ReadSQLOptions struct {
	RowNameColumn string // Specifies which column to use as row names, default is "row_name"
	Query         string // Custom SQL query, if provided, other options will be ignored
	Limit         int    // Limit the number of rows to read
	Offset        int    // Starting position for reading rows
	WhereClause   string // WHERE clause
	OrderBy       string // ORDER BY clause
}

// ReadSQL reads table data from the database and converts it into a DataTable object.
// If a custom Query is provided in options, it will use the custom query.
// Otherwise, it uses tableName and other options to construct the SQL query.
func ReadSQL(db *gorm.DB, tableName string, options ...ReadSQLOptions) (*DataTable, error) {
	if db == nil {
		return nil, fmt.Errorf("db 參數不能為空")
	}

	// 設定默認選項
	var opts ReadSQLOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts = ReadSQLOptions{
			RowNameColumn: "row_name",
			Limit:         0, // 0 表示不限制
			Offset:        0,
		}
	}

	// 確定使用的查詢語句
	var query string
	var params []any

	if opts.Query != "" {
		// 使用自訂查詢
		query = opts.Query
	} else {
		// 檢查表格是否存在
		dialect := db.Name()
		var result *gorm.DB
		var count int

		switch dialect {
		case "mysql":
			result = db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = (SELECT DATABASE())", tableName)
		case "postgres":
			result = db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = current_schema()", tableName)
		case "sqlite":
			// SQLite 特有的查詢方式
			result = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName)
		default:
			// 未知數據庫，使用一般性方法
			result = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName)
		}

		if err := result.Scan(&count).Error; err != nil {
			return nil, fmt.Errorf("檢查表格 %s 存在時發生錯誤: %w", tableName, err)
		}

		if count == 0 {
			return nil, fmt.Errorf("表格 %s 不存在", tableName)
		}

		// 構建 SQL 查詢
		query = fmt.Sprintf("SELECT * FROM %s", tableName)
		if opts.WhereClause != "" {
			query += " WHERE " + opts.WhereClause
		}
		if opts.OrderBy != "" {
			query += " ORDER BY " + opts.OrderBy
		}
		if opts.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", opts.Limit)
		}
		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", opts.Offset)
		}
	}
	// 執行查詢
	LogDebug("core", "ReadSQL", "執行SQL查詢: %s", query)
	rows, err := db.Raw(query, params...).Rows()
	if err != nil {
		return nil, fmt.Errorf("執行查詢時發生錯誤: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// 獲取列名
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("獲取列名時發生錯誤: %w", err)
	}

	// 確定行名列的索引（如果有）
	rowNameColIndex := -1
	for i, colName := range columnNames {
		if strings.EqualFold(colName, opts.RowNameColumn) {
			rowNameColIndex = i
			break
		}
	}

	// 為每一列創建 DataList
	dataLists := make([]*DataList, len(columnNames))
	for i, colName := range columnNames {
		if i != rowNameColIndex || rowNameColIndex == -1 {
			dl := NewDataList()
			dl.SetName(colName) // 明確設置列名為資料庫中的列名
			dataLists[i] = dl
		}
	}
	// 處理行
	rowValues := make([]any, len(columnNames))
	scanArgs := make([]any, len(columnNames))
	for i := range rowValues {
		scanArgs[i] = &rowValues[i]
	}

	// 用來保存行名稱，之後再設置
	rowNames := make(map[int]string)
	var rowIndex = 0

	for rows.Next() {
		// 讀取數據
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("讀取行數據時發生錯誤: %w", err)
		}

		// 保存行名（如果有）
		if rowNameColIndex >= 0 {
			if rowValues[rowNameColIndex] != nil {
				rowName := fmt.Sprintf("%v", rowValues[rowNameColIndex])
				rowNames[rowIndex] = rowName
			}
		}

		// 將數據添加到相應的列
		for i, val := range rowValues {
			// 跳過行名列，因為它已經被處理了
			if i != rowNameColIndex || rowNameColIndex == -1 {
				if dataLists[i] != nil {
					// 對 SQL NULL 值的處理
					if val == nil {
						dataLists[i].Append(nil)
					} else {
						// 處理不同類型的數據
						switch v := val.(type) {
						case []byte:
							// 將 []byte 轉換為字符串
							dataLists[i].Append(string(v))
						default:
							dataLists[i].Append(v)
						}
					}
				}
			}
		}
		rowIndex++
	} // 檢查是否有任何迭代錯誤
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代行時發生錯誤: %w", err)
	}

	// 將列添加到 DataTable 中
	// 確保列顯示順序與資料庫表格相同
	dt := NewDataTable()
	validColumns := make([]*DataList, 0, len(dataLists))
	for i, dl := range dataLists {
		if i != rowNameColIndex || rowNameColIndex == -1 {
			if dl != nil && len(dl.data) > 0 {
				validColumns = append(validColumns, dl)
			}
		}
	}
	if len(validColumns) > 0 {
		dt.AppendCols(validColumns...)

		// 在這裡設置行名
		if rowNameColIndex >= 0 {
			for i, name := range rowNames {
				dt.SetRowNameByIndex(i, name)
			}
		}
	}

	return dt, nil
}
