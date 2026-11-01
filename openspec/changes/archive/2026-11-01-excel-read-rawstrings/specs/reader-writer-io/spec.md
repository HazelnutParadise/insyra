## ADDED Requirements

### Requirement: Excel readers let the caller choose text or typed columns

`ReadExcelSheet` and `ReadExcel` SHALL take an optional `rawStrings`: true SHALL keep every cell as text, false SHALL type columns by the rule CSV reading follows, and more than one value SHALL be an error. Left out, every cell SHALL stay text and one warning SHALL say columns are typed by default from 0.4.

#### Scenario: Left out

- **WHEN** a sheet holds a column of numbers and `rawStrings` is left out
- **THEN** the cells stay text

#### Scenario: False

- **WHEN** a sheet holds a column of whole numbers and `rawStrings` is false
- **THEN** the cells are `int64`
