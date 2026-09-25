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

Any other name is an error listing what is available. insyra will not copy bytes it cannot decode into a table, because the result would be cells that are not valid UTF-8 with nothing to say so.

`Auto` detects the encoding from the file's first 8 KB. A UTF-32 byte-order mark is recognised before the UTF-16 one they share a prefix with, and a sample too short to identify falls back to UTF-8 with a warning rather than failing the read.

## Encoding Constants

```go
const (
    UTF8 = "utf-8"
    Big5 = "big5"
    Auto = "auto"
)
```

`Auto` is the default. An empty string and `"auto"` in any case (`"AUTO"`, `"Auto"`) also mean detection, as they do for the core CSV readers; an empty string used to mean UTF-8 taken as-is, and `"AUTO"` used to fail. If detection fails, the function returns an error and you should pass a specific encoding.

## File names

`csvxl` adds `.csv` as a convenience and never over a name you wrote.

- **Reading** (`CsvToExcel`, `AppendCsvToExcel`): each path is opened as written when it names a file, so a CSV called `export.txt` or `DATA.CSV` is read as itself. Only when nothing is there, or a directory is, is `.csv` appended and tried, so `data` still reads `data.csv`. When both `x` and `x.csv` exist, `x` is read. When neither exists, the error names both paths it tried and `errors.Is(err, os.ErrNotExist)` holds.
- **Writing** (`ExcelToCsv`'s `csvNames`): a name that already has an extension, of any case, is used as written, so `report.txt` stays `report.txt`. A name with no extension gets `.csv`.

## Main Functions

### `CsvToExcel`

```go
func CsvToExcel(csvFiles []string, sheetNames []string, output string, csvEncoding ...string) error
```

**Description:** Converts multiple CSV files into a new Excel workbook. If `sheetNames` is empty, filenames are used.

Each CSV is read in full before its sheet is created. When a CSV cannot be read, or Excel rejects its sheet name, that file gets no sheet, and the other files are still converted and saved. The returned error then begins with a count such as `1 of 3 CSV files failed to convert` and lists each failed file with its cause on its own line. `errors.Is` works on it, for example with `os.ErrNotExist`. When every file fails, no workbook is written.

**Parameters:**

- `csvFiles`: Paths of the CSV files, read as described in [File names](#file-names). Type: `[]string`.
- `sheetNames`: Sheet name or list of sheet names. Type: `[]string`.
- `output`: Output location or file name. Type: `string`.
- `csvEncoding`: Variadic `string` values.

**Returns:**

- `error`: Error when the operation fails.

### `AppendCsvToExcel`

```go
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string) error
```

**Description:** Appends CSV files as new sheets. An existing sheet with the same name is deleted first and replaced in full, so nothing from the old sheet survives — including cells outside the range of the new CSV. This works even when it is the workbook's only sheet.

Each CSV is read in full before its sheet is replaced, so a CSV that cannot be read leaves the existing sheet of that name as it was. The other files are still appended, and the error lists the files that failed in the same form as `CsvToExcel`. When every file fails, the workbook file is not rewritten.

**Parameters:**

- `csvFiles`: Paths of the CSV files, read as described in [File names](#file-names). Type: `[]string`.
- `sheetNames`: Sheet name or list of sheet names. Type: `[]string`.
- `existingFile`: File path to use. Type: `string`.
- `csvEncoding`: Variadic `string` values.

**Returns:**

- `error`: Error when the operation fails.

### `ExcelToCsv`

```go
func ExcelToCsv(excelFile string, outputDir string, csvNames []string, onlyContainSheets ...string) error
```

Each sheet becomes `<outputDir>/<sheet>.csv`, or the matching `csvNames` entry named as described in [File names](#file-names). A sheet name that cannot be a single file name — it contains `/`, `\` or is `..` — is rejected with an error before any file is touched, because sheet names come from the workbook and could otherwise escape `outputDir`. Each CSV is read fully from the sheet first and written through a temporary file, so a failing sheet never truncates an existing CSV.

A name in `onlyContainSheets` that the workbook does not have is an error naming the sheets it does have, rather than a sheet that quietly does not appear in the output.

```go
```

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

**Description:** Converts all CSV files in a directory into a single Excel workbook through `CsvToExcel`, so a file that fails is skipped and reported the same way.

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
