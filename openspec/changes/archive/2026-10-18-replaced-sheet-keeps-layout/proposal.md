# Proposal: replaced-sheet-keeps-layout

## Why

[#401](https://github.com/HazelnutParadise/insyra/issues/401): since v0.3.4 a replaced sheet is deleted and rebuilt, which also drops column widths and the view (frozen panes, zoom). Those do not hide data, so losing them only costs the user their formatting.

## What Changes

- `excelsheet.Replace` (used by `AppendCsvToExcel` and `ToExcel` with `SheetExistsReplace`) carries over the widths of visible columns and the sheet's first view and panes.
- Still dropped, because they hide or misplace the new data: hidden rows and columns, merged ranges, row heights, comments, links, and names scoped to the sheet (a name's target means nothing over new data).

## Capabilities

### Modified Capabilities

- `csvxl-excel-append`: column widths and the view are kept.

## Impact

`internal/excelsheet/replace.go`, tests in `csvxl`, `Docs/csvxl.md`, both CHANGELOGs.
