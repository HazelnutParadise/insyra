## ADDED Requirements

### Requirement: Excel-to-CSV conversion is guarded the same way

`csvxl.ExcelToCsv` and `EachExcelToCsv` SHALL guard formula-like text exactly as the core CSV writer does unless `ExcelToCsvOptions.AllowFormulas` is set, and SHALL convert every sheet when `Sheets` is empty.

#### Scenario: A text cell holding a formula

- **WHEN** a workbook whose text cell holds `=1+1` is converted with no options
- **THEN** the CSV holds `'=1+1`
