package csvxl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/xuri/excelize/v2"
)

// CSVDirToExcel converts every CSV file directly inside dir, those whose name
// ends in ".csv", into one Excel workbook saved at output, one sheet per file
// named after the file. encoding works as it does for CSVToExcel.
func CSVDirToExcel(dir string, output string, encoding ...string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.csv"))
	if err != nil {
		return fmt.Errorf("failed to list CSV files in %s: %w", dir, err)
	}

	var csvFiles []string
	csvFiles = append(csvFiles, files...)

	return CSVToExcel(csvFiles, nil, output, encoding...)
}

// EachCsvToOneExcel converts every CSV file in a directory into one workbook.
//
// Deprecated: use CSVDirToExcel, which is the same function. Removed in the
// release after the one that deprecated it.
func EachCsvToOneExcel(dir string, output string, encoding ...string) error {
	return CSVDirToExcel(dir, output, encoding...)
}

// ExcelDirToCSV writes every sheet of every Excel workbook directly inside
// dir, those whose name ends in ".xlsx", as CSV files in outputDir. Each file
// is named after its workbook and sheet, as in "Book_Sheet.csv". The options
// work as they do for ExcelToCSV; Sheets applies to every workbook.
func ExcelDirToCSV(dir string, outputDir string, options ...ExcelToCSVOptions) error {
	opts, err := oneExcelToCSVOptions(options)
	if err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.xlsx"))
	if err != nil {
		return fmt.Errorf("failed to list Excel files in %s: %w", dir, err)
	}

	for _, excelFile := range files {
		if err := excelFileToCsv(excelFile, outputDir, opts); err != nil {
			return err
		}
	}

	return nil
}

// EachExcelToCsv writes every sheet of every workbook in a directory as CSV.
//
// Deprecated: use ExcelDirToCSV, which is the same function. Removed in the
// release after the one that deprecated it.
func EachExcelToCsv(dir string, outputDir string, options ...ExcelToCsvOptions) error {
	return ExcelDirToCSV(dir, outputDir, options...)
}

// excelFileToCsv writes every sheet of one workbook as a CSV file and closes
// the workbook before returning, on every path.
func excelFileToCsv(excelFile, outputDir string, opts ExcelToCSVOptions) error {
	f, err := excelize.OpenFile(excelFile, insyra.ExcelReadOptions())
	if err != nil {
		return fmt.Errorf("failed to open Excel file %s: %w", excelFile, err)
	}
	defer func() { _ = f.Close() }()

	excelFileName := strings.TrimSuffix(filepath.Base(excelFile), ".xlsx")

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
		}
	}

	sheets, err := selectSheets(excelFile, f.GetSheetList(), opts.Sheets)
	if err != nil {
		return err
	}
	for _, sheet := range sheets {
		outputCsv, err := safeSheetCSVPath(outputDir, sheet, excelFileName+"_"+sheet+".csv")
		if err != nil {
			return err
		}
		if err := saveSheetAsCsv(f, sheet, outputCsv, opts.AllowFormulas); err != nil {
			return fmt.Errorf("failed to save sheet %s as CSV: %w", sheet, err)
		}
	}

	insyra.LogInfo("csvxl", "excelFileToCsv", "Successfully converted %d sheets to CSV files in %s.", len(sheets), outputDir)
	return nil
}
