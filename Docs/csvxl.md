# [ csvxl ] Package

`csvxl` is a Go package designed for working with CSV and Excel files. It allows you to convert multiple CSV files into an Excel workbook or append CSV files to an existing Excel file.

## Features

- **CSV to Excel conversion**: Convert multiple CSV files into an Excel file, where each CSV becomes a new sheet.
- **Append to Excel**: Append CSV files to an existing Excel workbook, with support for custom sheet names and the option to overwrite existing sheets.
- **Custom sheet names**: You can specify custom sheet names for each CSV file. If no sheet name is provided, the CSV file name is used as the default.
- **Automatic directory creation**: When splitting an Excel file into multiple CSV files, the target directory is automatically created if it does not exist.
- **🆕 Automatic encoding detection**: Automatically detects the encoding of CSV files, supporting UTF-8, Big5, and other common encodings.

> [!NOTE]
> The package now supports automatic encoding detection by default. When no encoding is explicitly specified, it will automatically detect the encoding of CSV files. Supported encodings include UTF-8, Big5, ASCII, and various others with automatic fallback to UTF-8.

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
    "github.com/HazelnutParadise/csvxl"
)

func main() {
    csvFiles := []string{"file1.csv", "file2.csv", "file3.csv"}
    sheetNames := []string{"Sheet1", "Sheet2", "Sheet3"} // Optional: If not provided, CSV filenames will be used as sheet names
    output := "output.xlsx"

    // Auto-detection (default behavior)
    csvxl.CsvToExcel(csvFiles, sheetNames, output)
    
    // Or explicitly specify auto-detection
    csvxl.CsvToExcel(csvFiles, sheetNames, output, csvxl.Auto)
    
    // Or specify explicit encoding
    csvxl.CsvToExcel(csvFiles, sheetNames, output, csvxl.UTF8)
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
    "github.com/HazelnutParadise/csvxl"
)

func main() {
    csvFiles := []string{"newfile1.csv", "newfile2.csv"}
    sheetNames := []string{"NewSheet1", "NewSheet2"} // Optional: You can specify custom sheet names for each CSV file
    existingExcel := "existing.xlsx"

    // Auto-detection (default behavior)
    csvxl.AppendCsvToExcel(csvFiles, sheetNames, existingExcel)
}
```

### 3. Split an Excel file into multiple CSV files

```go
package main

import (
    "github.com/HazelnutParadise/csvxl"
)

func main() {
    excelFile := "input.xlsx"
    outputDir := "csv_output"
    csvNames := []string{"custom1.csv", "custom2.csv"} // Optional: Specify names for the output CSV files

    csvxl.ExcelToCsv(excelFile, outputDir, csvNames)
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
func CsvToExcel(csvFiles []string, sheetNames []string, output string, csvEncoding ...string)
```

**Description**: Converts multiple CSV files into a new Excel file.

- `csvFiles`: A list of CSV file paths.
- `sheetNames`: Custom sheet names corresponding to each CSV file. If not provided, the CSV filename will be used as the sheet name.
- `output`: The name of the output Excel file.
- `csvEncoding`(optional): The encoding of the CSV files. If not provided, automatic detection is used. Supported values: `csvxl.UTF8`, `csvxl.Big5`, `csvxl.Auto`.

### 2. `AppendCsvToExcel`

```go
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string)
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
func ExcelToCsv(excelFile string, outputDir string, csvNames []string, onlyContainSheets ...string)
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
func EachExcelToCsv(dir string, outputDir string)
```

**Description**: Splits each sheet in each Excel file in the given directory into individual CSV files.

- `dir`: The directory containing the input Excel files.
- `outputDir`: The directory where the split CSV files will be saved.

### 6. `EachCsvToOneExcel`

```go
func EachCsvToOneExcel(dir string, output string, encoding ...string)
```

**Description**: Converts each CSV file in the given directory to an Excel file.

- `dir`: The directory containing the input CSV files.
- `output`: The name of the output Excel file.
- `encoding`(optional): The encoding of the CSV files. If not provided, automatic detection is used. Supported values: `csvxl.UTF8`, `csvxl.Big5`, `csvxl.Auto`.

## Encoding Support

> [!NOTE]
> 🆕 **NEW**: Automatic encoding detection is now supported! The package will automatically detect the encoding of CSV files when no explicit encoding is specified.

`csvxl` supports different CSV file encodings when reading (converting CSV to Excel), including:

- **UTF-8** (most common, default fallback)
- **Big5** (Traditional Chinese)
- **ASCII** (basic Latin characters)
- **Various other encodings** (automatically detected with fallback to UTF-8)

### Automatic Detection
When no encoding is explicitly specified, the package uses advanced charset detection algorithms to automatically identify the encoding of your CSV files. This makes the package much easier to use as you don't need to worry about specifying the correct encoding manually.

### Manual Encoding Specification
You can still explicitly specify an encoding using the constants:
- `csvxl.Auto` - Automatic detection (default)
- `csvxl.UTF8` - Force UTF-8 encoding
- `csvxl.Big5` - Force Big5 encoding

### Output Encoding
When writing to CSV files (converting Excel to CSV), only `UTF-8` encoding is supported and used by default.

## Error Handling

All functions log and skip files that cannot be processed while continuing with the remaining files. The results and errors are logged, allowing you to track which files succeeded or failed.

---

For any issues or requests, feel free to contact us or submit an issue on the GitHub repository.
