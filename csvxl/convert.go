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
	insyracsv "github.com/HazelnutParadise/insyra/internal/csv"

	"github.com/xuri/excelize/v2"
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
	defer func() { _ = f.Close() }()
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
				return fmt.Errorf("failed to set sheet name %s: %w", sheetName, err)
			}
		} else {
			_, err := f.NewSheet(sheetName)
			if err != nil {
				return fmt.Errorf("failed to create new sheet %s: %w", sheetName, err)
			}
		}

		err := addCsvSheet(f, sheetName, csvFile, encoding)
		if err != nil {
			failedFiles++
			continue
		}
	}

	if err := f.SaveAs(output); err != nil {
		return fmt.Errorf("failed to save Excel file %s: %w", output, err)
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
		return fmt.Errorf("failed to open Excel file %s: %w", existingFile, err)
	}
	defer func() { _ = f.Close() }()

	failedFiles := 0

	for idx, csvFile := range csvFiles {
		if !strings.HasSuffix(csvFile, ".csv") {
			csvFile += ".csv"
		}

		// 如果提供了自訂工作表名稱，則使用它，否則使用 CSV 檔案的名稱
		sheetName := getSheetName(csvFile, sheetNames, idx)

		// Read the whole CSV before touching its sheet. A CSV that cannot be
		// read is counted as failed and the workbook is saved anyway, so
		// replacing the sheet first would leave it empty in the saved file.
		records, err := readCsvRecords(csvFile, encoding)
		if err != nil {
			failedFiles++
			continue
		}

		if err := replaceSheet(f, sheetName); err != nil {
			return fmt.Errorf("failed to create new sheet %s: %w", sheetName, err)
		}

		if err := writeCsvRecords(f, sheetName, records); err != nil {
			failedFiles++
			continue
		}
	}

	if err := f.SaveAs(existingFile); err != nil {
		return fmt.Errorf("failed to save Excel file %s: %w", existingFile, err)
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
		return fmt.Errorf("failed to open Excel file %s: %w", excelFile, err)
	}
	defer func() { _ = f.Close() }()

	// Check if output directory exists, if not create it
	// todo: 移到後面
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.MkdirAll(outputDir, 0o755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
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
		var outputCsv string
		if len(csvNames) > idx && csvNames[idx] != "" {
			// The caller chose this name, so it is used as given, as it
			// always was; only a name taken from the workbook is checked.
			csvName := csvNames[idx]
			if !strings.HasSuffix(csvName, ".csv") {
				csvName += ".csv"
			}
			outputCsv = filepath.Join(outputDir, csvName)
		} else {
			outputCsv, err = safeSheetCSVPath(outputDir, sheet, sheet+".csv")
			if err != nil {
				return err
			}
		}
		err = saveSheetAsCsv(f, sheet, outputCsv)
		if err != nil {
			return fmt.Errorf("failed to save sheet %s as CSV: %w", sheet, err)
		}
	}

	insyra.LogInfo("csvxl", "ExcelToCsv", "Successfully converted %d sheets to CSV files in %s.", numSheets, outputDir)
	return nil
}

// ===============================

// replaceSheet makes sheetName an empty sheet in f: a new sheet, or an existing
// one of that name deleted and re-created, moved back to where it was, hidden
// again if it was hidden, and given back the workbook's active sheet, so none of
// the old sheet's cells, formulas, hidden rows, row heights, comments,
// hyperlinks, column widths, views or merged ranges survives. excelize refuses
// to delete a workbook's only sheet, so that case goes through a placeholder
// sheet. Sheet names match without regard to case, as in Excel. Names defined
// for a later sheet stay with that sheet; only the names belonging to the
// replaced sheet are deleted with it.
func replaceSheet(f *excelize.File, sheetName string) error {
	idx, err := f.GetSheetIndex(sheetName)
	if err != nil {
		return err
	}
	if idx == -1 {
		_, err = f.NewSheet(sheetName)
		return err
	}
	sheets := f.GetSheetList()
	active := f.GetSheetName(f.GetActiveSheetIndex())
	next := ""
	if idx+1 < len(sheets) {
		next = sheets[idx+1]
	}
	// Whether the sheet is hidden is recorded in the workbook, not in the
	// sheet, so it is kept like the sheet's position.
	state := ""
	if wb := f.WorkBook; wb != nil && idx < len(wb.Sheets.Sheet) {
		state = wb.Sheets.Sheet[idx].State
	}
	placeholder := ""
	if f.SheetCount == 1 {
		placeholder = "__insyra_placeholder__"
		if _, err := f.NewSheet(placeholder); err != nil {
			return err
		}
	}
	if err := f.DeleteSheet(sheetName); err != nil {
		return err
	}
	if _, err := f.NewSheet(sheetName); err != nil {
		return err
	}
	if placeholder != "" {
		if err := f.DeleteSheet(placeholder); err != nil {
			return err
		}
	}
	if next != "" {
		if err := f.MoveSheet(sheetName, next); err != nil {
			return err
		}
		// DeleteSheet moves every name defined for a later sheet down one
		// index, and MoveSheet does not move it back, so each of those names
		// would now belong to the sheet before its own. The rebuilt sheet is
		// back at idx with no names of its own, so every sheet-scoped name at
		// idx or above belongs one sheet further on.
		if wb := f.WorkBook; wb != nil && wb.DefinedNames != nil {
			for i := range wb.DefinedNames.DefinedName {
				if id := wb.DefinedNames.DefinedName[i].LocalSheetID; id != nil && *id >= idx {
					shifted := *id + 1
					wb.DefinedNames.DefinedName[i].LocalSheetID = &shifted
				}
			}
		}
	}
	if wb := f.WorkBook; state != "" && wb != nil && idx < len(wb.Sheets.Sheet) {
		wb.Sheets.Sheet[idx].State = state
	}
	activeIdx, err := f.GetSheetIndex(active)
	if err != nil {
		return err
	}
	f.SetActiveSheet(activeIdx)
	return nil
}

// safeSheetCSVPath joins a CSV file name made from a sheet name onto outputDir.
// A workbook's sheet names come from workbook.xml and are attacker-controlled,
// so such a name is refused when it holds a path separator or when the joined
// path would not be a file directly inside outputDir. Any other name is an
// ordinary file name: a sheet named "." or ".." becomes "..csv" or "...csv". A
// name the caller passes in csvNames does not come through here.
func safeSheetCSVPath(outputDir, sheet, fileName string) (string, error) {
	path := filepath.Join(outputDir, fileName)
	if fileName == "" || strings.ContainsAny(fileName, `/\`) || filepath.Dir(path) != filepath.Clean(outputDir) {
		return "", fmt.Errorf("sheet name %q cannot be used as a file name: %q", sheet, fileName)
	}
	return path, nil
}

// saveSheetAsCsv saves a specific sheet in an Excel file as a CSV file. The
// rows are read before the output is touched, so a sheet that cannot be read
// never truncates an existing file.
func saveSheetAsCsv(f *excelize.File, sheetName string, outputCsvName string) error {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("failed to read rows from sheet %s: %w", sheetName, err)
	}

	file, err := os.Create(outputCsvName)
	if err != nil {
		return fmt.Errorf("failed to create CSV file %s: %w", outputCsvName, err)
	}
	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)

	for rowIdx, row := range rows {
		// Check if the row is visible (not filtered out)
		visible, err := f.GetRowVisible(sheetName, rowIdx+1) // rowIdx is 0-based, GetRowVisible is 1-based
		if err != nil {
			return fmt.Errorf("failed to check visibility of row %d in sheet %s: %w", rowIdx+1, sheetName, err)
		}
		if visible {
			err := writer.Write(row)
			if err != nil {
				return fmt.Errorf("failed to write row to CSV file %s: %w", outputCsvName, err)
			}
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to write CSV file %s: %w", outputCsvName, err)
	}
	return nil
}

// 私有函數：將 CSV 數據加入 Excel 的指定工作表，並處理非 UTF-8 編碼
func addCsvSheet(f *excelize.File, sheetName, csvFile string, encoding string) error {
	records, err := readCsvRecords(csvFile, encoding)
	if err != nil {
		return err
	}
	return writeCsvRecords(f, sheetName, records)
}

// readCsvRecords reads the CSV file at csvFile in full and returns its records,
// detecting the encoding when encoding is Auto and trimming a UTF-8 BOM off the
// first field. Every step that can fail happens here, so a caller that has not
// yet touched its workbook can decide what an unreadable CSV costs.
func readCsvRecords(csvFile string, encoding string) ([][]string, error) {
	file, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file %s: %w", csvFile, err)
	}
	defer func() { _ = file.Close() }()

	var records [][]string

	// Auto-detect encoding if specified
	if encoding == Auto {
		detectedEncoding, err := insyra.DetectEncoding(csvFile)
		if err != nil {
			// Propagate the detection error instead of silently falling back
			return nil, fmt.Errorf("failed to auto-detect encoding for %s: %w", csvFile, err)
		}
		encoding = strings.ToLower(detectedEncoding)
		insyra.LogInfo("csvxl", "addCsvSheet", "Auto-detected encoding %s for file %s", encoding, csvFile)
	}

	// Ensure we start reading from the beginning of the file
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file %s: %w", csvFile, err)
	}

	csvReader := csv.NewReader(insyracsv.DecodingReader(file, encoding))
	records, err = csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file %s: %w", csvFile, err)
	}

	// Trim UTF-8 BOM if present
	if len(records) > 0 && len(records[0]) > 0 {
		records[0][0] = strings.TrimPrefix(records[0][0], "\uFEFF")
	}

	return records, nil
}

// writeCsvRecords writes records into sheetName, the first field of the first
// record in A1, each further field one column to the right and each further
// record one row down.
func writeCsvRecords(f *excelize.File, sheetName string, records [][]string) error {
	for rowIdx, record := range records {
		for colIdx, cell := range record {
			cellAddr, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			err := f.SetCellValue(sheetName, cellAddr, cell)
			if err != nil {
				return fmt.Errorf("failed to set cell value %s: %w", cellAddr, err)
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
