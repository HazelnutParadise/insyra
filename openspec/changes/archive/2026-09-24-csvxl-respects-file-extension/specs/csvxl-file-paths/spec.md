## ADDED Requirements

### Requirement: An input path that names a file is read as written

`CSVToExcel` and `AppendCSVToExcel` (and the `Csv` spellings) SHALL open an input path as given when it names a file. Only when the path names no file, or names a directory, SHALL `.csv` be appended and tried. When both `x` and `x.csv` are files, `x` SHALL be read.

#### Scenario: A name that does not end in .csv
- **WHEN** a CSV named `export.txt`, `DATA.CSV` or `plain` is passed
- **THEN** it is read as itself

#### Scenario: The extension was left off
- **WHEN** `data` is passed and only `data.csv` exists
- **THEN** `data.csv` is read

#### Scenario: Both exist
- **WHEN** `x` and `x.csv` both exist
- **THEN** `x` is read

#### Scenario: A directory of the same name
- **WHEN** `data` is a directory and `data.csv` is a file
- **THEN** `data.csv` is read
