package insyra

import (
	"encoding/csv"
	"fmt"
	"os"
)

// ToCSV converts the DataTable to CSV format and writes it to the provided file path.
// The function accepts two parameters:
// - setColumnNamesToFirstRow: if true, the column names will be included as the first row in the CSV.
// - setRowNamesToFirstColumn: if true, the row names will be included as the first column in the CSV.
func (dt *DataTable) ToCSV(filePath string, setColumnNamesToFirstRow bool, setRowNamesToFirstColumn bool) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	maxLength := dt.getMaxColumnLength()

	// Write column names as the first row if setColumnNamesToFirstRow is true
	if setColumnNamesToFirstRow {
		var header []string
		if setRowNamesToFirstColumn {
			header = append(header, "") // Leave the first cell empty for row names
		}
		sortedColumnNames := dt.getSortedColumnNames()
		for _, colName := range sortedColumnNames {
			column := dt.GetColumn(colName)
			header = append(header, column.name)
		}
		if err := writer.Write(header); err != nil {
			return err
		}
	}

	// Write the data rows
	for rowIndex := 0; rowIndex < maxLength; rowIndex++ {
		var record []string
		if setRowNamesToFirstColumn {
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
