package insyra

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

// ToCSV converts the DataTable to CSV format and writes it to the provided file path.
// The function accepts two parameters:
// - filePath: the file path to write the CSV file to
// - setRowNamesToFirstCol: if true, the first column will be used as row names
// - setColNamesToFirstRow: if true, the first row will be used as column names
func (dt *DataTable) ToCSV(filePath string, setRowNamesToFirstCol bool, setColNamesToFirstRow bool, includeBOM bool) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 寫入 UTF-8 BOM
	if includeBOM {
		if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			return err
		}
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	dt.mu.Lock()
	maxLength := dt.getMaxColLength()

	// Write column names as the first row if setColNamesToFirstRow is true
	if setColNamesToFirstRow {
		var header []string
		if setRowNamesToFirstCol {
			header = append(header, "") // Leave the first cell empty for row names
		}
		for _, column := range dt.columns {
			header = append(header, column.name)
		}
		if err := writer.Write(header); err != nil {
			return err
		}
	}

	dt.mu.Unlock()

	// Write the data rows
	for rowIndex := 0; rowIndex < maxLength; rowIndex++ {
		var record []string
		if setRowNamesToFirstCol {
			rowName := dt.GetRowNameByIndex(rowIndex)
			record = append(record, rowName)
		}

		for _, column := range dt.columns {
			if rowIndex < len(column.data) {
				record = append(record, fmt.Sprintf("%v", column.data[rowIndex]))
			} else {
				record = append(record, "")
			}
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// LoadFromCSV loads a CSV file into a DataTable, with options to set the first column as row names
// and the first row as column names.
func (dt *DataTable) LoadFromCSV(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return err
	}

	dt.columns = []*DataList{}
	dt.columnIndex = make(map[string]int)
	dt.rowNames = make(map[string]int)

	if len(rows) == 0 {
		return nil // 空的CSV
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

	return nil
}
