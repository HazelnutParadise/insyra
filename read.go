package insyra

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/core"
	csvInternal "github.com/HazelnutParadise/insyra/internal/csv"
	json "github.com/goccy/go-json"
	"github.com/xuri/excelize/v2"
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
				// 將值轉換為 any
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
	dt.rowNames = core.NewBiIndex(0)

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
		}
	}

	// 處理資料行和是否將第一欄作為行名
	for rowIndex, row := range rows[startRow:] {
		if setFirstColToRowNames {
			rowName := row[0]
			_, _ = dt.rowNames.Set(rowIndex, safeRowName(dt, rowName))
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
	dt.rowNames = core.NewBiIndex(0)

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
		}
		startRow = 1
	} else {
		for i := range rows[0] {
			if setFirstColToRowNames && i == 0 {
				continue
			}
			column := &DataList{}
			dt.columns = append(dt.columns, column)
		}
	}

	for rowIndex, row := range rows[startRow:] {
		if setFirstColToRowNames {
			rowName := row[0]
			_, _ = dt.rowNames.Set(rowIndex, safeRowName(dt, rowName))
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

// SQL read helpers (ReadSQL, ReadSQLContext, ReadSQLStream) live in
// datatable_from_sql.go.
