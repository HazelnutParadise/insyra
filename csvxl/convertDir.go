package csvxl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/xuri/excelize/v2"
)

// EachCsvToOneExcel converts each CSV file in the given directory to an Excel file.
// The output Excel file will be saved in the given output path.
// If encoding is not specified, auto-detection will be used.
func EachCsvToOneExcel(dir string, output string, encoding ...string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.csv"))
	if err != nil {
		return fmt.Errorf("failed to list CSV files in %s: %w", dir, err)
	}

	var csvFiles []string
	csvFiles = append(csvFiles, files...)

	return CsvToExcel(csvFiles, nil, output, encoding...)
}

// EachExcelToCsv converts each Excel file in the given directory to CSV files.
// The output CSV files will be saved in the given output directory.
// The CSV files will be named as the Excel file name plus the sheet name plus ".csv",
// for example, "ExcelFileName_SheetName.csv".
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
		outputCsv := filepath.Join(outputDir, excelFileName+"_"+sheet+".csv")
		if err := saveSheetAsCsv(f, sheet, outputCsv); err != nil {
			return fmt.Errorf("failed to save sheet %s as CSV: %w", sheet, err)
		}
	}

	insyra.LogInfo("csvxl", "EachExcelToCsv", "Successfully converted %d sheets to CSV files in %s.", len(sheets), outputDir)
	return nil
}
