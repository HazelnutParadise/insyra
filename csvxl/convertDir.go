package csvxl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// EachCsvToOneExcel converts each CSV file in the given directory to an Excel file.
// The output Excel file will be saved in the given output path.
// If encoding is not specified, auto-detection will be used.
//
// Deprecated: use CSVDirToExcel, which has the same arguments and behavior. Removed in 0.4.
func EachCsvToOneExcel(dir string, output string, encoding ...string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.csv"))
	if err != nil {
		return fmt.Errorf("failed to list CSV files in %s: %w", dir, err)
	}

	var csvFiles []string
	csvFiles = append(csvFiles, files...)

	return CsvToExcel(csvFiles, nil, output, encoding...)
}

// CSVDirToExcel converts every CSV in a directory to one workbook.
func CSVDirToExcel(dir string, output string, encoding ...string) error {
	return EachCsvToOneExcel(dir, output, encoding...)
}

// EachExcelToCsv converts each Excel file in the given directory to CSV files.
// The output CSV files will be saved in the given output directory.
// The CSV files will be named as the Excel file name plus the sheet name plus ".csv",
// for example, "ExcelFileName_SheetName.csv".
//
// Deprecated: use ExcelDirToCSV. Worksheet selection moves to ExcelToCSVOptions.Sheets;
// the new function guards formula-like text by default. Removed in 0.4.
func EachExcelToCsv(dir string, outputDir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.xlsx"))
	if err != nil {
		return fmt.Errorf("failed to list Excel files in %s: %w", dir, err)
	}

	for _, excelFile := range files {
		if err := excelFileToCsv(excelFile, outputDir); err != nil {
			return err
		}
	}

	return nil
}

// ExcelDirToCSV writes every sheet of every .xlsx workbook in dir as a CSV
// file in outputDir, sheet S of Book.xlsx as Book_S.csv. See
// ExcelToCSVOptions for choosing sheets and the formula guard; FileNames is
// refused, because one map cannot name the files of several workbooks.
//
// Every file name of every workbook is checked before the first CSV is
// written: a name that would leave outputDir, and two sheets that would be
// written to one file, even from different workbooks and compared without
// regard to case, are an error and nothing is written.
func ExcelDirToCSV(dir, outputDir string, options ...ExcelToCSVOptions) error {
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

// planExcelFileCSVs returns where each selected sheet of one workbook goes.
func planExcelFileCSVs(excelFile, outputDir string, opts ExcelToCSVOptions) ([]csvTarget, error) {
	f, err := excelize.OpenFile(excelFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file %s: %w", excelFile, err)
	}
	defer func() { _ = f.Close() }()
	base := strings.TrimSuffix(filepath.Base(excelFile), ".xlsx")
	sheets, err := selectSheets(excelFile, f.GetSheetList(), opts.Sheets)
	if err != nil {
		return nil, err
	}
	targets := make([]csvTarget, 0, len(sheets))
	for _, sheet := range sheets {
		path, err := safeSheetCSVPath(outputDir, sheet, base+"_"+sheet+".csv")
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
	f, err := excelize.OpenFile(excelFile)
	if err != nil {
		return fmt.Errorf("failed to open Excel file %s: %w", excelFile, err)
	}
	defer func() { _ = f.Close() }()
	return writeCSVTargets(f, outputDir, targets, allowFormulas)
}

// excelFileToCsv writes every sheet of one workbook as a CSV file and closes
// the workbook before returning, on every path.
func excelFileToCsv(excelFile, outputDir string) error {
	f, err := excelize.OpenFile(excelFile)
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

	sheets := f.GetSheetList()
	for _, sheet := range sheets {
		outputCsv, err := safeSheetCSVPath(outputDir, sheet, excelFileName+"_"+sheet+".csv")
		if err != nil {
			return err
		}
		if err := saveSheetAsCsv(f, sheet, outputCsv); err != nil {
			return fmt.Errorf("failed to save sheet %s as CSV: %w", sheet, err)
		}
	}

	return nil
}
