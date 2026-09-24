package csvxl

import (
	"encoding/csv"
	"errors"
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
//
// A CSV that cannot be read, or whose sheet cannot be created, gets no sheet,
// and the other files are still converted. The returned error lists every
// file that failed. When every file fails, no workbook is written.
func CsvToExcel(csvFiles []string, sheetNames []string, output string, csvEncoding ...string) error {
	encoding := Auto // Default to auto-detection
	if len(csvEncoding) == 1 {
		encoding = csvEncoding[0]
	} else if len(csvEncoding) > 1 {
		return fmt.Errorf("too many arguments for csvEncoding")
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	var failures []error
	converted := 0

	for idx, given := range csvFiles {
		csvFile, fileErr := resolveCsvSource(given)

		// 如果提供了自訂工作表名稱，則使用它，否則使用 CSV 檔案的名稱
		sheetName := getSheetName(csvFile, sheetNames, idx)

		var records [][]string
		if fileErr == nil {
			records, fileErr = readCsvRecords(csvFile, encoding)
		}
		if fileErr == nil {
			// 第一個成功的檔案沿用新工作簿的預設工作表，而不是另建一張
			if converted == 0 {
				fileErr = f.SetSheetName(f.GetSheetName(0), sheetName)
			} else {
				_, fileErr = f.NewSheet(sheetName)
			}
			if fileErr != nil {
				fileErr = fmt.Errorf("failed to create sheet %s for %s: %w", sheetName, csvFile, fileErr)
			}
		}
		if fileErr == nil {
			fileErr = writeRecords(f, sheetName, csvFile, records)
		}
		if fileErr != nil {
			failures = append(failures, fileErr)
			continue
		}
		converted++
	}

	if converted == 0 && len(failures) > 0 {
		return batchError("convert", failures, len(csvFiles))
	}

	if err := f.SaveAs(output); err != nil {
		return fmt.Errorf("failed to save Excel file %s: %w", output, err)
	}

	insyra.LogInfo("csvxl", "CsvToExcel", "Converted %d of %d CSV files to Excel file %s.", converted, len(csvFiles), output)
	if len(failures) > 0 {
		return batchError("convert", failures, len(csvFiles))
	}
	return nil
}

// Append CSV files to an existing Excel file, supporting custom sheet names.
// If the sheet name is not specified, the file name of the CSV file will be used.
// If the sheet is exists, it will be overwritten.
// If csvEncoding is not specified, auto-detection will be used.
//
// A CSV is read in full before its sheet is replaced, so a CSV that cannot be
// read leaves the sheet of the same name as it was, and the other files are
// still appended. The returned error lists every file that failed. When every
// file fails, the workbook is not rewritten.
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string) error {
	encoding := Auto // Default to auto-detection
	if len(csvEncoding) == 1 {
		encoding = csvEncoding[0]
	} else if len(csvEncoding) > 1 {
		return fmt.Errorf("too many arguments for csvEncoding")
	}

	f, err := excelize.OpenFile(existingFile, insyra.ExcelReadOptions())
	if err != nil {
		return fmt.Errorf("failed to open Excel file %s: %w", existingFile, err)
	}
	defer func() { _ = f.Close() }()

	var failures []error
	appended := 0

	for idx, given := range csvFiles {
		csvFile, fileErr := resolveCsvSource(given)

		// 如果提供了自訂工作表名稱，則使用它，否則使用 CSV 檔案的名稱
		sheetName := getSheetName(csvFile, sheetNames, idx)

		var records [][]string
		if fileErr == nil {
			records, fileErr = readCsvRecords(csvFile, encoding)
		}
		if fileErr == nil {
			if fileErr = replaceSheet(f, sheetName); fileErr != nil {
				fileErr = fmt.Errorf("failed to create sheet %s for %s: %w", sheetName, csvFile, fileErr)
			}
		}
		if fileErr == nil {
			fileErr = writeRecords(f, sheetName, csvFile, records)
		}
		if fileErr != nil {
			failures = append(failures, fileErr)
			continue
		}
		appended++
	}

	if appended == 0 && len(failures) > 0 {
		return batchError("append", failures, len(csvFiles))
	}

	if err := f.SaveAs(existingFile); err != nil {
		return fmt.Errorf("failed to save Excel file %s: %w", existingFile, err)
	}

	insyra.LogInfo("csvxl", "AppendCsvToExcel", "Appended %d of %d CSV files to Excel file %s.", appended, len(csvFiles), existingFile)
	if len(failures) > 0 {
		return batchError("append", failures, len(csvFiles))
	}
	return nil
}

// ExcelToCsv splits an Excel file into multiple CSV files, one per sheet.
// If customNames is provided, it uses them as CSV filenames; otherwise, it uses the sheet names.
func ExcelToCsv(excelFile string, outputDir string, csvNames []string, onlyContainSheets ...string) error {
	f, err := excelize.OpenFile(excelFile, insyra.ExcelReadOptions())
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
		// A name that is not in the file used to be dropped without a word, so
		// a typo produced a smaller conversion that looked like it had worked.
		var missing []string
		for _, s := range onlyContainSheets {
			if sliceutil.Contains(sheetsInXlsx, s) {
				sheetsToProcess = append(sheetsToProcess, s)
			} else {
				missing = append(missing, s)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("sheet(s) %s are not in %s (it has %s)",
				strings.Join(missing, ", "), excelFile, strings.Join(sheetsInXlsx, ", "))
		}
	} else {
		sheetsToProcess = sheetsInXlsx
	}

	numSheets := len(sheetsToProcess)
	for idx, sheet := range sheetsToProcess {
		if err := safeSheetFileName(sheet); err != nil {
			return err
		}
		csvName := sheet + ".csv"
		if len(csvNames) > idx && csvNames[idx] != "" {
			csvName = csvOutputName(csvNames[idx])
		}

		outputCsv := filepath.Join(outputDir, csvName)
		err := saveSheetAsCsv(f, sheet, outputCsv)
		if err != nil {
			return fmt.Errorf("failed to save sheet %s as CSV: %w", sheet, err)
		}
	}

	insyra.LogInfo("csvxl", "ExcelToCsv", "Successfully converted %d sheets to CSV files in %s.", numSheets, outputDir)
	return nil
}

// ===============================

// replaceSheet makes sheetName an empty sheet in f. An existing sheet of that
// name is deleted first so nothing from it survives; excelize.NewSheet alone
// would return the existing sheet and leave its cells in place. excelize
// refuses to delete a workbook's only sheet, so in that case a placeholder is
// created first and removed once the new sheet exists.
func replaceSheet(f *excelize.File, sheetName string) error {
	idx, err := f.GetSheetIndex(sheetName)
	if err != nil {
		return err
	}
	if idx != -1 {
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
			return f.DeleteSheet(placeholder)
		}
		return nil
	}
	_, err = f.NewSheet(sheetName)
	return err
}

// safeSheetFileName returns the sheet name if it can be used as a single
// path element under the output directory, or an error. A workbook's
// sheet names come from workbook.xml and are attacker-controlled, so
// "../x" or "a/b" must never be joined onto outputDir.
func safeSheetFileName(sheet string) error {
	if sheet == "" || sheet == "." || sheet == ".." ||
		strings.ContainsAny(sheet, `/\`) || filepath.Base(sheet) != sheet {
		return fmt.Errorf("sheet name %q cannot be used as a file name", sheet)
	}
	return nil
}

// saveSheetAsCsv saves a specific sheet in an Excel file as a CSV file. The
// rows are read before the output is touched, and the CSV is written to a
// temporary file that is renamed into place, so a bad sheet never truncates
// an existing file.
func saveSheetAsCsv(f *excelize.File, sheetName string, outputCsvName string) error {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("failed to read rows from sheet %s: %w", sheetName, err)
	}

	file, err := os.CreateTemp(filepath.Dir(outputCsvName), "."+filepath.Base(outputCsvName)+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create CSV file %s: %w", outputCsvName, err)
	}
	tmpPath := file.Name()
	cleanup := func() { _ = file.Close(); _ = os.Remove(tmpPath) }

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
				cleanup()
				return fmt.Errorf("failed to write row to CSV file %s: %w", outputCsvName, err)
			}
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		cleanup()
		return fmt.Errorf("failed to write CSV file %s: %w", outputCsvName, err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write CSV file %s: %w", outputCsvName, err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, outputCsvName); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to replace CSV file %s: %w", outputCsvName, err)
	}
	return nil
}

// readCsvRecords reads and decodes a whole CSV file, handling non-UTF-8
// encodings, so a file that cannot be read is known before any sheet is
// created or replaced. A file with more rows or columns than a sheet can hold
// is refused here for the same reason.
func readCsvRecords(csvFile string, encoding string) ([][]string, error) {
	file, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file %s: %w", csvFile, err)
	}
	defer func() { _ = file.Close() }()

	// Auto-detect encoding if specified
	if encoding == Auto {
		detectedEncoding, err := insyra.DetectEncoding(csvFile)
		if err != nil {
			// Propagate the detection error instead of silently falling back
			return nil, fmt.Errorf("failed to auto-detect encoding for %s: %w", csvFile, err)
		}
		encoding = strings.ToLower(detectedEncoding)
		insyra.LogInfo("csvxl", "readCsvRecords", "Auto-detected encoding %s for file %s", encoding, csvFile)
	}

	// Ensure we start reading from the beginning of the file
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file %s: %w", csvFile, err)
	}

	reader, decErr := insyracsv.DecodingReader(file, encoding)
	if decErr != nil {
		return nil, fmt.Errorf("failed to read CSV file %s: %w", csvFile, decErr)
	}

	records, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file %s: %w", csvFile, err)
	}

	// Trim UTF-8 BOM if present
	if len(records) > 0 && len(records[0]) > 0 {
		records[0][0] = strings.TrimPrefix(records[0][0], "\uFEFF")
	}

	if len(records) > excelize.TotalRows {
		return nil, fmt.Errorf("CSV file %s has %d rows, more than the %d a sheet can hold", csvFile, len(records), excelize.TotalRows)
	}
	for i, record := range records {
		if len(record) > excelize.MaxColumns {
			return nil, fmt.Errorf("row %d of CSV file %s has %d columns, more than the %d a sheet can hold", i+1, csvFile, len(record), excelize.MaxColumns)
		}
	}
	return records, nil
}

// writeRecords writes rows read by readCsvRecords into a sheet.
func writeRecords(f *excelize.File, sheetName, csvFile string, records [][]string) error {
	for rowIdx, record := range records {
		for colIdx, cell := range record {
			cellAddr, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			if err == nil {
				err = f.SetCellValue(sheetName, cellAddr, cell)
			}
			if err != nil {
				return fmt.Errorf("failed to write row %d of %s to sheet %s: %w", rowIdx+1, csvFile, sheetName, err)
			}
		}
	}
	return nil
}

// batchError reports every file in a batch that failed, one per line, and
// wraps each cause so errors.Is still finds it.
func batchError(verb string, failures []error, total int) error {
	return fmt.Errorf("%d of %d CSV files failed to %s:\n%w", len(failures), total, verb, errors.Join(failures...))
}

// 私有函數：取得工作表名稱，如果提供了自訂名稱則使用，否則使用 CSV 檔案名稱
func getSheetName(csvFile string, sheetNames []string, idx int) string {
	if len(sheetNames) > idx && sheetNames[idx] != "" {
		return sheetNames[idx]
	}
	return strings.TrimSuffix(filepath.Base(csvFile), filepath.Ext(csvFile))
}

// resolveCsvSource turns a path the caller gave into the file to read. The
// path is used as written when it names a file, so a CSV called export.txt or
// DATA.CSV is read as itself. Only when nothing is there, or a directory is,
// is .csv appended as a convenience for a name whose extension was left off.
func resolveCsvSource(given string) (string, error) {
	if isRegularFile(given) {
		return given, nil
	}
	if strings.EqualFold(filepath.Ext(given), ".csv") {
		return given, fmt.Errorf("no CSV file at \"%s\": %w", given, os.ErrNotExist)
	}
	withCsv := given + ".csv"
	if isRegularFile(withCsv) {
		return withCsv, nil
	}
	return given, fmt.Errorf("no CSV file at \"%s\" or \"%s\": %w", given, withCsv, os.ErrNotExist)
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// csvOutputName keeps a name the caller gave as written when it already has an
// extension, of any case, and adds .csv only when it has none.
func csvOutputName(name string) string {
	if filepath.Ext(name) != "" {
		return name
	}
	return name + ".csv"
}
