## MODIFIED Requirements

### Requirement: A source path is read as written first

`CSVToExcel` and `AppendCSVToExcel` SHALL open each source path as given when it names an existing file. Only when it does not exist, or names a directory, SHALL they append `.csv` and try that path. When neither exists the failure SHALL name both paths.

#### Scenario: A CSV whose name does not end in .csv
- **WHEN** the source is `export.txt` or `DATA.CSV` and that file exists
- **THEN** that file is read

#### Scenario: The extension was left off
- **WHEN** the source is `data` and only `data.csv` exists
- **THEN** `data.csv` is read

#### Scenario: Both files exist
- **WHEN** the source is `x` and both `x` and `x.csv` exist
- **THEN** `x` is read

#### Scenario: A directory of the same name
- **WHEN** the source is `data`, `data` is a directory and `data.csv` exists
- **THEN** `data.csv` is read

#### Scenario: Nothing to read
- **WHEN** neither the path nor the path with `.csv` exists
- **THEN** the failure names both paths

### Requirement: An output name keeps the extension it was given

`ExcelToCSV` SHALL use a `FileNames` value as written when it has an extension of any case, and SHALL append `.csv` only to a value with no extension.

#### Scenario: A name with its own extension
- **WHEN** `FileNames` maps a sheet to `report.txt` or `REPORT.CSV`
- **THEN** the file written is `report.txt` or `REPORT.CSV`

#### Scenario: A bare name
- **WHEN** `FileNames` maps a sheet to `report`
- **THEN** the file written is `report.csv`
