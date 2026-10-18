package csvxl

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HazelnutParadise/Go-Utils/sliceutil"
	"github.com/HazelnutParadise/insyra"
	insyracsv "github.com/HazelnutParadise/insyra/internal/csv"
	"github.com/HazelnutParadise/insyra/internal/excelsheet"

	"github.com/xuri/excelize/v2"
)

// UTF8, Big5 and Auto are common values for the encoding argument of
// CSVToExcel, AppendCSVToExcel, CSVDirToExcel and ReadCSVToString. Any name
// the core CSV readers accept works as well, and a name no decoder handles is
// an error.
const (
	// UTF8 reads a file as UTF-8.
	UTF8 = "utf-8"
	// Big5 reads a file as Big5 (Traditional Chinese).
	Big5 = "big5"
	// Auto detects the encoding of each file. It is the default, and "" means
	// the same.
	Auto = "auto"
)

// isAutoEncoding reports whether an encoding argument asks for detection. It
// reads "" and "auto" in any case the way the core CSV readers do, so the same
// argument means the same thing in both packages.
func isAutoEncoding(encoding string) bool {
	return encoding == "" || strings.EqualFold(encoding, Auto)
}

// checkEncoding refuses an encoding no decoder handles before any file is
// read, so the error names it once and an empty file list does not hide it.
// Detection ("" and "auto" in any case) is always accepted.
func checkEncoding(encoding string) error {
	if isAutoEncoding(encoding) {
		return nil
	}
	// DecodingReader refuses a name it has no decoder for before it reads
	// anything, so an empty reader is enough to ask it.
	_, err := insyracsv.DecodingReader(strings.NewReader(""), encoding)
	return err
}

// SheetFromCSV is one sheet of the workbook CSVToExcel or AppendCSVToExcel
// writes: the CSV file at Path, written as the sheet named Sheet.
type SheetFromCSV struct {
	// Path is the CSV file. When nothing is there, Path plus ".csv" is tried.
	Path string
	// Sheet names the sheet. Empty means the CSV file's name without its
	// extension.
	Sheet string
}

// CSVToExcel converts CSV files into one Excel workbook and saves it at
// output, one sheet per element of sheets, in the order given. csvEncoding is
// the files' encoding, detected when it is left out.
//
// Two elements that name the same sheet, compared without regard to case as
// Excel compares sheet names, are an error before any file is read, and no
// workbook is written. A CSV that cannot be read, or whose sheet cannot be
// created, gets no sheet, and the other files are still converted. The
// returned error lists every file that failed. When every file fails, no
// workbook is written.
func CSVToExcel(sheets []SheetFromCSV, output string, csvEncoding ...string) error {
	encoding := Auto // Default to auto-detection
	if len(csvEncoding) == 1 {
		encoding = csvEncoding[0]
	} else if len(csvEncoding) > 1 {
		return fmt.Errorf("too many arguments for csvEncoding")
	}
	if err := checkEncoding(encoding); err != nil {
		return err
	}
	plans, err := planSheets(sheets)
	if err != nil {
		return err
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	var failures []error
	converted := 0

	for _, p := range plans {
		csvFile, fileErr, sheetName := p.csvFile, p.resolveErr, p.sheet

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
		return batchError("convert", failures, len(plans))
	}

	if err := f.SaveAs(output); err != nil {
		return fmt.Errorf("failed to save Excel file %s: %w", output, err)
	}

	if len(failures) > 0 {
		return batchError("convert", failures, len(plans))
	}
	return nil
}

// CsvToExcel converts CSV files into one Excel workbook. sheetNames[i] names
// the sheet of csvFiles[i]; a missing or empty name uses the CSV file's name.
//
// Deprecated: use CSVToExcel, which takes each CSV together with its sheet
// name. Removed in the release after the one that deprecated it.
func CsvToExcel(csvFiles []string, sheetNames []string, output string, csvEncoding ...string) error {
	return CSVToExcel(pairByPosition(csvFiles, sheetNames), output, csvEncoding...)
}

// AppendCSVToExcel writes CSV files into the existing Excel workbook at
// existingFile, one sheet per element of sheets, and saves it in place. A
// sheet that already has that name is replaced by a fresh one in the same
// position. csvEncoding is the files' encoding, detected when it is left out.
//
// Two elements that name the same sheet, compared without regard to case, are
// an error before any file is read, and the workbook is not rewritten. A CSV
// is read in full before its sheet is replaced, so a CSV that cannot be read
// leaves the sheet of the same name as it was, and the other files are still
// appended. The returned error lists every file that failed. When every file
// fails, the workbook is not rewritten.
func AppendCSVToExcel(sheets []SheetFromCSV, existingFile string, csvEncoding ...string) error {
	encoding := Auto // Default to auto-detection
	if len(csvEncoding) == 1 {
		encoding = csvEncoding[0]
	} else if len(csvEncoding) > 1 {
		return fmt.Errorf("too many arguments for csvEncoding")
	}
	if err := checkEncoding(encoding); err != nil {
		return err
	}
	plans, err := planSheets(sheets)
	if err != nil {
		return err
	}

	f, err := excelize.OpenFile(existingFile, insyra.ExcelReadOptions())
	if err != nil {
		return fmt.Errorf("failed to open Excel file %s: %w", existingFile, err)
	}
	defer func() { _ = f.Close() }()

	var failures []error
	appended := 0

	for _, p := range plans {
		csvFile, fileErr, sheetName := p.csvFile, p.resolveErr, p.sheet

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
		return batchError("append", failures, len(plans))
	}

	if err := f.SaveAs(existingFile); err != nil {
		return fmt.Errorf("failed to save Excel file %s: %w", existingFile, err)
	}

	if len(failures) > 0 {
		return batchError("append", failures, len(plans))
	}
	return nil
}

// AppendCsvToExcel writes CSV files into an existing Excel workbook.
// sheetNames[i] names the sheet of csvFiles[i]; a missing or empty name uses
// the CSV file's name.
//
// Deprecated: use AppendCSVToExcel, which takes each CSV together with its
// sheet name. Removed in the release after the one that deprecated it.
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string) error {
	return AppendCSVToExcel(pairByPosition(csvFiles, sheetNames), existingFile, csvEncoding...)
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
	// AllowFormulas writes text a spreadsheet would run as a formula exactly
	// as it is. By default such text (starting with =, +, - or @, and not
	// only a number) gets a leading single quote so a spreadsheet opening the
	// CSV shows it instead of running it: text that was safe inside the
	// workbook becomes a formula again once it is a CSV.
	AllowFormulas bool
}

// ExcelToCsvOptions is the old name of ExcelToCSVOptions.
//
// Deprecated: use ExcelToCSVOptions, which is the same type. Removed in the
// release after the one that deprecated it.
type ExcelToCsvOptions = ExcelToCSVOptions

func oneExcelToCSVOptions(opts []ExcelToCSVOptions) (ExcelToCSVOptions, error) {
	if len(opts) > 1 {
		return ExcelToCSVOptions{}, fmt.Errorf("at most one ExcelToCSVOptions may be given, got %d", len(opts))
	}
	if len(opts) == 1 {
		return opts[0], nil
	}
	return ExcelToCSVOptions{}, nil
}

// ExcelToCSV writes each sheet of the workbook at excelFile as a CSV file in
// outputDir. See ExcelToCSVOptions for choosing sheets, naming their files and
// the formula guard.
//
// Every file name is checked before the first CSV is written: a name that
// would leave outputDir, and two sheets that would be written to one file,
// compared without regard to case, are an error and nothing is written.
func ExcelToCSV(excelFile string, outputDir string, options ...ExcelToCSVOptions) error {
	opts, err := oneExcelToCSVOptions(options)
	if err != nil {
		return err
	}
	return excelToCSV(excelFile, outputDir, opts, nil)
}

// ExcelToCsv writes each sheet of a workbook as a CSV file. csvNames[i] names
// the file of the i-th converted sheet; a missing or empty name uses the
// sheet's name plus ".csv".
//
// Deprecated: use ExcelToCSV, which names a sheet's file by the sheet's name
// through ExcelToCSVOptions.FileNames. Removed in the release after the one
// that deprecated it.
func ExcelToCsv(excelFile string, outputDir string, csvNames []string, options ...ExcelToCsvOptions) error {
	opts, err := oneExcelToCSVOptions(options)
	if err != nil {
		return err
	}
	if len(csvNames) > 0 && len(opts.FileNames) > 0 {
		return fmt.Errorf("file names were given both in csvNames and in FileNames; give them in FileNames")
	}
	return excelToCSV(excelFile, outputDir, opts, csvNames)
}

// SheetNames returns the names of the sheets of the workbook at excelFile, in
// workbook order.
func SheetNames(excelFile string) ([]string, error) {
	f, err := excelize.OpenFile(excelFile, insyra.ExcelReadOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file %s: %w", excelFile, err)
	}
	defer func() { _ = f.Close() }()
	return f.GetSheetList(), nil
}

// excelToCSV is ExcelToCSV. csvNames, when not nil, are ExcelToCsv's file
// names, the i-th naming the i-th converted sheet.
func excelToCSV(excelFile, outputDir string, opts ExcelToCSVOptions, csvNames []string) error {
	f, err := excelize.OpenFile(excelFile, insyra.ExcelReadOptions())
	if err != nil {
		return fmt.Errorf("failed to open Excel file %s: %w", excelFile, err)
	}
	defer func() { _ = f.Close() }()

	sheets, err := selectSheets(excelFile, f.GetSheetList(), opts.Sheets)
	if err != nil {
		return err
	}
	fileNames := opts.FileNames
	if csvNames != nil {
		fileNames = map[string]string{}
		for idx, sheet := range sheets {
			if idx < len(csvNames) && csvNames[idx] != "" {
				fileNames[sheet] = csvNames[idx]
			}
		}
	}
	if err := checkFileNameKeys(excelFile, fileNames, sheets); err != nil {
		return err
	}

	targets := make([]csvTarget, 0, len(sheets))
	for _, sheet := range sheets {
		csvName := sheet + ".csv"
		if name := fileNames[sheet]; name != "" {
			csvName = csvOutputName(name)
		}
		path, err := safeSheetCSVPath(outputDir, sheet, csvName)
		if err != nil {
			return err
		}
		targets = append(targets, csvTarget{source: "sheet " + sheet, sheet: sheet, path: path})
	}
	if err := refuseSharedCSVFiles(targets); err != nil {
		return err
	}
	return writeCSVTargets(f, outputDir, targets, opts.AllowFormulas)
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
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
		}
	}
	for _, t := range targets {
		if err := saveSheetAsCsv(f, t.sheet, t.path, allowFormulas); err != nil {
			return fmt.Errorf("failed to save sheet %s as CSV: %w", t.sheet, err)
		}
	}
	return nil
}

// ===============================

// replaceSheet makes sheetName an empty sheet in f: a new sheet, or the
// existing one rebuilt in its place by excelsheet.Replace, so nothing of the
// old sheet survives while it keeps its position among the sheets.
func replaceSheet(f *excelize.File, sheetName string) error {
	idx, err := f.GetSheetIndex(sheetName)
	if err != nil {
		return err
	}
	if idx == -1 {
		_, err = f.NewSheet(sheetName)
		return err
	}
	return excelsheet.Replace(f, sheetName, idx)
}

// safeSheetCSVPath joins the CSV file name used for a sheet onto outputDir. A
// workbook's sheet names come from workbook.xml and are attacker-controlled, so
// the name actually used, the sheet name plus ".csv" or the caller's FileNames
// entry, is refused when it holds a path separator or when the joined path
// would not be a file directly inside outputDir. Any other name is an ordinary
// file name: a sheet named "." or ".." becomes "..csv" or "...csv".
func safeSheetCSVPath(outputDir, sheet, fileName string) (string, error) {
	path := filepath.Join(outputDir, fileName)
	if fileName == "" || strings.ContainsAny(fileName, `/\`) || filepath.Dir(path) != filepath.Clean(outputDir) {
		return "", fmt.Errorf("sheet name %q cannot be used as a file name: %q", sheet, fileName)
	}
	return path, nil
}

// saveSheetAsCsv saves a specific sheet in an Excel file as a CSV file. The
// rows are read before the output is touched, and the CSV is written to a
// temporary file that is renamed into place, so a bad sheet never truncates
// an existing file.
func saveSheetAsCsv(f *excelize.File, sheetName string, outputCsvName string, allowFormulas bool) error {
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
			if !allowFormulas {
				guarded := make([]string, len(row))
				for i, cell := range row {
					guarded[i] = insyracsv.GuardFormula(cell)
				}
				row = guarded
			}
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
	if isAutoEncoding(encoding) {
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

// plannedSheet is one element of CSVToExcel's or AppendCSVToExcel's sheets:
// the CSV file to read, or why it cannot be found, and the sheet it becomes.
type plannedSheet struct {
	csvFile    string
	resolveErr error
	sheet      string
}

// planSheets finds each CSV and names its sheet, and refuses two elements
// that name the same sheet, compared without regard to case as Excel compares
// sheet names, before any file is read.
func planSheets(sheets []SheetFromCSV) ([]plannedSheet, error) {
	plans := make([]plannedSheet, len(sheets))
	seen := make(map[string]int, len(sheets))
	for i, s := range sheets {
		csvFile, err := resolveCsvSource(s.Path)
		name := s.Sheet
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(csvFile), filepath.Ext(csvFile))
		}
		if j, ok := seen[strings.ToLower(name)]; ok {
			if plans[j].sheet == name {
				return nil, fmt.Errorf("%s and %s would both be written to sheet %q", sheets[j].Path, s.Path, name)
			}
			return nil, fmt.Errorf("%s and %s would both be written to sheet %q, which Excel treats as the same sheet as %q", sheets[j].Path, s.Path, name, plans[j].sheet)
		}
		seen[strings.ToLower(name)] = i
		plans[i] = plannedSheet{csvFile: csvFile, resolveErr: err, sheet: name}
	}
	return plans, nil
}

// pairByPosition builds the deprecated functions' sheets: sheetNames[i]
// names the sheet of csvFiles[i], and a name past the last file is ignored.
func pairByPosition(csvFiles, sheetNames []string) []SheetFromCSV {
	sheets := make([]SheetFromCSV, len(csvFiles))
	for i, path := range csvFiles {
		sheets[i].Path = path
		if i < len(sheetNames) {
			sheets[i].Sheet = sheetNames[i]
		}
	}
	return sheets
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

// selectSheets returns the sheets to convert: every sheet when wanted is
// empty, otherwise the wanted ones, each once. A wanted name the file does not have is
// an error; it used to be dropped without a word, so a typo produced a
// smaller conversion that looked like it had worked.
func selectSheets(excelFile string, inFile, wanted []string) ([]string, error) {
	if len(wanted) == 0 {
		return inFile, nil
	}
	var selected, missing []string
	for _, s := range wanted {
		if sliceutil.Contains(selected, s) {
			continue
		}
		if sliceutil.Contains(inFile, s) {
			selected = append(selected, s)
		} else {
			missing = append(missing, s)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("sheet(s) %s are not in %s (it has %s)",
			strings.Join(missing, ", "), excelFile, strings.Join(inFile, ", "))
	}
	return selected, nil
}
