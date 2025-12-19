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
		return fmt.Errorf("failed to list CSV files in %s: %v", dir, err)
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
		return fmt.Errorf("failed to list Excel files in %s: %v", dir, err)
	}

	for _, excelFile := range files {
		f, err := excelize.OpenFile(excelFile)
		if err != nil {
			return fmt.Errorf("failed to open Excel file %s: %v", excelFile, err)
		}

		excelFileName := filepath.Base(excelFile)
		excelFileName = strings.TrimSuffix(excelFileName, ".xlsx")

		sheets := f.GetSheetList()
		for _, sheet := range sheets {
			csvName := excelFileName + "_" + sheet + ".csv"

			// Check if output directory exists, if not create it
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				err := os.MkdirAll(outputDir, os.ModePerm)
				if err != nil {
					return fmt.Errorf("failed to create directory %s: %v", outputDir, err)
				}
			}
			outputCsv := filepath.Join(outputDir, csvName)
			if err := saveSheetAsCsv(f, sheet, outputCsv); err != nil {
				return fmt.Errorf("failed to save sheet %s as CSV: %v", sheet, err)
			}
		}

		insyra.LogInfo("csvxl", "EachCsvToOneExcel", "Successfully converted %d sheets to CSV files in %s.", len(sheets), outputDir)
	}

	return nil
}
