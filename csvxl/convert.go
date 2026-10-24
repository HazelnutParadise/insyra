package csvxl

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/HazelnutParadise/Go-Utils/sliceutil"
	"github.com/HazelnutParadise/insyra"
	insyracsv "github.com/HazelnutParadise/insyra/internal/csv"
	"github.com/HazelnutParadise/insyra/internal/excelsheet"

	"github.com/xuri/excelize/v2"
)

// CsvEncoding Options
const (
	UTF8 = "utf-8"
	Big5 = "big5"
	Auto = "auto"
)

// isAutoEncoding reports whether an encoding argument asks for detection. It
// reads "" and "auto" in any case the way the core CSV readers do, so the same
// argument means the same thing in both packages.
func isAutoEncoding(encoding string) bool {
	return encoding == "" || strings.EqualFold(encoding, Auto)
}

// CsvToExcel converts multiple CSV files to an Excel file, supporting custom sheet names.
// If the sheet name is not specified, the file name of the CSV file will be used.
// If csvEncoding is not specified, auto-detection will be used.
//
// Deprecated: use CSVToExcel, which takes each CSV together with its sheet
// name as a SheetFromCSV. Removed in 0.4.
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

	for idx, given := range csvFiles {
		// A file that is not there fails when it is read, and is counted then.
		csvFile, _ := resolveCsvSource(given)

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

	return nil
}

// SheetFromCSV is one sheet of the workbook CSVToExcel or AppendCSVToExcel
// writes: the CSV file at Path, written as the sheet named Sheet.
type SheetFromCSV struct {
	// Path is the CSV file, opened as written; ".csv" is appended only when no
	// file has that path.
	Path string
	// Sheet names the sheet. Empty means the CSV file's name without its
	// extension.
	Sheet string
}

// CSVToExcel converts CSV files into one Excel workbook saved at output, one
// sheet per element of sheets, in the order given. csvEncoding is the files'
// encoding, detected when it is left out.
//
// Two elements that name the same sheet, compared without regard to case as
// Excel compares sheet names, are an error before any file is read, and no
// workbook is written. Otherwise it behaves as CsvToExcel: a CSV that cannot
// be read is counted as failed and the others are still converted.
func CSVToExcel(sheets []SheetFromCSV, output string, csvEncoding ...string) error {
	csvFiles, sheetNames, err := planSheets(sheets)
	if err != nil {
		return err
	}
	return CsvToExcel(csvFiles, sheetNames, output, csvEncoding...)
}

// AppendCsvToExcel appends CSV files to an existing Excel file, supporting custom sheet names.
// If the sheet name is not specified, the file name of the CSV file will be used.
// If the sheet is exists, it will be overwritten.
// If csvEncoding is not specified, auto-detection will be used.
//
// Deprecated: use AppendCSVToExcel, which takes each CSV together with its
// sheet name as a SheetFromCSV. Removed in 0.4.
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

	for idx, given := range csvFiles {
		// A file that is not there fails when it is read, and is counted then.
		csvFile, _ := resolveCsvSource(given)

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

		idx, err := f.GetSheetIndex(sheetName)
		if err != nil {
			return fmt.Errorf("failed to create new sheet %s: %w", sheetName, err)
		}
		if idx == -1 {
			if _, err := f.NewSheet(sheetName); err != nil {
				return fmt.Errorf("failed to create new sheet %s: %w", sheetName, err)
			}
		} else if err := excelsheet.Replace(f, sheetName, idx); err != nil {
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

	return nil
}

// AppendCSVToExcel writes CSV files into the existing Excel workbook at
// existingFile, one sheet per element of sheets, and saves it in place. A
// sheet that already has that name is replaced. csvEncoding is the files'
// encoding, detected when it is left out.
//
// Two elements that name the same sheet, compared without regard to case, are
// an error before any file is read, and the workbook is not rewritten.
// Otherwise it behaves as AppendCsvToExcel.
func AppendCSVToExcel(sheets []SheetFromCSV, existingFile string, csvEncoding ...string) error {
	csvFiles, sheetNames, err := planSheets(sheets)
	if err != nil {
		return err
	}
	return AppendCsvToExcel(csvFiles, sheetNames, existingFile, csvEncoding...)
}

// planSheets splits sheets into the deprecated functions' parallel slices and
// refuses two elements that name the same sheet, compared without regard to
// case as Excel compares sheet names. The sheet name of an element without
// one is worked out the way CsvToExcel works it out.
func planSheets(sheets []SheetFromCSV) ([]string, []string, error) {
	csvFiles := make([]string, len(sheets))
	sheetNames := make([]string, len(sheets))
	resolved := make([]string, len(sheets))
	seen := make(map[string]int, len(sheets))
	for i, s := range sheets {
		csvFiles[i] = s.Path
		sheetNames[i] = s.Sheet
		csvFile := s.Path
		if !strings.HasSuffix(csvFile, ".csv") {
			csvFile += ".csv"
		}
		name := getSheetName(csvFile, sheetNames, i)
		resolved[i] = name
		if j, ok := seen[strings.ToLower(name)]; ok {
			prev := resolved[j]
			if prev == name {
				return nil, nil, fmt.Errorf("%s and %s would both be written to sheet %q", sheets[j].Path, s.Path, name)
			}
			return nil, nil, fmt.Errorf("%s and %s would both be written to sheet %q, which Excel treats as the same sheet as %q", sheets[j].Path, s.Path, name, prev)
		}
		seen[strings.ToLower(name)] = i
	}
	return csvFiles, sheetNames, nil
}

// ExcelToCSVOptions configures ExcelToCSV and ExcelDirToCSV. The zero value
// converts every sheet and guards formula-like text.
type ExcelToCSVOptions struct {
	// Sheets limits the conversion to these sheets. Empty means every sheet.
	// A name the workbook does not have is an error.
	Sheets []string
	// FileNames maps a sheet name to the name of the CSV file written for it.
	// A sheet with no entry, or an empty one, is written as its name plus
	// ".csv". A name with an extension of any case is used as written; one
	// with none gets ".csv". A key that is not among the sheets converted is
	// an error. ExcelDirToCSV, which names files after their workbook, refuses
	// FileNames.
	FileNames map[string]string
	// AllowFormulas writes formula-like text without a leading apostrophe.
	AllowFormulas bool
}

func oneExcelToCSVOptions(options []ExcelToCSVOptions) (ExcelToCSVOptions, error) {
	if len(options) > 1 {
		return ExcelToCSVOptions{}, fmt.Errorf("at most one ExcelToCSVOptions may be given, got %d", len(options))
	}
	if len(options) == 1 {
		return options[0], nil
	}
	return ExcelToCSVOptions{}, nil
}

// ExcelToCSV writes each sheet of the workbook at excelFile as a CSV file in
// outputDir. See ExcelToCSVOptions for choosing sheets, naming their files and
// the formula guard, which prefixes formula-like text with an apostrophe
// unless AllowFormulas is set.
//
// Every file name is checked before the first CSV is written: a name that
// would leave outputDir, and two sheets that would be written to one file,
// compared without regard to case, are an error and nothing is written.
func ExcelToCSV(excelFile, outputDir string, options ...ExcelToCSVOptions) error {
	opts, err := oneExcelToCSVOptions(options)
	if err != nil {
		return err
	}
	f, err := excelize.OpenFile(excelFile)
	if err != nil {
		return fmt.Errorf("failed to open Excel file %s: %w", excelFile, err)
	}
	defer func() { _ = f.Close() }()
	sheets, err := selectSheets(excelFile, f.GetSheetList(), opts.Sheets)
	if err != nil {
		return err
	}
	if err := checkFileNameKeys(excelFile, opts.FileNames, sheets); err != nil {
		return err
	}
	targets := make([]csvTarget, 0, len(sheets))
	for _, sheet := range sheets {
		name := sheet + ".csv"
		if given := opts.FileNames[sheet]; given != "" {
			name = csvOutputName(given)
		}
		path, err := safeSheetCSVPath(outputDir, sheet, name)
		if err != nil {
			return err
		}
		targets = append(targets, csvTarget{source: "sheet " + sheet, sheet: sheet, path: path})
	}
	if err := refuseSharedCSVFiles(targets); err != nil {
		return err
	}
	if err := writeCSVTargets(f, outputDir, targets, opts.AllowFormulas); err != nil {
		return err
	}
	return nil
}

// SheetNames returns the names of the sheets of the workbook at excelFile, in
// workbook order.
func SheetNames(excelFile string) ([]string, error) {
	f, err := excelize.OpenFile(excelFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file %s: %w", excelFile, err)
	}
	defer func() { _ = f.Close() }()
	return f.GetSheetList(), nil
}

// selectSheets returns the sheets to convert: every sheet when wanted is
// empty, otherwise the wanted ones, each once. A wanted name the workbook does
// not have is an error.
func selectSheets(excelFile string, inFile, wanted []string) ([]string, error) {
	if len(wanted) == 0 {
		return inFile, nil
	}
	selected := make([]string, 0, len(wanted))
	for _, s := range wanted {
		if !sliceutil.Contains(inFile, s) {
			return nil, fmt.Errorf("sheet %q not found in %s", s, excelFile)
		}
		if !sliceutil.Contains(selected, s) {
			selected = append(selected, s)
		}
	}
	return selected, nil
}

// csvTarget is one sheet and the CSV file it is written to. source names the
// sheet in an error, with its workbook when there are several.
type csvTarget struct {
	source string
	sheet  string
	path   string
}

// checkFileNameKeys refuses a FileNames key that is not among the sheets
// being converted, so a typo is not silently ignored.
func checkFileNameKeys(excelFile string, fileNames map[string]string, sheets []string) error {
	var unknown []string
	for key := range fileNames {
		if !sliceutil.Contains(sheets, key) {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("FileNames names sheet(s) %s, which are not among the sheets converted from %s (%s)",
		strings.Join(unknown, ", "), excelFile, strings.Join(sheets, ", "))
}

// refuseSharedCSVFiles refuses two sheets that would be written to one file.
// File names are compared without regard to case, because the file systems of
// macOS and Windows do not tell them apart.
func refuseSharedCSVFiles(targets []csvTarget) error {
	seen := make(map[string]csvTarget, len(targets))
	for _, t := range targets {
		key := strings.ToLower(t.path)
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("%s and %s would both be written to %s", prev.source, t.source, filepath.Base(prev.path))
		}
		seen[key] = t
	}
	return nil
}

// writeCSVTargets creates outputDir when it is missing and writes each sheet
// of f to its CSV file.
func writeCSVTargets(f *excelize.File, outputDir string, targets []csvTarget, allowFormulas bool) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}
	for _, t := range targets {
		if err := saveSheetAsCsv(f, t.sheet, t.path, allowFormulas); err != nil {
			return fmt.Errorf("failed to save sheet %s as CSV: %w", t.sheet, err)
		}
	}
	return nil
}

func csvOutputName(name string) string {
	if filepath.Ext(name) != "" {
		return name
	}
	return name + ".csv"
}

// ExcelToCsv splits an Excel file into multiple CSV files, one per sheet.
// If customNames is provided, it uses them as CSV filenames; otherwise, it uses the sheet names.
//
// Deprecated: use ExcelToCSV. Sheet selection moves to ExcelToCSVOptions.Sheets
// and file names to ExcelToCSVOptions.FileNames, keyed by sheet name;
// ExcelToCSV also guards formula-like text by default. Removed in 0.4.
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

	return nil
}

// plainNumber matches a number written in decimal digits, with an optional
// sign, fraction and exponent, the way a spreadsheet reads one. It decides by
// spelling, not by what strconv.ParseFloat accepts: "+Inf" and "0x1p3" are
// not numbers to a spreadsheet, and "-1e309" is one even though it overflows
// a float64.
var plainNumber = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$`)

func guardFormula(s string) string {
	trimmed := strings.TrimLeft(s, " \t\r\n")
	if trimmed == "" {
		return s
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		if plainNumber.MatchString(strings.TrimSpace(trimmed)) {
			return s
		}
		return "'" + s
	}
	return s
}

// ===============================

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
func saveSheetAsCsv(f *excelize.File, sheetName string, outputCsvName string, allowFormulas ...bool) error {
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
			return fmt.Errorf("failed to check visibility of row index %d in sheet %s: %w", rowIdx, sheetName, err)
		}
		if visible {
			if len(allowFormulas) > 0 && !allowFormulas[0] {
				for i, cell := range row {
					row[i] = guardFormula(cell)
				}
			}
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
	if isAutoEncoding(encoding) {
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

// resolveCsvSource turns a path the caller gave into the file to read. The
// path is used as written when it names a file, so a CSV called export.txt or
// DATA.CSV is read as itself. Only when nothing is there, or a directory is,
// is .csv appended as a convenience for a name whose extension was left off.
// When neither is a file it returns the path as given and an error naming
// what was tried.
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
