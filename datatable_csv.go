package insyra

import (
	"encoding/csv"
	"fmt"
	"os"
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
				value := column.data[rowIndex]
				if value == nil {
					record = append(record, "")
				} else {
					record = append(record, fmt.Sprintf("%v", value))
				}
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
