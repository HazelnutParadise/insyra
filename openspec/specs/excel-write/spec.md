# excel-write Specification

## Purpose
How a DataTable is written into an Excel workbook, from the library and from the CLI's `save`. A workbook holds sheets the table never came from, so a save writes only the sheet it names and never removes or overwrites another one without being asked.

## Requirements
### Requirement: A save touches only its own sheet

`ToExcel` SHALL write the table as the sheet named in its options, `Sheet1` when none is given. It SHALL create the workbook when the path does not exist, add the sheet when the workbook lacks it, and leave every other sheet as it was.

#### Scenario: A new workbook
- **WHEN** the path does not exist
- **THEN** a workbook is created holding one sheet with the table

#### Scenario: A workbook with other sheets
- **WHEN** the workbook holds sheets `2023` and `2024` and the table is saved as `2025`
- **THEN** it holds `2023`, `2024` and `2025`, and the first two are unchanged

### Requirement: An existing sheet is replaced only when asked

When the sheet already exists, `ToExcel` SHALL refuse with an error matching `ErrSheetExists` and leave the file unchanged, unless `IfSheetExists` is `SheetExistsReplace`, in which case it SHALL replace that sheet's contents in the sheet's original position.

#### Scenario: Saving over a sheet by default
- **WHEN** the sheet exists and `IfSheetExists` is left at its zero value
- **THEN** the call fails with `ErrSheetExists` and the file's bytes are unchanged

#### Scenario: Replacing a sheet
- **WHEN** the sheet exists and `IfSheetExists` is `SheetExistsReplace`
- **THEN** the sheet holds only the new table, the other sheets are unchanged, and the sheet order is unchanged

### Requirement: The CLI saves Excel with the same rule

`save <var> <file.xlsx>` SHALL accept `sheet <name>` and `if-exists fail|replace` with the defaults above, and SHALL refuse a `.xls` path and those two options on other formats.

#### Scenario: Saving twice
- **WHEN** `save t out.xlsx sheet s` runs twice
- **THEN** the second fails and says to add `if-exists replace`, and with it succeeds

#### Scenario: Saving twice without naming a sheet
- **WHEN** `save t out.xlsx` runs twice
- **THEN** the second fails without renaming the sheet to `Sheet2`, and says to add `sheet <name>` for a new sheet or `if-exists replace` to overwrite `Sheet1`

