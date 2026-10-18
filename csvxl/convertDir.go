package csvxl

import (
	"fmt"
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

	sheets := make([]SheetFromCSV, len(files))
	for i, file := range files {
		sheets[i].Path = file
	}
	return CSVToExcel(sheets, output, encoding...)
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
// work as they do for ExcelToCSV; Sheets applies to every workbook, and
// FileNames, which cannot name the files of several workbooks, is refused.
//
// Every file name is checked before the first CSV is written, across all the
// workbooks: two sheets that would be written to one file, such as the sheet
// "b_c" of a.xlsx and the sheet "c" of a_b.xlsx, are an error and nothing is
// written.
func ExcelDirToCSV(dir string, outputDir string, options ...ExcelToCSVOptions) error {
	opts, err := oneExcelToCSVOptions(options)
	if err != nil {
		return err
	}
	if len(opts.FileNames) > 0 {
		return fmt.Errorf("ExcelDirToCSV names each file after its workbook and sheet; FileNames applies only to ExcelToCSV")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.xlsx"))
	if err != nil {
		return fmt.Errorf("failed to list Excel files in %s: %w", dir, err)
	}

	var all []csvTarget
	perFile := make([][]csvTarget, len(files))
	for i, excelFile := range files {
		targets, err := planExcelFileCSVs(excelFile, outputDir, opts)
		if err != nil {
			return err
		}
		perFile[i] = targets
		all = append(all, targets...)
	}
	if err := refuseSharedCSVFiles(all); err != nil {
		return err
	}
	for i, excelFile := range files {
		if err := writeExcelFileCSVs(excelFile, outputDir, perFile[i], opts.AllowFormulas); err != nil {
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

// planExcelFileCSVs returns where each selected sheet of one workbook goes.
func planExcelFileCSVs(excelFile, outputDir string, opts ExcelToCSVOptions) ([]csvTarget, error) {
	f, err := excelize.OpenFile(excelFile, insyra.ExcelReadOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file %s: %w", excelFile, err)
	}
	defer func() { _ = f.Close() }()

	excelFileName := strings.TrimSuffix(filepath.Base(excelFile), ".xlsx")
	sheets, err := selectSheets(excelFile, f.GetSheetList(), opts.Sheets)
	if err != nil {
		return nil, err
	}
	targets := make([]csvTarget, 0, len(sheets))
	for _, sheet := range sheets {
		path, err := safeSheetCSVPath(outputDir, sheet, excelFileName+"_"+sheet+".csv")
		if err != nil {
			return nil, err
		}
		targets = append(targets, csvTarget{source: filepath.Base(excelFile) + " sheet " + sheet, sheet: sheet, path: path})
	}
	return targets, nil
}

// writeExcelFileCSVs writes the planned sheets of one workbook and closes the
// workbook before returning, on every path.
func writeExcelFileCSVs(excelFile, outputDir string, targets []csvTarget, allowFormulas bool) error {
	f, err := excelize.OpenFile(excelFile, insyra.ExcelReadOptions())
	if err != nil {
		return fmt.Errorf("failed to open Excel file %s: %w", excelFile, err)
	}
	defer func() { _ = f.Close() }()
	return writeCSVTargets(f, outputDir, targets, allowFormulas)
}
