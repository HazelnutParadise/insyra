package insyra

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
)

// ReadCSV loads a CSV file into a DataTable, with options to set the first column as row names
// and the first row as column names.
func ReadCSV(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error) {
	dt := NewDataTable()

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
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
