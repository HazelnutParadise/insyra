# read-write-names Specification

## Purpose
How the file readers and writers are named and what their options mean by default. Each has one name taking one optional options struct, and the zero value of every options struct is the common file, so the plain call does the usual thing and a setting is only written when the file differs.

## Requirements
### Requirement: A reader or writer with no options handles the common file

`ReadCSVFile`, `ReadCSVString`, `ReadCSV`, `StreamCSV`, `ToCSV`, `WriteCSV` and `ToExcel` SHALL treat the zero value of their options as a file whose first row names the columns and that has no row-name column. `NoHeaderRow` SHALL say the file has no header row and `HasRowNames` that its first column holds row names, in every options struct that reads or writes a table.

#### Scenario: Reading a file with a header

- **WHEN** `ReadCSVFile(path)` reads `name,age\nAmy,30\n`
- **THEN** the columns are `name` and `age` and there is one row

#### Scenario: Writing a table

- **WHEN** `dt.ToCSV(path)` writes a table with a column `a`
- **THEN** the first line of the file is `a`

### Requirement: The old names keep their old meaning for one release

`ReadCSV_File`, `ReadCSV_FileWithOptions`, `ReadCSV_String`, `ReadCSV_StringWithOptions`, `ReadJSON_File`, `ToCSVWithOptions`, `ToJSON_Bytes` and `ToJSON_String` SHALL remain, marked Deprecated, and SHALL return what they returned before, positional bools included.

#### Scenario: The old reader without a header

- **WHEN** `ReadCSV_File(path, false, false)` reads a two-line file
- **THEN** both lines are data rows

