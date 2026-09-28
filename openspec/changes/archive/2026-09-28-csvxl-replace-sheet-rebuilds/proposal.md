# Proposal

## Why

`AppendCsvToExcel` replaces an existing sheet by clearing its cells in place, which v0.3.3 shipped and this line took from dev. Clearing removes values and formulas but leaves everything else a cell or row carries. Measured on 2026-09-28: a replaced sheet whose old rows were hidden keeps them hidden, so a new data row lands on a hidden row, is invisible in Excel, and is skipped when `ExcelToCsv` reads the sheet back; an old comment stays on B2 and an old hyperlink stays on B3 over the new value. The in-place clear also costs in proportion to what the old sheet held: removing a formula walks excelize's calculation chain, so replacing a sheet of 20,000 formulas took 157 ms against 6 ms rebuilt (measured on this branch with the same probe), and a review probe found a 6.7 KB file whose rows omit cell addresses taking 11 s and 5.4 GB. The package's own spec says the result holds only the CSV's content.

## What Changes

- `AppendCsvToExcel` replaces an existing sheet the way `DataTable.ToExcel` does with `IfSheetExists: SheetExistsReplace`: it deletes the sheet, creates it again, moves it back to its position and restores the active sheet. Nothing of the old sheet survives, and the cost no longer grows with what the old sheet held.
- **Behaviour change against v0.3.3**: the replaced sheet no longer keeps the old sheet's column widths, views or merged ranges.
- The replacement routine moves from the root package into an internal package that both callers use, so there is one way to replace a sheet.

## Capabilities

### New Capabilities

### Modified Capabilities
- `csvxl-excel-append`: a replaced sheet keeps its position and nothing else of the old sheet.

## Impact

- `csvxl/convert.go` (`replaceSheet`, `paddedSheetCells` removed), `datatable_excel.go`, a new `internal/excelsheet` package.
- `Docs/csvxl.md`, both changelogs.
