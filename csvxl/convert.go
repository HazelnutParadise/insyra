package csvxl

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// Convert multiple CSV files to an Excel file, supporting custom sheet names.
// If the sheet name is not specified, the file name of the CSV file will be used.
func CsvToExcel(csvFiles []string, sheetNames []string, output string) {
	f := excelize.NewFile()
	failedFiles := 0

	for idx, csvFile := range csvFiles {
		// 如果提供了自訂工作表名稱，則使用它，否則使用 CSV 檔案的名稱
		sheetName := getSheetName(csvFile, sheetNames, idx)

		// 第一個工作表重命名，而不是創建新工作表
		if idx == 0 {
			f.SetSheetName(f.GetSheetName(0), sheetName)
		} else {
			f.NewSheet(sheetName)
		}

		err := addCsvSheet(f, sheetName, csvFile)
		if err != nil {
			insyra.LogWarning("csvxl.CsvToExcel(): Failed to add CSV sheet %s: %v", csvFile, err)
			failedFiles++
			continue
		}
	}

	if err := f.SaveAs(output); err != nil {
		insyra.LogWarning("csvxl.CsvToExcel(): Failed to save Excel file %s: %v", output, err)
		return
	}

	insyra.LogInfo("csvxl.CsvToExcel(): Successfully converted %d CSV files to Excel file %s. %d files failed.", len(csvFiles)-failedFiles, output, failedFiles)
}

// Append CSV files to an existing Excel file, supporting custom sheet names.
// If the sheet name is not specified, the file name of the CSV file will be used.
// If the sheet is exists, it will be overwritten.
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existing string) {
	f, err := excelize.OpenFile(existing)
	if err != nil {
		insyra.LogWarning("csvxl.AppendCsvToExcel(): Failed to open Excel file %s: %v", existing, err)
		return
	}

	failedFiles := 0

	for idx, csvFile := range csvFiles {
		// 如果提供了自訂工作表名稱，則使用它，否則使用 CSV 檔案的名稱
		sheetName := getSheetName(csvFile, sheetNames, idx)

		f.NewSheet(sheetName)

		err := addCsvSheet(f, sheetName, csvFile)
		if err != nil {
			insyra.LogWarning("csvxl.AppendCsvToExcel(): Failed to add CSV sheet %s: %v", csvFile, err)
			failedFiles++
			continue
		}
	}

	if err := f.SaveAs(existing); err != nil {
		insyra.LogWarning("csvxl.AppendCsvToExcel(): Failed to save Excel file %s: %v", existing, err)
		return
	}

	insyra.LogInfo("csvxl.AppendCsvToExcel(): Successfully appended %d CSV files to Excel file %s. %d files failed.", len(csvFiles)-failedFiles, existing, failedFiles)
}

// 私有函數：將 CSV 數據加入 Excel 的指定工作表，並處理非 UTF-8 編碼
func addCsvSheet(f *excelize.File, sheetName, csvFile string) error {
	file, err := os.Open(csvFile)
	if err != nil {
		return fmt.Errorf("failed to open CSV file %s: %v", csvFile, err)
	}
	defer file.Close()

	// 嘗試使用 GBK 編碼進行轉換，處理非 UTF-8 字符
	reader := transform.NewReader(file, traditionalchinese.Big5.NewDecoder())
	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV file %s: %v", csvFile, err)
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
