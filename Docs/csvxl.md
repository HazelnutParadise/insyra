# [ csvxl ] Package

`csvxl` is a Go package for working with CSV and Excel files. It allows you to convert multiple CSV files into an Excel workbook or append CSV files to an existing Excel file. The package supports custom sheet names and automatically handles various encodings, including Big5 and UTF-8.

## Features

- **CSV to Excel conversion**: Convert multiple CSV files into an Excel file, where each CSV becomes a new sheet.
- **Append to Excel**: Append CSV files to an existing Excel workbook, with support for custom sheet names and the option to overwrite existing sheets.
- **Automatic encoding handling**: Automatically detects and handles file encodings such as Big5 and UTF-8, ensuring correct display of characters.
- **Custom sheet names**: You can specify custom sheet names for each CSV file. If no sheet name is provided, the CSV file name is used as the default.

---

## Installation

You can install the package using `go get`:

```bash
go get github.com/HazelnutParadise/csvxl
```

---

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

    csvxl.CsvToExcel(csvFiles, sheetNames, output)
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

---

## Function Descriptions

### 1. `CsvToExcel`

```go
func CsvToExcel(csvFiles []string, sheetNames []string, output string)
```

**Description**: Converts multiple CSV files into a new Excel file.

- `csvFiles`: A list of CSV file paths.
- `sheetNames`: Custom sheet names corresponding to each CSV file. If not provided, the CSV filename will be used as the sheet name.
- `output`: The name of the output Excel file.

### 2. `AppendCsvToExcel`

```go
func AppendCsvToExcel(csvFiles []string, sheetNames []string, existing string)
```

**Description**: Appends multiple CSV files to an existing Excel workbook.

>[!NOTE]
>If the sheet is exists, it will be overwritten.

- `csvFiles`: A list of CSV file paths.
- `sheetNames`: Custom sheet names corresponding to each CSV file. If not provided, the CSV filename will be used as the sheet name.
- `existing`: The name of the existing Excel file.

---

## Encoding Support

`csvxl` automatically handles different CSV file encodings, including UTF-8 and Big5. This ensures that all characters (including non-English ones like Chinese) are correctly written into the Excel file.

---

## Error Handling

All functions log and skip files that cannot be processed, while continuing with the rest of the files. Results and errors are logged, allowing you to track which files succeeded or failed.

---

For any issues or requests, feel free to contact us or submit an issue on the GitHub repository.
