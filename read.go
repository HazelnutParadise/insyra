package insyra

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
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

// CSVReadOptions configures ReadCSV_FileWithOptions and ReadCSV_StringWithOptions.
// The zero value reproduces the default behavior of ReadCSV_File/ReadCSV_String:
// no row/column names taken from the data, auto-detected encoding, and
// column-level type inference enabled.
type CSVReadOptions struct {
	FirstColToRowNames bool
	FirstRowToColNames bool
	// Encoding applies to file input only; "" or "auto" auto-detects.
	Encoding string
	// RawStrings keeps every cell as its original string and skips column
	// type inference entirely. Use it for data that looks numeric but must
	// not be parsed as numbers (stock IDs like "0050", tax IDs, phone
	// numbers, exact monetary amounts). Empty cells stay "".
	RawStrings bool
}

// ReadCSV_File loads a CSV file into a DataTable, with options to set the first column as row names
// and the first row as column names.
func ReadCSV_File(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool, encoding ...string) (*DataTable, error) {
	opts := CSVReadOptions{
		FirstColToRowNames: setFirstColToRowNames,
		FirstRowToColNames: setFirstRowToColNames,
	}
	if len(encoding) > 0 {
		opts.Encoding = encoding[0]
	}
	return ReadCSV_FileWithOptions(filePath, opts)
}

// ReadCSV_FileWithOptions loads a CSV file into a DataTable according to opts.
func ReadCSV_FileWithOptions(filePath string, opts CSVReadOptions) (*DataTable, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	useEncoding := strings.ToLower(opts.Encoding)
	if useEncoding == "" || useEncoding == "auto" {
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

	return csvRowsToDataTable(rows, opts), nil
}

// csvRowsToDataTable builds a DataTable from parsed CSV rows, applying the
// row/column-name options and (unless opts.RawStrings) column type inference.
func csvRowsToDataTable(rows [][]string, opts CSVReadOptions) *DataTable {
	dt := NewDataTable()
	dt.columns = []*DataList{}
	dt.rowNames = core.NewBiIndex(0)

	if len(rows) == 0 {
		return dt // 空的CSV
	}

	// 處理第一行是否為欄名
	startRow := 0
	if opts.FirstRowToColNames {
		for i, colName := range rows[0] {
			if opts.FirstColToRowNames && i == 0 {
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
			if opts.FirstColToRowNames && i == 0 {
				continue
			}
			column := &DataList{}
			dt.columns = append(dt.columns, column)
		}
	}

	// 處理資料行和是否將第一欄作為行名
	for rowIndex, row := range rows[startRow:] {
		if opts.FirstColToRowNames {
			rowName := row[0]
			_, _ = dt.rowNames.Set(rowIndex, safeRowName(dt, rowName))
			row = row[1:] // 移除第一欄作為行名
		}

		for colIndex, cell := range row {
			if colIndex >= len(dt.columns) {
				continue
			}
			column := dt.columns[colIndex]
			column.data = append(column.data, cell)
		}
	}

	if !opts.RawStrings {
		inferCSVColumnTypes(dt)
	}
	return dt
}

// inferCSVColumnTypes converts each column's raw string cells to a homogeneous
// type using pandas-style, column-level inference:
//   - every cell a valid int64 and no cell empty  → []int64
//   - otherwise every non-empty cell numeric       → []float64 (empty → NaN)
//   - otherwise                                     → cells left as strings
//
// Integer columns become int64 rather than float64 so large-integer columns
// (e.g. 19-digit IDs, nanosecond timestamps) keep full precision instead of
// being rounded to the nearest float64 above 2^53. This matches how
// pandas.read_csv types a column.
func inferCSVColumnTypes(dt *DataTable) {
	for _, col := range dt.columns {
		if len(col.data) == 0 {
			continue
		}
		allInt, allFloat, hasEmpty := true, true, false
		for _, v := range col.data {
			s, ok := v.(string)
			if !ok {
				allInt, allFloat = false, false
				break
			}
			if s == "" {
				hasEmpty = true
				continue
			}
			if _, err := strconv.ParseInt(s, 10, 64); err != nil {
				allInt = false
			}
			if _, err := strconv.ParseFloat(s, 64); err != nil {
				allFloat = false
			}
		}
		switch {
		case allInt && !hasEmpty:
			for i, v := range col.data {
				n, _ := strconv.ParseInt(v.(string), 10, 64)
				col.data[i] = n
			}
		case allFloat:
			for i, v := range col.data {
				s := v.(string)
				if s == "" {
					col.data[i] = math.NaN()
					continue
				}
				f, _ := strconv.ParseFloat(s, 64)
				col.data[i] = f
			}
		}
	}
}

func ReadCSV_String(csvString string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error) {
	return ReadCSV_StringWithOptions(csvString, CSVReadOptions{
		FirstColToRowNames: setFirstColToRowNames,
		FirstRowToColNames: setFirstRowToColNames,
	})
}

// ReadCSV_StringWithOptions loads a CSV string into a DataTable according to opts.
// opts.Encoding is ignored: the input is already a Go string.
func ReadCSV_StringWithOptions(csvString string, opts CSVReadOptions) (*DataTable, error) {
	// Strip a leading UTF-8 BOM so the first header/cell is not corrupted
	// (consistent with the file reader).
	csvString = strings.TrimPrefix(csvString, string([]byte{0xEF, 0xBB, 0xBF}))

	reader := csv.NewReader(strings.NewReader(csvString))
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return csvRowsToDataTable(rows, opts), nil
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

// unmarshalJSONRows decodes JSON bytes into row maps while preserving integer
// precision: numbers are decoded as json.Number (later typed by coerceJSONNumber)
// instead of collapsing to float64. Falls back to a single object.
func unmarshalJSONRows(b []byte) ([]map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var rows []map[string]any
	if err := dec.Decode(&rows); err != nil {
		single := map[string]any{}
		dec2 := json.NewDecoder(bytes.NewReader(b))
		dec2.UseNumber()
		if err2 := dec2.Decode(&single); err2 == nil {
			return []map[string]any{single}, nil
		}
		return nil, err
	}
	return rows, nil
}

// coerceJSONNumber types a json.Number as int64 when it is an integer literal
// (preserving values beyond 2^53) and float64 otherwise.
func coerceJSONNumber(n json.Number) any {
	if i, err := n.Int64(); err == nil {
		return i
	}
	if f, err := n.Float64(); err == nil {
		return f
	}
	return n.String()
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
		r, err := unmarshalJSONRows(v)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
		rows = r
	case string:
		r, err := unmarshalJSONRows([]byte(v))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
		rows = r
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
			r, err := unmarshalJSONRows(b)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal array element: %w", err)
			}
			rows = append(rows, r...)
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input: %w", err)
		}
		r, err := unmarshalJSONRows(b)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal/unmarshal input as JSON: %w", err)
		}
		rows = r
	}

	// Type JSON numbers: an integer literal (25) becomes int64 so large integers
	// keep full precision; a decimal literal (25.5, 25.0) stays float64. JSON
	// authors the int/float distinction, so this is per value (matching how
	// encoding/json + json.Number and Python's json.loads type numbers) and is
	// consistent with ReadCSV loading integer columns as int64.
	for _, row := range rows {
		for k, val := range row {
			if n, ok := val.(json.Number); ok {
				row[k] = coerceJSONNumber(n)
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
