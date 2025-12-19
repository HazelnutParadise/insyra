# [ csvxl ] Package

`csvxl` is a Go package designed for working with CSV and Excel files. It allows you to convert multiple CSV files into an Excel workbook or append CSV files to an existing Excel file.

## Features

- **CSV to Excel conversion**: Convert multiple CSV files into an Excel file, where each CSV becomes a new sheet.
- **Append to Excel**: Append CSV files to an existing Excel workbook, with support for custom sheet names and the option to overwrite existing sheets.
- **Custom sheet names**: You can specify custom sheet names for each CSV file. If no sheet name is provided, the CSV file name is used as the default.
- **Automatic directory creation**: When splitting an Excel file into multiple CSV files, the target directory is automatically created if it does not exist.
- **🆕 Automatic encoding detection**: Automatically detects the encoding of CSV files, supporting UTF-8, Big5, and other common encodings.

> [!NOTE]
> The package now supports automatic encoding detection by default. When no encoding is explicitly specified, it will automatically detect the encoding of CSV files. Supported encodings include UTF-8, Big5, ASCII, and others. Note: for certain Chinese encodings the detector may fall back to UTF-8 and return a warning-like error indicating the fallback; for unknown or unrecognized encodings `DetectEncoding` will return an error and callers should handle it (for example by specifying an explicit encoding).

## Installation

You can install the package using `go get`:

```bash
go get github.com/HazelnutParadise/csvxl
```

## Usage Examples

### 1. Convert multiple CSV files into a new Excel file (with auto-detection)

```go
package main

import (
    "log"
    "github.com/HazelnutParadise/csvxl"
)

func main() {
    csvFiles := []string{"file1.csv", "file2.csv", "file3.csv"}
    sheetNames := []string{"Sheet1", "Sheet2", "Sheet3"} // Optional: If not provided, CSV filenames will be used as sheet names
    output := "output.xlsx"

    // Auto-detection (default behavior)
    if err := csvxl.CsvToExcel(csvFiles, sheetNames, output); err != nil {
        log.Fatalf("CsvToExcel failed: %v", err)
    }

    // Or explicitly specify auto-detection
    if err := csvxl.CsvToExcel(csvFiles, sheetNames, output, csvxl.Auto); err != nil {
        log.Fatalf("CsvToExcel failed: %v", err)
    }

    // Or specify explicit encoding
    if err := csvxl.CsvToExcel(csvFiles, sheetNames, output, csvxl.UTF8); err != nil {
        log.Fatalf("CsvToExcel failed: %v", err)
    }
}
```

### 2. Detect encoding of a CSV file

```go
package main

import (
    "fmt"
    "log"
    "github.com/HazelnutParadise/csvxl"
)

func main() {
    encoding, err := csvxl.DetectEncoding("myfile.csv")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Detected encoding: %s\n", encoding)
}
```

### 2. Append multiple CSV files to an existing Excel file

```go
package main

import (
    "log"
    "github.com/HazelnutParadise/csvxl"
)

func main() {
    csvFiles := []string{"newfile1.csv", "newfile2.csv"}
    sheetNames := []string{"NewSheet1", "NewSheet2"} // Optional: You can specify custom sheet names for each CSV file
    existingExcel := "existing.xlsx"

    // Auto-detection (default behavior)
    if err := csvxl.AppendCsvToExcel(csvFiles, sheetNames, existingExcel); err != nil {
        log.Fatalf("AppendCsvToExcel failed: %v", err)
    }
}
```

### 3. Split an Excel file into multiple CSV files

```go
package main

import (
    "log"
    "github.com/HazelnutParadise/csvxl"
)

func main() {
    excelFile := "input.xlsx"
    outputDir := "csv_output"
    csvNames := []string{"custom1.csv", "custom2.csv"} // Optional: Specify names for the output CSV files

    if err := csvxl.ExcelToCsv(excelFile, outputDir, csvNames); err != nil {
        log.Fatalf("ExcelToCsv failed: %v", err)
    }
}
```

## Constants

### CsvEncoding

```go
const (
 UTF8 = "utf-8"
 Big5 = "big5"
 Auto = "auto"  // 🆕 New: Automatic encoding detection
)
```

## Function Descriptions

### 1. `CsvToExcel`

```go
func CsvToExcel(csvFiles []string, sheetNames []string, output string, csvEncoding ...string) error
```

**Description**: Converts multiple CSV files into a new Excel file.

- `csvFiles`: A list of CSV file paths.
- `sheetNames`: Custom sheet names corresponding to each CSV file. If not provided, the CSV filename will be used as the sheet name.
- `output`: The name of the output Excel file.
- `csvEncoding`(optional): The encoding of the CSV files. If not provided, automatic detection is used. Supported values: `csvxl.UTF8`, `csvxl.Big5`, `csvxl.Auto`.

### 2. `AppendCsvToExcel`

```go
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string) error
```

**Description**: Appends multiple CSV files to an existing Excel workbook.

- `csvFiles`: A list of CSV file paths.
- `sheetNames`: Custom sheet names corresponding to each CSV file. If not provided, the CSV filename will be used as the sheet name.
- `existingFile`: The name of the existing Excel file.
- `csvEncoding`(optional): The encoding of the CSV files. If not provided, automatic detection is used. Supported values: `csvxl.UTF8`, `csvxl.Big5`, `csvxl.Auto`.

### 3. `DetectEncoding` 🆕

```go
func DetectEncoding(csvFile string) (string, error)
```

**Description**: Automatically detects the encoding of a CSV file.

- `csvFile`: The path to the CSV file to analyze.
- Returns: The detected encoding as a string (e.g., "utf-8", "big5") and an error if detection fails.

### 4. `ExcelToCsv`

```go
func ExcelToCsv(excelFile string, outputDir string, csvNames []string, onlyContainSheets ...string) error
```

**Description**: Splits each sheet in an Excel file into individual CSV files.

- `excelFile`: The path to the input Excel file.
- `outputDir`: The directory where the split CSV files will be saved. The directory will be created automatically if it does not exist.
- `csvNames`: Custom CSV file names. If not provided, the sheet name will be used as the default CSV file name.
- `onlyContainSheets`(optional): Only convert the specified sheets. If not provided, all sheets will be converted.

> [!NOTE]
> The CSV file names will be in the format of "ExcelFileName_SheetName.csv".

### 5. `EachExcelToCsv`

```go
func EachExcelToCsv(dir string, outputDir string) error
```

**Description**: Splits each sheet in each Excel file in the given directory into individual CSV files.

- `dir`: The directory containing the input Excel files.
- `outputDir`: The directory where the split CSV files will be saved.

### 6. `EachCsvToOneExcel`

```go
func EachCsvToOneExcel(dir string, output string, encoding ...string) error
```

**Description**: Converts each CSV file in the given directory to an Excel file.

- `dir`: The directory containing the input CSV files.
- `output`: The name of the output Excel file.
- `encoding`(optional): The encoding of the CSV files. If not provided, automatic detection is used. Supported values: `csvxl.UTF8`, `csvxl.Big5`, `csvxl.Auto`.

## Encoding Support

> [!NOTE]
> 🆕 **NEW**: Automatic encoding detection is now supported! The package will automatically detect the encoding of CSV files when no explicit encoding is specified.

`csvxl` supports different CSV file encodings when reading (converting CSV to Excel), including:

- **UTF-8** (most common; used as a fallback for certain detected encodings such as some Chinese encodings)
- **Big5** (Traditional Chinese)
- **ASCII** (basic Latin characters)
- **Various other encodings** (automatically detected). Unknown or unrecognizable encodings will cause `DetectEncoding` to return an error.

### Automatic Detection

When no encoding is explicitly specified, the package uses advanced charset detection algorithms to automatically identify the encoding of your CSV files. This makes the package much easier to use as you don't need to worry about specifying the correct encoding manually. If detection cannot reliably determine an encoding, `DetectEncoding` will return an error — callers should either provide an explicit encoding (e.g., `csvxl.UTF8`) or handle the error appropriately.

### Manual Encoding Specification

You can still explicitly specify an encoding using the constants:

- `csvxl.Auto` - Automatic detection (default)
- `csvxl.UTF8` - Force UTF-8 encoding
- `csvxl.Big5` - Force Big5 encoding

### Output Encoding

When writing to CSV files (converting Excel to CSV), only `UTF-8` encoding is supported and used by default.

## Error Handling

All public functions now return an `error` value to indicate failures.

- Batch operations (`CsvToExcel`, `AppendCsvToExcel`) will attempt to process all provided files; if some files fail, the functions will return a non-nil error summarizing the number of failed files (e.g. `"2 files failed to convert"`). Successful conversions will still produce output files when possible.
- Operations that work on a single input or per-file basis (`ExcelToCsv`, `EachExcelToCsv`, `EachCsvToOneExcel`) will return an error immediately when a fatal failure occurs (e.g. cannot open file or cannot save a sheet).
- `DetectEncoding(csvFile)` returns `(string, error)`. It returns the detected encoding and an error if detection failed. For certain Chinese encodings (e.g., GBK/GB18030) it may return `UTF-8` and a non-nil error indicating a fallback; for unknown or unrecognized encodings it returns an error and an empty encoding — callers should treat this as a failure and handle it (e.g., by specifying `csvxl.UTF8` or skipping the file).

Please update your callers to check and handle returned errors (for example: `if err := csvxl.CsvToExcel(...); err != nil { // handle }`).

---

For any issues or requests, feel free to contact us or submit an issue on the GitHub repository.
