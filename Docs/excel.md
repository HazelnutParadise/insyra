# [ excel ] Package

The `excel` package reads an Excel workbook into a `DataTable` and writes a `DataTable` as a sheet of a workbook.

It is a package of its own so that a program that only uses the core `insyra` package does not compile the Excel library into its binary: measured on linux/amd64, a program calling only `insyra.NewDataList(1, 2).Sum()` went from 10.17 MB to 6.68 MB. Converting CSV files to workbooks and back without building a table is [csvxl](csvxl.md)'s job.

```go
import "github.com/HazelnutParadise/insyra/excel"
```

The functions take no context: the Excel library cannot stop opening or decompressing a workbook part-way, so a context could not change when a call returns. Every function takes at most one options value; giving more than one is an error, never "first wins".

## Table of Contents

- [Data Structures](#data-structures)
  - [ReadOptions](#readoptions)
  - [WriteOptions](#writeoptions)
  - [SheetExistsPolicy](#sheetexistspolicy)
- [Main Functions](#main-functions)
  - [Read](#read)
  - [ReadFrom](#readfrom--from-any-source)
  - [Write](#write)
  - [WriteTo](#writeto--to-any-destination)
  - [SheetNames](#sheetnames)
- [Examples](#examples)
- [Moved from the core package](#moved-from-the-core-package)

## Data Structures

### ReadOptions

```go
type ReadOptions struct {
    Sheet       string // sheet to read; "" means the first sheet of the workbook
    NoHeaderRow bool   // the sheet has no row of column names, so its first row is data
    HasRowNames bool   // the first column holds row names
    RawStrings  bool   // keep every cell as the text Excel displays
}
```

The zero value reads the first sheet, takes its first row as the column names, keeps no row names and infers each column's type.

### WriteOptions

```go
type WriteOptions struct {
    Sheet         string            // sheet to write; "" means "Sheet1"
    NoHeaderRow   bool              // leave out the row of column names, which is written by default
    HasRowNames   bool              // write the row names as the first column
    IfSheetExists SheetExistsPolicy // SheetExistsFail (default) or SheetExistsReplace
}
```

The zero value writes the sheet `Sheet1` with a row of column names and without row names, and refuses to overwrite a sheet that is already there.

### SheetExistsPolicy

```go
type SheetExistsPolicy int

const (
    SheetExistsFail    SheetExistsPolicy = iota // refuse and leave the file untouched (the zero value)
    SheetExistsReplace                          // discard the old sheet and write the table in its place
)

var ErrSheetExists error // match it with errors.Is
```

`SheetExistsPolicy` prints its constant's name, and an unknown value as `SheetExistsPolicy(n)`.

## Main Functions

### Read

```go
func Read(path string, opts ...ReadOptions) (*insyra.DataTable, error)
```

**Description:** Reads one sheet of the workbook at `path` into a new `DataTable`. `Sheet` empty means the first sheet. At most one `ReadOptions` may be given.

A column is read the way a CSV column is: all whole numbers become `int64`, all numbers become `float64`, and any other value keeps the whole column as text. An empty cell in a numeric column becomes `NaN`, so a column of whole numbers with a gap is `float64`.

The reader sees each cell as Excel displays it, so `1,234`, `50%`, `$12` and dates stay text, and one `#N/A` keeps its whole column as text. A code such as `007` becomes `7`; set `RawStrings` to `true` to keep every cell as text, `007` included.

A workbook may expand to at most 512 MB while it is read, which keeps a small file that decompresses into more memory than the host has (a zip bomb) from exhausting memory; such a file is refused.

**Returns:**

- `*insyra.DataTable`: A new table with the sheet's data.
- `error`: An error if the file cannot be opened, the sheet does not exist, the workbook has no sheet, or the data cannot be processed.

**Example:**

```go
dt, err := excel.Read("data.xlsx", excel.ReadOptions{Sheet: "Sheet1", HasRowNames: true})
if err != nil {
    log.Fatal(err)
}
dt.Show()

// The first sheet, first row as column names, types inferred.
dt, err = excel.Read("data.xlsx")
```

### ReadFrom — from any source

```go
func ReadFrom(r io.Reader, opts ...ReadOptions) (*insyra.DataTable, error)
```

**Description:** Reads one sheet of a workbook that is not a file on disk — an upload, an HTTP response, a zip entry, bytes in memory — the same way `Read` reads a path, with the same 512 MB limit on how far the workbook may expand. It does not close `r`.

**Example:**

```go
resp, err := http.Get("https://example.com/report.xlsx")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
dt, err := excel.ReadFrom(resp.Body, excel.ReadOptions{Sheet: "2026"})
```

### Write

```go
func Write(dt insyra.IDataTable, path string, opts ...WriteOptions) error
```

**Description:** Writes the table as one sheet of the workbook at `path` and leaves every other sheet as it was. With no other fields set, the column names are written as the first row and there are no row names. If the file does not exist, a new workbook is created. If the workbook lacks the sheet, the sheet is added after the existing ones. A nil table is an error.

If the sheet already exists, the default is to refuse: the call returns an error matching `ErrSheetExists` and the file is not touched. With `IfSheetExists: excel.SheetExistsReplace`, the old sheet is thrown away, comments, tables, drawings, hidden rows and columns and merged ranges included, the widths of its visible columns and its view (zoom, frozen panes) carry over, and the table is written in its place, in the same position among the other sheets and hidden again if it was hidden. Formulas on other sheets that point at it by name then read the new values, and the names other sheets define stay with them; names defined for the replaced sheet alone go with it. Sheet names are matched the way Excel matches them, without regard to case, so `data` and `Data` are the same sheet, and the replaced sheet takes the name as given.

Numbers, booleans and `time.Time` values are written as Excel values, not text, so they sort and sum in a spreadsheet. A time keeps its wall-clock reading but not its time zone, because Excel does not store one. Anything else is written as the same text `ToCSV` would write. The workbook is written to a temporary file and renamed into place, so a failure never leaves a damaged workbook behind. The path must end in an extension Excel's current format uses (`.xlsx`, `.xlsm`, …); the legacy `.xls` format cannot be written.

**Example:** keep one sheet per year in a single report.

```go
opts := excel.WriteOptions{Sheet: "2025"}
err := excel.Write(sales2025, "report.xlsx", opts)
if errors.Is(err, excel.ErrSheetExists) {
    // 2025 is already there. Overwrite it on purpose; 2023 and 2024 stay.
    opts.IfSheetExists = excel.SheetExistsReplace
    err = excel.Write(sales2025, "report.xlsx", opts)
}
```

### WriteTo — to any destination

```go
func WriteTo(dt insyra.IDataTable, w io.Writer, opts ...WriteOptions) error
```

**Description:** Writes the table as a new one-sheet workbook to any destination, such as an HTTP response, a zip entry or a buffer. The cells are written the same way `Write` writes them. `IfSheetExists` is ignored, because the workbook is always new. It does not close `w`.

**Example:**

```go
http.HandleFunc("/report.xlsx", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    if err := excel.WriteTo(dt, w, excel.WriteOptions{Sheet: "report"}); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
})
```

### SheetNames

```go
func SheetNames(path string) ([]string, error)
```

**Description:** Lists the sheets of the workbook at `path` in the order the workbook keeps them, hidden sheets included. Use it to choose `ReadOptions.Sheet`, or the sheets and file names `csvxl.ExcelToCSV` takes. It replaces `csvxl.SheetNames`.

**Example:**

```go
names, err := excel.SheetNames("report.xlsx")
if err != nil {
    log.Fatal(err)
}
for _, name := range names {
    dt, err := excel.Read("report.xlsx", excel.ReadOptions{Sheet: name})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(name, dt.NumRows())
}
```

## Examples

### Round trip

```go
package main

import (
    "log"

    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/excel"
)

func main() {
    dt := insyra.NewDataTable(
        insyra.NewDataList("a", "b", "c").SetName("name"),
        insyra.NewDataList(1, 2, 3).SetName("count"),
    )
    if err := excel.Write(dt, "out.xlsx", excel.WriteOptions{Sheet: "data"}); err != nil {
        log.Fatal(err)
    }
    back, err := excel.Read("out.xlsx", excel.ReadOptions{Sheet: "data"})
    if err != nil {
        log.Fatal(err)
    }
    back.Show()
}
```

## Moved from the core package

These names used to live in the root `insyra` package and no longer exist there:

| Before | Now |
|---|---|
| `insyra.ReadExcelSheet(path, sheet, rowNames, headerRow, rawStrings...)` | `excel.Read(path, excel.ReadOptions{Sheet: sheet, HasRowNames: rowNames, NoHeaderRow: !headerRow, RawStrings: raw})` |
| `insyra.ReadExcel(r, sheet, rowNames, headerRow, rawStrings...)` | `excel.ReadFrom(r, excel.ReadOptions{...})` |
| `dt.ToExcel(path, insyra.ExcelWriteOptions{...})` | `excel.Write(dt, path, excel.WriteOptions{...})` |
| `dt.WriteExcel(w, opts)` | `excel.WriteTo(dt, w, opts)` |
| `insyra.ExcelWriteOptions`, `SheetExistsPolicy`, `SheetExistsFail`, `SheetExistsReplace`, `ErrSheetExists` | the same names in `excel` |
| `insyra.ExcelReadOptions()` | removed (internal) |
| `csvxl.SheetNames` | `excel.SheetNames` |
