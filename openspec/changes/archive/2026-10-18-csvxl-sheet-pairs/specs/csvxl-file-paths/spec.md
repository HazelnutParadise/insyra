## MODIFIED Requirements

### Requirement: An output name keeps the extension it was given

`ExcelToCSV` SHALL use a `FileNames` value as written when it has an extension of any case, and SHALL append `.csv` only to a value with no extension. The deprecated `ExcelToCsv` SHALL treat its `csvNames` entries the same way.

#### Scenario: A name with its own extension
- **WHEN** `FileNames` maps a sheet to `report.txt` or `REPORT.CSV`
- **THEN** the file written is `report.txt` or `REPORT.CSV`

#### Scenario: A bare name
- **WHEN** `FileNames` maps a sheet to `report`
- **THEN** the file written is `report.csv`
