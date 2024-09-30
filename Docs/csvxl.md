# [ csvxl ] Package

`csvxl` is a Go package designed for working with CSV and Excel files. It allows you to convert multiple CSV files into an Excel workbook or append CSV files to an existing Excel file.

## Features

- **CSV to Excel conversion**: Convert multiple CSV files into an Excel file, where each CSV becomes a new sheet.
- **Append to Excel**: Append CSV files to an existing Excel workbook, with support for custom sheet names and the option to overwrite existing sheets.
- **Custom sheet names**: You can specify custom sheet names for each CSV file. If no sheet name is provided, the CSV file name is used as the default.
- **Automatic directory creation**: When splitting an Excel file into multiple CSV files, the target directory is automatically created if it does not exist.

> [!NOTE]
> Currently only support `UTF-8` and `Big5` encoding.

## Installation

You can install the package using `go get`:

```bash
go get github.com/HazelnutParadise/csvxl
```

## Usage Examples

### 1. Convert multiple CSV files into a new Excel file

```go
package main

import (
    "github.com/HazelnutParadise/csvxl"
)

func main() {
    csvFiles := []string{"file1.csv", "file2.csv", "file3.csv"}
    sheetNames := []string{"Sheet1", "Sheet2", "Sheet3"} // Optional: If not provided, CSV filenames will be used as sheet names
    output := "output.xlsx"

    csvxl.CsvToExcel(csvFiles, sheetNames, output, csvxl.UTF8)
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

### 2. `AppendCsvToExcel`

```go
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string)
```

**Description**: Appends multiple CSV files to an existing Excel workbook.

- `csvFiles`: A list of CSV file paths.
- `sheetNames`: Custom sheet names corresponding to each CSV file. If not provided, the CSV filename will be used as the sheet name.
- `existingFile`: The name of the existing Excel file.

### 3. `ExcelToCsv`

```go
func ExcelToCsv(excelFile string, outputDir string, csvNames []string, onlyContainSheets ...string)
```

**Description**: Splits each sheet in an Excel file into individual CSV files.

- `excelFile`: The path to the input Excel file.
- `outputDir`: The directory where the split CSV files will be saved. The directory will be created automatically if it does not exist.
- `csvNames`: Custom CSV file names. If not provided, the sheet name will be used as the default CSV file name.
- `onlyContainSheets`(optional): Only convert the specified sheets. If not provided, all sheets will be converted.

## Encoding Support

> [!NOTE]
> Currently only support `UTF-8` and `Big5` encoding.

`csvxl` supports different CSV file encodings when reading(converting to Excel), including UTF-8 and Big5. This ensures that all characters (including non-English ones like Chinese) are correctly written into the Excel file or split into CSV files.

However, when writing to CSV files, only `UTF-8` encoding is supported(and default).

## Error Handling

All functions log and skip files that cannot be processed while continuing with the remaining files. The results and errors are logged, allowing you to track which files succeeded or failed.

---

For any issues or requests, feel free to contact us or submit an issue on the GitHub repository.
