# excel-package Specification

## Purpose
Gives 0.3.x the `excel` package with the API it has on 0.4: `excel.Read`, `excel.Write`, their stream forms and `excel.SheetNames`, so a caller moves off the deprecated core Excel names once and the same code then builds on both lines.

## Requirements

### Requirement: Excel reading and writing live in the excel package

Reading a workbook into a DataTable and writing a DataTable as a sheet SHALL be provided by the package `excel`, through `Read`, `ReadFrom`, `SheetNames`, `Write` and `WriteTo`. On the 0.3.x line the root package's `ReadExcelSheet`, `ReadExcel`, `DataTable.ToExcel`, `DataTable.WriteExcel` and `ExcelWriteOptions`, and `csvxl.SheetNames`, SHALL keep their behaviour and SHALL be marked Deprecated, naming their `excel` replacement.

#### Scenario: The old names are deprecated
- **WHEN** the doc comment of `insyra.ReadExcelSheet` is read
- **THEN** it has a `Deprecated:` paragraph naming `excel.Read`

#### Scenario: A round trip
- **WHEN** a table with columns `a` (1, 2) and `b` (x, y) and row names r1, r2 is written with `excel.Write(dt, path, excel.WriteOptions{HasRowNames: true})` and read back with `excel.Read(path, excel.ReadOptions{HasRowNames: true})`
- **THEN** the columns are `a` and `b`, the cells are `int64` 1, 2 and the text x, y, and the second row is named r2

### Requirement: The excel functions take one optional options value

`Read`, `ReadFrom`, `Write` and `WriteTo` SHALL take their settings as an optional trailing `ReadOptions` or `WriteOptions`, SHALL treat the zero value as a sheet with a header row and without row names, and SHALL return an error naming the options type when given more than one.

#### Scenario: Reading without options
- **WHEN** `excel.Read(path)` reads a workbook whose first sheet is `Q3` and second is `Q1`
- **THEN** the table holds `Q3`'s data

#### Scenario: Two options values
- **WHEN** `excel.Write(dt, path, excel.WriteOptions{}, excel.WriteOptions{})` is called
- **THEN** it returns an error that names `WriteOptions` and writes nothing

### Requirement: The sheets of a workbook can be listed

`excel.SheetNames(path)` SHALL return the workbook's sheet names in workbook order, hidden sheets included, or an error naming the file when it cannot be opened.

#### Scenario: Listing
- **WHEN** a workbook holds `Q3` and then `Q1`
- **THEN** `SheetNames` returns `[Q3 Q1]`

### Requirement: Writing refuses a nil table

`excel.Write` and `excel.WriteTo` SHALL return an error for a nil table, including a nil pointer behind the interface, and SHALL NOT write anything.

#### Scenario: A nil table
- **WHEN** `excel.Write((*insyra.DataTable)(nil), path)` is called
- **THEN** it returns an error and no file is created
