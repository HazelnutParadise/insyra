# csv-formula-guard Specification

## Purpose
What CSV output does with text a spreadsheet would run as a formula. The file is safe to open in a spreadsheet by default, numbers are never altered, and a program that must read back exactly what it wrote can turn the guard off.
## Requirements
### Requirement: CSV output guards formula-like text by default

`ToCSV` and `WriteCSV` SHALL prefix with a single quote any text that begins, after leading whitespace, with `=`, `+`, `-` or `@` and is not only a number, unless `AllowFormulas` is set. Numbers and text that is only a number SHALL be written unchanged.

#### Scenario: Formula-like text

- **WHEN** a table holding `=1+1`, `-2+3` and `@SUM(A1)` is written with no options
- **THEN** the file holds `'=1+1`, `'-2+3` and `'@SUM(A1)`

#### Scenario: Numbers

- **WHEN** a table holding the number `-5` and the text `+886912345678` is written with no options
- **THEN** both are written unchanged

#### Scenario: Turning the guard off

- **WHEN** the same table is written with `AllowFormulas: true`
- **THEN** every value is written exactly

### Requirement: Excel-to-CSV conversion is guarded the same way

`csvxl.ExcelToCsv` and `EachExcelToCsv` SHALL guard formula-like text exactly as the core CSV writer does unless `ExcelToCsvOptions.AllowFormulas` is set, and SHALL convert every sheet when `Sheets` is empty.

#### Scenario: A text cell holding a formula

- **WHEN** a workbook whose text cell holds `=1+1` is converted with no options
- **THEN** the CSV holds `'=1+1`

