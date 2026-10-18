# [ csvxl ] Package

`csvxl` handles CSV/Excel conversion with optional encoding detection.

## Features

- Convert multiple CSV files into one Excel workbook (each CSV becomes a sheet)
- Append CSV files to an existing Excel workbook (sheets are overwritten if duplicated)
- Split Excel sheets into CSV files
- Batch helpers for converting entire directories
- Automatic encoding detection via `insyra.DetectEncoding`

## Installation

```bash
go get github.com/HazelnutParadise/insyra/csvxl
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/HazelnutParadise/insyra/csvxl"
)

func main() {
    sheets := []csvxl.SheetFromCSV{
        {Path: "jan.csv", Sheet: "January"},
        {Path: "feb.csv"}, // sheet "feb"
    }
    if err := csvxl.CSVToExcel(sheets, "output.xlsx"); err != nil {
        log.Fatal(err)
    }
}
```

## Supported encodings

Reading decodes UTF-8/ASCII, UTF-16 and UTF-32 (LE/BE, BOM-aware), Big5, GB18030/GBK/GB2312, Shift-JIS, ISO-2022-JP, EUC-JP, EUC-KR, every ISO-8859 part x/text ships, Windows-1250 through 1258, KOI8-R/U, IBM866 and Macintosh Roman, plus the usual aliases (`latin1`, `cp1252`, `sjis`, …). Separators and case do not matter: `ISO-8859-1`, `iso8859_1` and `ISO 8859 1` are the same.

A name outside that list is matched the way earlier releases did: one containing `utf8` reads as UTF-8, one containing `big5` as Big5, one containing `gb` as GB18030, one containing `utf16` as UTF-16. Separators and case do not matter here either, so `UTF-8-SIG`, `Big5-HKSCS` and `X-GBK` read the same as their lower-case spellings. For any other name `ReadCsvToString` returns an error naming the supported encodings, because it returns UTF-8 content. `CsvToExcel`, `AppendCsvToExcel` and `EachCsvToOneExcel` read the bytes without decoding instead, so pass the file's real encoding when it is not UTF-8. That includes four charsets the auto-detector can report but nothing here decodes, ISO-2022-KR, ISO-2022-CN, IBM424 and IBM420: with `Auto`, `ReadCsvToString` refuses a file detected as one of them, and the Excel conversions read it as raw bytes, with no error.

`Auto` detects the encoding from the file's first 8 KB. A UTF-32 byte-order mark is recognised before the UTF-16 one they share a prefix with.

## Encoding Constants

```go
const (
    UTF8 = "utf-8"
    Big5 = "big5"
    Auto = "auto"
)
```

`Auto` is the default. An empty string and `"auto"` in any case (`"AUTO"`, `"Auto"`) also mean detection, as they do for the core CSV readers; an empty string used to mean UTF-8 taken as-is, and `"AUTO"` used to be treated as an unknown encoding name. If detection fails, the function returns an error and you should pass a specific encoding.

## Main Functions

The uppercase `CSVToExcel`, `AppendCSVToExcel`, `CSVDirToExcel`,
`ReadCSVToString`, `ExcelToCSV` and `ExcelDirToCSV` names are the preferred
spellings. The earlier `CsvToExcel`, `AppendCsvToExcel`, `EachCsvToOneExcel`,
`ReadCsvToString`, `ExcelToCsv` and `EachExcelToCsv` names remain available,
deprecated, with their old signatures and behavior. The new names never pair a
CSV with a sheet by position: each CSV is named together with its sheet, and a
pairing that would put two CSVs in one sheet or two sheets in one CSV is an
error before anything is written. The deprecated names keep pairing by
position and do not refuse such a pairing.

### `CSVToExcel` and `AppendCSVToExcel`

```go
type SheetFromCSV struct {
    Path  string // the CSV file; ".csv" is appended when Path does not end in it
    Sheet string // the sheet's name; empty means the file's name without its extension
}
func CSVToExcel(sheets []SheetFromCSV, output string, csvEncoding ...string) error
func AppendCSVToExcel(sheets []SheetFromCSV, existingFile string, csvEncoding ...string) error
```

`CSVToExcel` writes a new workbook at `output`, one sheet per element, in the
order given. `AppendCSVToExcel` writes the sheets into an existing workbook and
replaces a sheet of the same name, as `AppendCsvToExcel` describes below.

Two elements that name the same sheet are an error that names the sheet and
both files, before any CSV is read: `CSVToExcel` writes no workbook and
`AppendCSVToExcel` leaves the workbook as it was. Sheet names are compared
without regard to case, as Excel compares them, so `{Path: "a.csv"}` and
`{Path: "b.csv", Sheet: "A"}` collide. Otherwise they behave as the deprecated
functions: a CSV that cannot be read is counted as failed and the others are
still converted.

### `SheetNames`

```go
func SheetNames(excelFile string) ([]string, error)
```

Returns the workbook's sheet names in workbook order, or an error naming the
file when it cannot be opened.

### `CsvToExcel`

```go
func CsvToExcel(csvFiles []string, sheetNames []string, output string, csvEncoding ...string) error
```

**Description:** Converts multiple CSV files into a new Excel workbook. `sheetNames[i]` names the sheet of `csvFiles[i]`; a missing or empty name uses the file's name. Deprecated: use `CSVToExcel`.

**Parameters:**

- `csvFiles`: File path to use. Type: `[]string`.
- `sheetNames`: Sheet name or list of sheet names. Type: `[]string`.
- `output`: Output location or file name. Type: `string`.
- `csvEncoding`: Variadic `string` values.

**Returns:**

- `error`: Error when the operation fails.

### `AppendCsvToExcel`

```go
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string) error
```

**Description:** Deprecated: use `AppendCSVToExcel`. `sheetNames[i]` names the sheet of `csvFiles[i]`. Appends CSV files as new sheets. An existing sheet with the same name is replaced by a fresh sheet before the CSV is written. What the workbook records about the sheet stays: its position among the sheets, whether it is hidden, and which sheet is active. Names and formulas elsewhere in the workbook are kept, including those that refer to it. Nothing stored in the old sheet itself survives: its cells and formulas, hidden rows, row heights, comments, hyperlinks, column widths, views and merged ranges all go, and so do the names defined for that sheet alone. The sheet name is matched without regard to case and the replaced sheet takes the name as given, so appending to `TARGET` renames a sheet called `Target`. This works even when it is the workbook's only sheet. Each CSV is read in full before its sheet is touched, so a CSV that cannot be read, because it is missing, its encoding cannot be detected or it is not valid CSV, leaves its sheet as it was, or adds no sheet when there was none; the other CSVs are still appended, and the error counts the files that failed.

**Parameters:**

- `csvFiles`: File path to use. Type: `[]string`.
- `sheetNames`: Sheet name or list of sheet names. Type: `[]string`.
- `existingFile`: File path to use. Type: `string`.
- `csvEncoding`: Variadic `string` values.

**Returns:**

- `error`: Error when the operation fails.

### `ExcelToCsv`

```go
func ExcelToCsv(excelFile string, outputDir string, csvNames []string, onlyContainSheets ...string) error
```

Each sheet becomes `<outputDir>/<sheet>.csv` (or the matching `csvNames` entry). A file name made from a sheet name is checked for each sheet before that sheet's CSV is written: a name containing `/` or `\`, or one that would not be a file directly inside `outputDir`, is rejected with an error, because sheet names come from the workbook and could otherwise escape `outputDir`. A `csvNames` entry is the caller's own choice and is used as given. A sheet named `.` or `..` is an ordinary name that becomes `..csv` or `...csv`. Each sheet is read fully before its CSV is created, so a sheet that cannot be read never truncates an existing CSV.

**Description:** Splits an Excel workbook into CSV files. Use `onlyContainSheets` to export selected sheets.

**Parameters:**

- `excelFile`: File path to use. Type: `string`.
- `outputDir`: Directory path to use. Type: `string`.
- `csvNames`: CSV file path or CSV-related value. Type: `[]string`.
- `onlyContainSheets`: Variadic `string` values.

**Returns:**

- `error`: Error when the operation fails.

### `EachCsvToOneExcel`

```go
func EachCsvToOneExcel(dir string, output string, encoding ...string) error
```

**Description:** Converts all CSV files in a directory into a single Excel workbook.

**Parameters:**

- `dir`: Directory path to use. Type: `string`.
- `output`: Output location or file name. Type: `string`.
- `encoding`: Variadic `string` values.

**Returns:**

- `error`: Error when the operation fails.

### `EachExcelToCsv`

```go
func EachExcelToCsv(dir string, outputDir string) error
```

**Description:** Converts all `.xlsx` files in a directory into CSV files.

**Parameters:**

- `dir`: Directory path to use. Type: `string`.
- `outputDir`: Directory path to use. Type: `string`.

**Returns:**

- `error`: Error when the operation fails.

### `ReadCsvToString`

```go
func ReadCsvToString(filePath string, encoding ...string) (string, error)
```

**Description:** Reads a CSV file and returns UTF-8 content. An encoding nothing here decodes, whether passed or detected by `Auto`, is an error naming the supported encodings; the file's raw bytes are never returned.

**Parameters:**

- `filePath`: File path to use. Type: `string`.
- `encoding`: Variadic `string` values.

**Returns:**

- `string`: Return value.
- `error`: Error when the operation fails.

`ReadCSVToString` is the preferred spelling and has the same signature and behavior.

### `ExcelToCSV`

```go
type ExcelToCSVOptions struct {
    Sheets        []string          // sheets to convert; empty means every sheet
    FileNames     map[string]string // sheet name -> CSV file name
    AllowFormulas bool
}
func ExcelToCSV(excelFile, outputDir string, options ...ExcelToCSVOptions) error
func ExcelDirToCSV(dir, outputDir string, options ...ExcelToCSVOptions) error
```

An empty `Sheets` converts every worksheet. A requested worksheet that is not
in the workbook returns an error. At most one options value may be supplied.
Formula-like text beginning with `=`, `+`, `-` or `@` is prefixed with an
apostrophe unless it is a number written in decimal digits (`-12`, `+3.5`,
`-1e309`) or `AllowFormulas` is true; `+Inf` and `-NaN` are guarded.

`CSVToExcel` and `AppendCSVToExcel` open an input path as it is written, so a CSV named `export.txt`, `DATA.CSV` or `plain` is read as itself. `.csv` is appended and tried only when the path names no file, or names a directory, and when both `x` and `x.csv` exist `x` is read.

`ExcelToCSV` writes each sheet as its name plus `.csv`, unless `FileNames` maps
the sheet to another name. A `FileNames` value keeps an extension it already
has (`report.txt` stays `report.txt`) and gets `.csv` when it has none. A key
that is not among the sheets converted is an error naming it, so a misspelled
sheet is not silently ignored.

```go
// Q1.csv, second.csv and Q3.csv
err := csvxl.ExcelToCSV("book.xlsx", "out", csvxl.ExcelToCSVOptions{
    FileNames: map[string]string{"Q2": "second"},
})
```

`ExcelDirToCSV` writes each sheet of `Book.xlsx` as `Book_<sheet>.csv` and
refuses `FileNames`, which cannot name the files of several workbooks.

Every file name is checked before the first CSV is written. A name holding a
path separator, and two sheets that would be written to one file, compared
without regard to case as macOS and Windows compare file names, are an error
and nothing is written. For `ExcelDirToCSV` this covers every workbook in the
directory at once: `a.xlsx`'s sheet `b_c` and `a_b.xlsx`'s sheet `c` would both
be `a_b_c.csv`.

`ExcelToCsv` and `EachExcelToCsv` keep their old behavior: no formula guard,
`csvNames[i]` naming the file of the i-th converted sheet, `.csv` appended to
any caller-given name that does not end in `.csv` (`report.txt` becomes
`report.txt.csv`), a caller-given relative path used as given, and no check
for two sheets written to one file.

### `insyra.DetectEncoding`

```go
func insyra.DetectEncoding(csvFile string) (string, error)
```

**Description:** Detects file encoding using BOM checks, UTF-8 validation, and `chardet` fallback.

**Parameters:**

- `csvFile`: File path to use. Type: `string`.

**Returns:**

- `string`: Return value.
- `error`: Error when the operation fails.

## Errors

Errors wrap the underlying cause, so `errors.Is(err, os.ErrNotExist)` and similar checks work through them. Output directories are created with mode 0755.
