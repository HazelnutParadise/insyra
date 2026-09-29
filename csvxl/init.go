// Package csvxl converts between CSV files and Excel workbooks.
//
// CSVToExcel puts CSV files into one workbook, one sheet each, and
// AppendCSVToExcel adds them to an existing workbook. ExcelToCSV writes each
// sheet of a workbook as a CSV file. CSVDirToExcel and ExcelDirToCSV do the
// same for every file in a directory, and ReadCSVToString reads a CSV file as
// UTF-8 text.
//
// A CSV file's encoding is detected unless one is given. Any encoding the core
// CSV readers accept can be named, such as UTF8, Big5 or "windows-1252", and a
// name no decoder handles is an error.
//
// Example:
//
//	// Convert CSV files to one workbook, detecting their encodings
//	if err := csvxl.CSVToExcel([]string{"file1.csv", "file2.csv"}, nil, "output.xlsx"); err != nil {
//		// handle the error
//	}
//
//	// Read a Big5 CSV file as UTF-8 text
//	text, err := csvxl.ReadCSVToString("big5.csv", csvxl.Big5)
package csvxl

func init() {}
