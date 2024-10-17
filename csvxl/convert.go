package csvxl

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/Go-Utils/sliceutil"
	"github.com/HazelnutParadise/insyra"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// CsvEncoding Options
const (
	UTF8 = "utf-8"
	Big5 = "big5"
)

// Convert multiple CSV files to an Excel file, supporting custom sheet names.
// If the sheet name is not specified, the file name of the CSV file will be used.
func CsvToExcel(csvFiles []string, sheetNames []string, output string, csvEncoding ...string) {
	encoding := "utf-8"
	if len(csvEncoding) == 1 {
		encoding = csvEncoding[0]
	} else if len(csvEncoding) > 1 {
		insyra.LogWarning("csvxl.CsvToExcel: Too many arguments for csvEncoding")
		return
	}

	f := excelize.NewFile()
	failedFiles := 0

	for idx, csvFile := range csvFiles {
		if !strings.HasSuffix(csvFile, ".csv") {
			csvFile += ".csv"
		}

		// 如果提供了自訂工作表名稱，則使用它，否則使用 CSV 檔案的名稱
		sheetName := getSheetName(csvFile, sheetNames, idx)

		// 第一個工作表重命名，而不是創建新工作表
		if idx == 0 {
			err := f.SetSheetName(f.GetSheetName(0), sheetName)
			if err != nil {
				insyra.LogWarning("csvxl.CsvToExcel: Failed to set sheet name %s: %v", sheetName, err)
				return
			}
		} else {
			_, err := f.NewSheet(sheetName)
			if err != nil {
				insyra.LogWarning("csvxl.CsvToExcel: Failed to create new sheet %s: %v", sheetName, err)
				return
			}
		}

		err := addCsvSheet(f, sheetName, csvFile, encoding)
		if err != nil {
			insyra.LogWarning("csvxl.CsvToExcel: Failed to add CSV sheet %s: %v", csvFile, err)
			failedFiles++
			continue
		}
	}

	if err := f.SaveAs(output); err != nil {
		insyra.LogWarning("csvxl.CsvToExcel: Failed to save Excel file %s: %v", output, err)
		return
	}

	insyra.LogInfo("csvxl.CsvToExcel: Successfully converted %d CSV files to Excel file %s. %d files failed.", len(csvFiles)-failedFiles, output, failedFiles)
}

// Append CSV files to an existing Excel file, supporting custom sheet names.
// If the sheet name is not specified, the file name of the CSV file will be used.
// If the sheet is exists, it will be overwritten.
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string) {
	encoding := "utf-8"
	if len(csvEncoding) == 1 {
		encoding = csvEncoding[0]
	} else if len(csvEncoding) > 1 {
		insyra.LogWarning("csvxl.AppendCsvToExcel: Too many arguments for csvEncoding")
		return
	}

	f, err := excelize.OpenFile(existingFile)
	if err != nil {
		insyra.LogWarning("csvxl.AppendCsvToExcel: Failed to open Excel file %s: %v", existingFile, err)
		return
	}

	failedFiles := 0

	for idx, csvFile := range csvFiles {
		if !strings.HasSuffix(csvFile, ".csv") {
			csvFile += ".csv"
		}

		// 如果提供了自訂工作表名稱，則使用它，否則使用 CSV 檔案的名稱
		sheetName := getSheetName(csvFile, sheetNames, idx)

		f.NewSheet(sheetName)

		err := addCsvSheet(f, sheetName, csvFile, encoding)
		if err != nil {
			insyra.LogWarning("csvxl.AppendCsvToExcel: Failed to add CSV sheet %s: %v", csvFile, err)
			failedFiles++
			continue
		}
	}

	if err := f.SaveAs(existingFile); err != nil {
		insyra.LogWarning("csvxl.AppendCsvToExcel: Failed to save Excel file %s: %v", existingFile, err)
		return
	}

	insyra.LogInfo("csvxl.AppendCsvToExcel: Successfully appended %d CSV files to Excel file %s. %d files failed.", len(csvFiles)-failedFiles, existingFile, failedFiles)
}

// ExcelToCsv splits an Excel file into multiple CSV files, one per sheet.
// If customNames is provided, it uses them as CSV filenames; otherwise, it uses the sheet names.
func ExcelToCsv(excelFile string, outputDir string, csvNames []string, onlyContainSheets ...string) {
	f, err := excelize.OpenFile(excelFile)
	if err != nil {
		insyra.LogWarning("csvxl.ExcelToCsv: Failed to open Excel file %s: %v", excelFile, err)
		return
	}

	sheets := f.GetSheetList()
	nameIdx := 0
	for _, sheet := range sheets {
		if len(onlyContainSheets) > 0 && !sliceutil.Contains(onlyContainSheets, sheet) {
			continue
		}

		csvName := sheet + ".csv"
		if len(csvNames) > nameIdx && csvNames[nameIdx] != "" {
			if strings.HasSuffix(csvNames[nameIdx], ".csv") {
				csvName = csvNames[nameIdx]
			} else {
				csvName = csvNames[nameIdx] + ".csv"
			}
			nameIdx++
		}

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

// ===============================

// saveSheetAsCsv saves a specific sheet in an Excel file as a CSV file.
func saveSheetAsCsv(f *excelize.File, sheet string, outputCsv string) error {
	file, err := os.Create(outputCsv)
	if err != nil {
		return fmt.Errorf("failed to create CSV file %s: %v", outputCsv, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("failed to read rows from sheet %s: %v", sheet, err)
	}

	for _, row := range rows {
		err := writer.Write(row)
		if err != nil {
			return fmt.Errorf("failed to write row to CSV file %s: %v", outputCsv, err)
		}
	}

	return nil
}

// 私有函數：將 CSV 數據加入 Excel 的指定工作表，並處理非 UTF-8 編碼
func addCsvSheet(f *excelize.File, sheetName, csvFile string, encoding string) error {
	file, err := os.Open(csvFile)
	if err != nil {
		return fmt.Errorf("failed to open CSV file %s: %v", csvFile, err)
	}
	defer file.Close()

	var records [][]string

	switch encoding {
	case "big5":
		reader := transform.NewReader(file, traditionalchinese.Big5.NewDecoder())
		csvReader := csv.NewReader(reader)
		records, err = csvReader.ReadAll()
		if err != nil {
			return fmt.Errorf("failed to read CSV file %s: %v", csvFile, err)
		}
	default:
		reader := csv.NewReader(file)
		records, err = reader.ReadAll()
		if err != nil {
			return fmt.Errorf("failed to read CSV file %s: %v", csvFile, err)
		}
	}

	for rowIdx, record := range records {
		for colIdx, cell := range record {
			cellAddr, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			f.SetCellValue(sheetName, cellAddr, cell)
		}
	}

	return nil
}

// 私有函數：取得工作表名稱，如果提供了自訂名稱則使用，否則使用 CSV 檔案名稱
func getSheetName(csvFile string, sheetNames []string, idx int) string {
	if len(sheetNames) > idx && sheetNames[idx] != "" {
		return sheetNames[idx]
	}
	return strings.TrimSuffix(filepath.Base(csvFile), filepath.Ext(csvFile))
}
