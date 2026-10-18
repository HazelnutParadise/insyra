# csvxl-sheet-pairing Specification

## Purpose
How `csvxl` pairs a CSV file with a sheet: each name travels with what it names, never by position, and a pairing that would put two CSVs in one sheet or two sheets in one CSV is refused before anything is written.
## Requirements
### Requirement: A CSV and its sheet are named together

`CSVToExcel` and `AppendCSVToExcel` SHALL take one `[]SheetFromCSV`, each element naming a CSV file (`Path`) and the sheet it is written to (`Sheet`). An empty `Sheet` SHALL be the CSV file's name without its extension. The sheets SHALL appear in the workbook in the order of the slice.

#### Scenario: Named sheets
- **WHEN** `CSVToExcel([]SheetFromCSV{{Path: "jan.csv", Sheet: "Sales"}, {Path: "feb.csv"}}, out)` is called
- **THEN** `out` has the sheets `Sales` and `feb`, in that order, holding `jan.csv` and `feb.csv`

### Requirement: Two CSVs aimed at one sheet are refused

When two elements resolve to the same sheet name, compared without regard to case, `CSVToExcel` and `AppendCSVToExcel` SHALL return an error naming the sheet and both files before reading any CSV: `CSVToExcel` SHALL write no file, and `AppendCSVToExcel` SHALL leave the workbook as it was. The deprecated `CsvToExcel` and `AppendCsvToExcel` SHALL do the same.

#### Scenario: Two files, one sheet name
- **WHEN** `jan.csv` and `feb.csv` are both given the sheet `Sales`
- **THEN** an error names `Sales`, `jan.csv` and `feb.csv`, and no workbook is written

#### Scenario: A default name collides with a given one
- **WHEN** `{Path: "a.csv"}` and `{Path: "b.csv", Sheet: "A"}` are given
- **THEN** an error names the sheet `a`, and no workbook is written

#### Scenario: Appending
- **WHEN** `AppendCSVToExcel` is given two CSVs for the sheet `Z`
- **THEN** an error is returned and the workbook's bytes are unchanged

### Requirement: A CSV file name is chosen by sheet name

`ExcelToCSV` SHALL take no file-name parameter. `ExcelToCSVOptions.FileNames` SHALL map a sheet name to the CSV file name written for it; a sheet with no entry, or an empty one, SHALL be written as its name plus `.csv`. A key that is not among the sheets being converted SHALL be an error naming it and the sheets that are. `ExcelDirToCSV` SHALL refuse a non-empty `FileNames`.

#### Scenario: Naming one sheet's file
- **WHEN** a workbook with `Q1`, `Q2` and `Q3` is converted with `FileNames: {"Q2": "second"}`
- **THEN** `Q1.csv`, `second.csv` and `Q3.csv` are written

#### Scenario: A key for a sheet that is not converted
- **WHEN** `Sheets` is `["Q1"]` and `FileNames` has the key `Q2`
- **THEN** an error names `Q2`, and no CSV is written

#### Scenario: FileNames in a directory conversion
- **WHEN** `ExcelDirToCSV` is given a non-empty `FileNames`
- **THEN** an error is returned and no CSV is written

### Requirement: Two sheets aimed at one CSV file are refused

When two sheets would be written to the same file, compared without regard to case, `ExcelToCSV` and `ExcelDirToCSV` SHALL return an error naming the file and both sheets before writing any CSV. The deprecated `ExcelToCsv` SHALL do the same.

#### Scenario: Two equal file names
- **WHEN** `FileNames` gives `Q1` and `Q2` the name `same`
- **THEN** an error names `same.csv`, `Q1` and `Q2`, and no CSV is written

#### Scenario: A given name collides with a default one
- **WHEN** `FileNames` gives `Q1` the name `Q2.csv` and `Q2` is also converted
- **THEN** an error is returned and no CSV is written

#### Scenario: Two workbooks in one directory
- **WHEN** `a.xlsx` has the sheet `b_c` and `a_b.xlsx` has the sheet `c`
- **THEN** `ExcelDirToCSV` returns an error naming `a_b_c.csv`, and no CSV is written

### Requirement: The sheets of a workbook can be listed

`SheetNames(excelFile)` SHALL return the workbook's sheet names in workbook order, or an error naming the file when it cannot be opened.

#### Scenario: Listing
- **WHEN** a workbook holds `Q1`, `Q2` and `Q3`
- **THEN** `SheetNames` returns `[Q1 Q2 Q3]`

