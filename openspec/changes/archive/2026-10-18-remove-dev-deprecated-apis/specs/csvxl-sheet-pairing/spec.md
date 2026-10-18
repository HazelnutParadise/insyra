## MODIFIED Requirements

### Requirement: Two CSVs aimed at one sheet are refused

When two elements resolve to the same sheet name, compared without regard to case, `CSVToExcel` and `AppendCSVToExcel` SHALL return an error naming the sheet and both files before reading any CSV: `CSVToExcel` SHALL write no file, and `AppendCSVToExcel` SHALL leave the workbook as it was.

#### Scenario: Two files, one sheet name
- **WHEN** `jan.csv` and `feb.csv` are both given the sheet `Sales`
- **THEN** an error names `Sales`, `jan.csv` and `feb.csv`, and no workbook is written

#### Scenario: A default name collides with a given one
- **WHEN** `{Path: "a.csv"}` and `{Path: "b.csv", Sheet: "A"}` are given
- **THEN** an error names the sheet `a`, and no workbook is written

#### Scenario: Appending
- **WHEN** `AppendCSVToExcel` is given two CSVs for the sheet `Z`
- **THEN** an error is returned and the workbook's bytes are unchanged

### Requirement: Two sheets aimed at one CSV file are refused

When two sheets would be written to the same file, compared without regard to case, `ExcelToCSV` and `ExcelDirToCSV` SHALL return an error naming the file and both sheets before writing any CSV.

#### Scenario: Two equal file names
- **WHEN** `FileNames` gives `Q1` and `Q2` the name `same`
- **THEN** an error names `same.csv`, `Q1` and `Q2`, and no CSV is written

#### Scenario: A given name collides with a default one
- **WHEN** `FileNames` gives `Q1` the name `Q2.csv` and `Q2` is also converted
- **THEN** an error is returned and no CSV is written

#### Scenario: Two workbooks in one directory
- **WHEN** `a.xlsx` has the sheet `b_c` and `a_b.xlsx` has the sheet `c`
- **THEN** `ExcelDirToCSV` returns an error naming `a_b_c.csv`, and no CSV is written
