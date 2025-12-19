package csvxl

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/Go-Utils/sliceutil"
	"github.com/HazelnutParadise/insyra"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// CsvEncoding Options
const (
	UTF8 = "utf-8"
	Big5 = "big5"
	Auto = "auto"
)

// Convert multiple CSV files to an Excel file, supporting custom sheet names.
// If the sheet name is not specified, the file name of the CSV file will be used.
// If csvEncoding is not specified, auto-detection will be used.
func CsvToExcel(csvFiles []string, sheetNames []string, output string, csvEncoding ...string) error {
	encoding := Auto // Default to auto-detection
	if len(csvEncoding) == 1 {
		encoding = csvEncoding[0]
	} else if len(csvEncoding) > 1 {
		return fmt.Errorf("too many arguments for csvEncoding")
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
				return fmt.Errorf("failed to set sheet name %s: %v", sheetName, err)
			}
		} else {
			_, err := f.NewSheet(sheetName)
			if err != nil {
				return fmt.Errorf("failed to create new sheet %s: %v", sheetName, err)
			}
		}

		err := addCsvSheet(f, sheetName, csvFile, encoding)
		if err != nil {
			failedFiles++
			continue
		}
	}

	if err := f.SaveAs(output); err != nil {
		return fmt.Errorf("failed to save Excel file %s: %v", output, err)
	}

	if failedFiles > 0 {
		return fmt.Errorf("%d files failed to convert", failedFiles)
	}

	insyra.LogInfo("csvxl", "CsvToExcel", "Successfully converted %d CSV files to Excel file %s. %d files failed.", len(csvFiles)-failedFiles, output, failedFiles)
	return nil
}

// Append CSV files to an existing Excel file, supporting custom sheet names.
// If the sheet name is not specified, the file name of the CSV file will be used.
// If the sheet is exists, it will be overwritten.
// If csvEncoding is not specified, auto-detection will be used.
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string) error {
	encoding := Auto // Default to auto-detection
	if len(csvEncoding) == 1 {
		encoding = csvEncoding[0]
	} else if len(csvEncoding) > 1 {
		return fmt.Errorf("too many arguments for csvEncoding")
	}

	f, err := excelize.OpenFile(existingFile)
	if err != nil {
		return fmt.Errorf("failed to open Excel file %s: %v", existingFile, err)
	}

	failedFiles := 0

	for idx, csvFile := range csvFiles {
		if !strings.HasSuffix(csvFile, ".csv") {
			csvFile += ".csv"
		}

		// 如果提供了自訂工作表名稱，則使用它，否則使用 CSV 檔案的名稱
		sheetName := getSheetName(csvFile, sheetNames, idx)

		_, err := f.NewSheet(sheetName)
		if err != nil {
			return fmt.Errorf("failed to create new sheet %s: %v", sheetName, err)
		}

		err = addCsvSheet(f, sheetName, csvFile, encoding)
		if err != nil {
			failedFiles++
			continue
		}
	}

	if err := f.SaveAs(existingFile); err != nil {
		return fmt.Errorf("failed to save Excel file %s: %v", existingFile, err)
	}

	if failedFiles > 0 {
		return fmt.Errorf("%d files failed to append", failedFiles)
	}

	insyra.LogInfo("csvxl", "AppendCsvToExcel", "Successfully appended %d CSV files to Excel file %s. %d files failed.", len(csvFiles)-failedFiles, existingFile, failedFiles)
	return nil
}

// ExcelToCsv splits an Excel file into multiple CSV files, one per sheet.
// If customNames is provided, it uses them as CSV filenames; otherwise, it uses the sheet names.
func ExcelToCsv(excelFile string, outputDir string, csvNames []string, onlyContainSheets ...string) error {
	f, err := excelize.OpenFile(excelFile)
	if err != nil {
		return fmt.Errorf("failed to open Excel file %s: %v", excelFile, err)
	}

	// Check if output directory exists, if not create it
	// todo: 移到後面
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %v", outputDir, err)
		}
	}

	sheetsInXlsx := f.GetSheetList()
	// Determine the sheets to process. If `onlyContainSheets` is provided,
	// filter it against `sheetsInXlsx` to only process existing sheets.
	var sheetsToProcess []string
	if len(onlyContainSheets) > 0 {
		for _, s := range onlyContainSheets {
			if sliceutil.Contains(sheetsInXlsx, s) {
				sheetsToProcess = append(sheetsToProcess, s)
			}
		}
	} else {
		sheetsToProcess = sheetsInXlsx
	}

	numSheets := len(sheetsToProcess)
	for idx, sheet := range sheetsToProcess {
		csvName := sheet + ".csv"
		if len(csvNames) > idx && csvNames[idx] != "" {
			if strings.HasSuffix(csvNames[idx], ".csv") {
				csvName = csvNames[idx]
			} else {
				csvName = csvNames[idx] + ".csv"
			}
		}

		outputCsv := filepath.Join(outputDir, csvName)
		err := saveSheetAsCsv(f, sheet, outputCsv)
		if err != nil {
			return fmt.Errorf("failed to save sheet %s as CSV: %v", sheet, err)
		}
	}

	insyra.LogInfo("csvxl", "ExcelToCsv", "Successfully converted %d sheets to CSV files in %s.", numSheets, outputDir)
	return nil
}

// ===============================

// saveSheetAsCsv saves a specific sheet in an Excel file as a CSV file.
func saveSheetAsCsv(f *excelize.File, sheetName string, outputCsvName string) error {
	file, err := os.Create(outputCsvName)
	if err != nil {
		return fmt.Errorf("failed to create CSV file %s: %v", outputCsvName, err)
	}
	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("failed to read rows from sheet %s: %v", sheetName, err)
	}

	for rowIdx, row := range rows {
		// Check if the row is visible (not filtered out)
		visible, err := f.GetRowVisible(sheetName, rowIdx+1) // rowIdx is 0-based, GetRowVisible is 1-based
		if err != nil {
			return fmt.Errorf("failed to check visibility of row %d in sheet %s: %v", rowIdx+1, sheetName, err)
		}
		if visible {
			err := writer.Write(row)
			if err != nil {
				return fmt.Errorf("failed to write row to CSV file %s: %v", outputCsvName, err)
			}
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
	defer func() { _ = file.Close() }()

	var records [][]string

	// Auto-detect encoding if specified
	if encoding == Auto {
		detectedEncoding, err := insyra.DetectEncoding(csvFile)
		if err != nil {
			// Propagate the detection error instead of silently falling back
			return fmt.Errorf("failed to auto-detect encoding for %s: %v", csvFile, err)
		}
		encoding = strings.ToLower(detectedEncoding)
		insyra.LogInfo("csvxl", "addCsvSheet", "Auto-detected encoding %s for file %s", encoding, csvFile)
	}

	// Ensure we start reading from the beginning of the file
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file %s: %v", csvFile, err)
	}

	var reader io.Reader
	switch {
	case strings.Contains(encoding, "big5"):
		reader = transform.NewReader(file, traditionalchinese.Big5.NewDecoder())
	case strings.Contains(encoding, "gb") || strings.Contains(encoding, "gb-"):
		reader = transform.NewReader(file, simplifiedchinese.GB18030.NewDecoder())
	case strings.Contains(encoding, "utf-16") || strings.Contains(encoding, "utf16"):
		reader = transform.NewReader(file, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())
	default:
		reader = file
	}

	csvReader := csv.NewReader(reader)
	records, err = csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV file %s: %v", csvFile, err)
	}

	// Trim UTF-8 BOM if present
	if len(records) > 0 && len(records[0]) > 0 {
		records[0][0] = strings.TrimPrefix(records[0][0], "\uFEFF")
	}

	for rowIdx, record := range records {
		for colIdx, cell := range record {
			cellAddr, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			err := f.SetCellValue(sheetName, cellAddr, cell)
			if err != nil {
				return fmt.Errorf("failed to set cell value %s: %v", cellAddr, err)
			}
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
