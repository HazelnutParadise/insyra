# Proposal

## Why

`excelsheet.Replace`, which `csvxl.AppendCsvToExcel` and `DataTable.ToExcel` with `SheetExistsReplace` use to rebuild a sheet, breaks two things the workbook records outside the sheet. Found while porting 7e9b431e to `dev` (PR #400) and measured there with this same routine: excelize's `DeleteSheet` lowers the `localSheetId` of every name defined for a later sheet and `MoveSheet` never raises it again, so with sheets `First`, `Target`, `Last`, a name `Rate` defined for `Last` moved to `Target` and `Last!C1 = Rate*10` became `#NAME?` after replacing `Target`, where the in-place clear it replaced still gave 70. And a hidden or veryHidden sheet came back visible. The first breaks the `excel-write` requirement that the other sheets are unchanged.

## What Changes

- After moving the rebuilt sheet back, `Replace` raises every sheet-scoped defined name at its index or above by one, so names defined for the sheets after it stay with them.
- `Replace` writes the old sheet's hidden state back, `hidden` and `veryHidden` alike, as it already keeps the sheet's position and the active sheet.
- `Docs/csvxl.md`, `Docs/DataTable.md` and both changelog entries say what a replaced sheet keeps, including that names defined for the replaced sheet alone go with it and that a sheet matched in another case takes the name as given.

## Capabilities

### New Capabilities

### Modified Capabilities
- `csvxl-excel-append`: a rebuilt sheet keeps its hidden state, and names and formulas elsewhere keep their sheets.
- `excel-write`: the same for `ToExcel` with `SheetExistsReplace`.

## Impact

- `internal/excelsheet/replace.go`, `csvxl/append_replace_test.go`, `datatable_excel_test.go`.
- `Docs/csvxl.md`, `Docs/DataTable.md`, both changelogs.
