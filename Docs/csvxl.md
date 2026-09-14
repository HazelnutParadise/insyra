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
    csvFiles := []string{"file1.csv", "file2.csv"}
    if err := csvxl.CsvToExcel(csvFiles, nil, "output.xlsx"); err != nil {
        log.Fatal(err)
    }
}
```

## Supported encodings

Reading decodes UTF-8/ASCII, UTF-16 and UTF-32 (LE/BE, BOM-aware), Big5, GB18030/GBK/GB2312, Shift-JIS, ISO-2022-JP, EUC-JP, EUC-KR, every ISO-8859 part x/text ships, Windows-1250 through 1258, KOI8-R/U, IBM866 and Macintosh Roman — every charset the auto-detector can report, plus the usual aliases (`latin1`, `cp1252`, `sjis`, …). Separators and case do not matter: `ISO-8859-1`, `iso8859_1` and `ISO 8859 1` are the same.

A name outside that list is matched the way earlier releases did: one containing `big5` reads as Big5, one containing `gb` as GB18030, one containing `utf-16` as UTF-16. Any other name reads the bytes without decoding, so pass the file's real encoding when it is not UTF-8.

`Auto` detects the encoding from the file's first 8 KB. A UTF-32 byte-order mark is recognised before the UTF-16 one they share a prefix with.

## Encoding Constants

```go
const (
    UTF8 = "utf-8"
    Big5 = "big5"
    Auto = "auto"
)
```

`Auto` is the default. If detection fails, the function returns an error and you should pass a specific encoding.

## Main Functions

### `CsvToExcel`

```go
func CsvToExcel(csvFiles []string, sheetNames []string, output string, csvEncoding ...string) error
```

**Description:** Converts multiple CSV files into a new Excel workbook. If `sheetNames` is empty, filenames are used.

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

**Description:** Appends CSV files as new sheets. An existing sheet with the same name is cleared in place before the CSV is written: every old cell value and formula is removed, including cells outside the range of the new CSV, while the sheet keeps its position among the sheets and its sheet-level settings such as column widths, views and merged ranges. This works even when it is the workbook's only sheet.

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

Each sheet becomes `<outputDir>/<sheet>.csv` (or the matching `csvNames` entry). The file name actually used is checked for each sheet before that sheet's CSV is written: a name containing `/` or `\`, or one that would not be a file directly inside `outputDir`, is rejected with an error, because sheet names come from the workbook and could otherwise escape `outputDir`. A `csvNames` entry is checked in place of the sheet name it replaces, and a sheet named `.` or `..` is an ordinary name that becomes `..csv` or `...csv`. Each sheet is read fully before its CSV is created, so a sheet that cannot be read never truncates an existing CSV.

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

**Description:** Reads a CSV file and returns UTF-8 content.

**Parameters:**

- `filePath`: File path to use. Type: `string`.
- `encoding`: Variadic `string` values.

**Returns:**

- `string`: Return value.
- `error`: Error when the operation fails.

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
