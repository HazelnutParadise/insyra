# Proposal

## Why

`AppendCsvToExcel` replaces an existing sheet by clearing its cells in place, as v0.3.3 shipped. Clearing removes values and formulas but leaves everything else a row or cell carries. Measured on `origin/dev` (da6bb263) on 2026-09-28: after hiding rows 2 and 3 of a sheet, adding a comment on B2 and a hyperlink on B3, and appending a five-row CSV to it, rows 2 and 3 were still hidden, the comment and the link were still there, and `ExcelToCsv` read back three of the five rows (`h`, `r3`, `r4`), because the rows landing on the old hidden rows were skipped. The in-place clear also costs in proportion to what the old sheet held: replacing a sheet of 20,000 formulas took 106 ms (best of 3). The `0.4` line fixed this in 7e9b431e by rebuilding the sheet; the owner's direction is to use the same fix here.

## What Changes

- `AppendCsvToExcel` replaces an existing sheet the way `0.4` does: it deletes the sheet, creates it again, moves it back to its position and restores the active sheet. Nothing of the old sheet survives, and the cost no longer grows with what the old sheet held.
- Two corrections go beyond `0.4`'s routine, found by this port's review and measured: excelize's `DeleteSheet` moves every name defined for a later sheet onto the sheet before it and `MoveSheet` does not move it back, so a formula on an untouched sheet using its own name turned into `#NAME?`; and a hidden sheet came back visible. The rebuilt sheet puts those names back on their sheets and keeps its hidden state, like its position.
- **Behaviour change against v0.3.3**: the replaced sheet no longer keeps the old sheet's column widths, views or merged ranges, which the v0.3.3 changelog said were kept, nor the names defined for that sheet alone; formulas on other sheets and workbook-wide names that refer to it are kept (measured). A sheet found under a name that differs only in case takes the name as given, where v0.3.3 kept the old name. Data that reads back correctly is put ahead of keeping the old sheet's formatting.

## Capabilities

### New Capabilities

### Modified Capabilities
- `csvxl-excel-append`: a replaced sheet keeps its position and the active sheet, and nothing else of the old sheet.

## Impact

- `csvxl/convert.go` (`replaceSheet` rebuilt, `paddedSheetCells` removed), `csvxl/append_replace_test.go`.
- `Docs/csvxl.md`, both changelogs.
