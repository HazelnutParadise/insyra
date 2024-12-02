package csvxl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/xuri/excelize/v2"
)

// EachCsvToOneExcel converts each CSV file in the given directory to an Excel file.
// The output Excel file will be saved in the given output path.
func EachCsvToOneExcel(dir string, output string, encoding ...string) {
	files, err := filepath.Glob(filepath.Join(dir, "*.csv"))
	if err != nil {
		insyra.LogWarning("csvxl.EachCsvToOneExcel: %s", err)
		return
	}

	var csvFiles []string
	csvFiles = append(csvFiles, files...)

	CsvToExcel(csvFiles, nil, output, encoding...)
}

// EachExcelToCsv converts each Excel file in the given directory to CSV files.
// The output CSV files will be saved in the given output directory.
// The CSV files will be named as the Excel file name plus the sheet name plus ".csv",
// for example, "excelFileName_SheetName.csv".
func EachExcelToCsv(dir string, outputDir string) {
	files, err := filepath.Glob(filepath.Join(dir, "*.xlsx"))
	if err != nil {
		insyra.LogWarning("csvxl.EachExcelToCsv: %s", err)
		return
	}

	xlsx2Csv := func(excelFile string, outputDir string) {
		f, err := excelize.OpenFile(excelFile)
		if err != nil {
			insyra.LogWarning("csvxl.ExcelToCsv: Failed to open Excel file %s: %v", excelFile, err)
			return
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
					insyra.LogWarning("csvxl.ExcelToCsv: Failed to create directory %s: %v", outputDir, err)
					return
				}
			}
			outputCsv := filepath.Join(outputDir, csvName)
			err := saveSheetAsCsv(f, sheet, outputCsv)
			if err != nil {
				insyra.LogWarning("csvxl.ExcelToCsv: Failed to save sheet %s as CSV: %v", sheet, err)
				return
			}
		}

		insyra.LogInfo("csvxl.ExcelToCsv: Successfully converted %d sheets to CSV files in %s.", len(sheets), outputDir)
	}

	for _, file := range files {
		xlsx2Csv(file, outputDir)
	}
}
