# Spec Delta

## MODIFIED Requirements

### Requirement: An existing sheet is replaced only when asked

When the sheet already exists, `ToExcel` SHALL refuse with an error matching `ErrSheetExists` and leave the file unchanged, unless `IfSheetExists` is `SheetExistsReplace`, in which case it SHALL replace that sheet's contents in the sheet's original position. A replaced sheet SHALL stay hidden when it was hidden, and every defined name and formula elsewhere in the workbook SHALL keep the sheet it belonged to.

#### Scenario: Saving over a sheet by default
- **WHEN** the sheet exists and `IfSheetExists` is left at its zero value
- **THEN** the call fails with `ErrSheetExists` and the file's bytes are unchanged

#### Scenario: Replacing a sheet
- **WHEN** the sheet exists and `IfSheetExists` is `SheetExistsReplace`
- **THEN** the sheet holds only the new table, the other sheets are unchanged, and the sheet order is unchanged

#### Scenario: Replacing a hidden sheet before a sheet with its own names
- **WHEN** the workbook holds `First`, a hidden `Data` and `Last`, `Last` defines `Rate` for itself as `Last!$B$2` holding 7 with `Last!C1 = Rate*10`, and a table is saved over `Data` with `SheetExistsReplace`
- **THEN** `Data` is still hidden, `Rate` still belongs to `Last`, and `Last!C1` calculates 70
