## MODIFIED Requirements

### Requirement: A reader or writer with no options handles the common file

`ReadCSVFile`, `ReadCSVString`, `ReadCSV`, `StreamCSV`, `ToCSV`, `WriteCSV`, `excel.Read`, `excel.ReadFrom`, `excel.Write` and `excel.WriteTo` SHALL treat the zero value of their options as a file whose first row names the columns and that has no row-name column. `NoHeaderRow` SHALL say the file has no header row and `HasRowNames` that its first column holds row names, in every options struct that reads or writes a table.

#### Scenario: Reading a file with a header

- **WHEN** `ReadCSVFile(path)` reads `name,age\nAmy,30\n`
- **THEN** the columns are `name` and `age` and there is one row

#### Scenario: Writing a table

- **WHEN** `dt.ToCSV(path)` writes a table with a column `a`
- **THEN** the first line of the file is `a`
