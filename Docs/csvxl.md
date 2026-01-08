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

Converts multiple CSV files into a new Excel workbook. If `sheetNames` is empty, filenames are used.

### `AppendCsvToExcel`

```go
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string) error
```

Appends CSV files as new sheets. Existing sheets with the same name are overwritten.

### `ExcelToCsv`

```go
func ExcelToCsv(excelFile string, outputDir string, csvNames []string, onlyContainSheets ...string) error
```

Splits an Excel workbook into CSV files. Use `onlyContainSheets` to export selected sheets.

### `EachCsvToOneExcel`

```go
func EachCsvToOneExcel(dir string, output string, encoding ...string) error
```

Converts all CSV files in a directory into a single Excel workbook.

### `EachExcelToCsv`

```go
func EachExcelToCsv(dir string, outputDir string) error
```

Converts all `.xlsx` files in a directory into CSV files.

### `ReadCsvToString`

```go
func ReadCsvToString(filePath string, encoding ...string) (string, error)
```

Reads a CSV file and returns UTF-8 content.

### `insyra.DetectEncoding`

```go
func insyra.DetectEncoding(csvFile string) (string, error)
```

Detects file encoding using BOM checks, UTF-8 validation, and `chardet` fallback.
