## ADDED Requirements

### Requirement: Excel readers infer column types

`ReadExcelSheet` and `ReadExcel` SHALL infer each column's type by the rule CSV reading follows, after the header row and row names are taken out, and SHALL read a cell the file left out as empty. An optional `rawStrings` argument set to true SHALL keep every cell as text, and more than one value SHALL be an error.

#### Scenario: Numeric and text columns

- **WHEN** a sheet has a column of 1, 2, 3, a column of 1.5, 2.5, 3 and a column of apple, 12, pear
- **THEN** the columns load as `int64`, `float64` and text, in that order

#### Scenario: A gap in a numeric column

- **WHEN** a numeric column has cells Excel left out
- **THEN** those cells read as `NaN` and the column is `float64`

#### Scenario: Keeping text

- **WHEN** `rawStrings` is true
- **THEN** every cell stays the text Excel displays
