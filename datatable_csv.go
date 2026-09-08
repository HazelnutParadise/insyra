package insyra

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

// ToCSV converts the DataTable to CSV format and writes it to the provided file path.
// The function accepts two parameters:
// - filePath: the file path to write the CSV file to
// - setRowNamesToFirstCol: if true, the first column will be used as row names
// - setColNamesToFirstRow: if true, the first row will be used as column names
func (dt *DataTable) ToCSV(filePath string, setRowNamesToFirstCol bool, setColNamesToFirstRow bool, includeBOM bool) error {
	return dt.ToCSVWithOptions(filePath, CSVWriteOptions{
		SetRowNamesToFirstCol: setRowNamesToFirstCol,
		SetColNamesToFirstRow: setColNamesToFirstRow,
		IncludeBOM:            includeBOM,
	})
}

// CSVWriteOptions controls how ToCSVWithOptions writes a table. The zero value
// writes the data as-is, exactly like ToCSV(path, false, false, false).
type CSVWriteOptions struct {
	// SetRowNamesToFirstCol writes the row names as the first column.
	SetRowNamesToFirstCol bool
	// SetColNamesToFirstRow writes the column names as the first row.
	SetColNamesToFirstRow bool
	// IncludeBOM writes a UTF-8 byte-order mark, which some spreadsheet
	// programs need to read the file as UTF-8.
	IncludeBOM bool
	// SanitizeFormulas guards against CSV formula injection: a spreadsheet
	// opening the file executes a cell that begins with =, +, - or @, so a
	// value that came from an untrusted source can run there. With this set,
	// such a cell is prefixed with a single quote, which spreadsheets treat as
	// "this is text".
	//
	// It is off by default because it changes the value written: a table saved
	// with it on and read back is not identical to the original. Turn it on
	// when the file is meant to be opened in a spreadsheet and the data is not
	// wholly your own.
	SanitizeFormulas bool
}

// ToCSVWithOptions writes the DataTable as CSV using opts. See ToCSV for the
// file-level guarantees: the data goes to a temporary file that is renamed
// into place, so a failure never leaves a truncated file behind.
func (dt *DataTable) ToCSVWithOptions(filePath string, opts CSVWriteOptions) error {
	return writeFileAtomically(filePath, func(w io.Writer) error {
		return dt.writeCSV(w, opts)
	})
}

// sanitizeCSVFormula prefixes a value a spreadsheet would execute with a
// single quote. Only leading =, +, - and @ (and the whitespace a spreadsheet
// skips before them) start a formula.
func sanitizeCSVFormula(s string) string {
	trimmed := strings.TrimLeft(s, " \t\r\n")
	if trimmed == "" {
		return s
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + s
	}
	return s
}

// writeCSV streams the table as CSV to w. Every write error, including the
// one csv.Writer only reports at Flush, is returned.
func (dt *DataTable) writeCSV(w io.Writer, opts CSVWriteOptions) error {
	setRowNamesToFirstCol := opts.SetRowNamesToFirstCol
	setColNamesToFirstRow := opts.SetColNamesToFirstRow
	// 寫入 UTF-8 BOM
	if opts.IncludeBOM {
		if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			return err
		}
	}

	writer := csv.NewWriter(w)

	var maxLength int
	var columns []*DataList
	var columnNames []string

	var err2 *error
	dt.AtomicDo(func(dt *DataTable) {
		maxLength = dt.getMaxColLength()
		columns = make([]*DataList, len(dt.columns))
		copy(columns, dt.columns)
		columnNames = make([]string, len(dt.columns))
		for i, column := range dt.columns {
			columnNames[i] = column.name
		}

		// Write column names as the first row if setColNamesToFirstRow is true
		if setColNamesToFirstRow {
			var header []string
			if setRowNamesToFirstCol {
				header = append(header, "") // Leave the first cell empty for row names
			}
			header = append(header, columnNames...)
			if err := writer.Write(header); err != nil {
				err2 = &err
				return
			}
		}

		// Write the data rows
		for rowIndex := 0; rowIndex < maxLength; rowIndex++ {
			var record []string
			if setRowNamesToFirstCol {
				rowName, _ := dt.GetRowNameByIndex(rowIndex)
				record = append(record, rowName)
			}
			for _, column := range columns {
				if rowIndex < len(column.data) {
					value := column.data[rowIndex]
					switch v := value.(type) {
					case nil:
						record = append(record, "")
					case time.Time:
						// RFC 3339 is the first layout ParseDates tries, so a
						// table written here reads back as the same instants.
						record = append(record, v.Format(time.RFC3339Nano))
					default:
						cell := fmt.Sprintf("%v", value)
						if opts.SanitizeFormulas {
							cell = sanitizeCSVFormula(cell)
						}
						record = append(record, cell)
					}
				} else {
					record = append(record, "")
				}
			}
			if err := writer.Write(record); err != nil {
				err2 = &err
				return
			}
		}
	})
	if err2 != nil {
		return *err2
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return nil
}
