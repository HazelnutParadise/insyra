## Why

The core CSV writer guards formula-like text by default since `csv-formula-guard-by-default`, but `csvxl.ExcelToCsv` and `EachExcelToCsv` wrote every cell as it was. Text that is safe inside a workbook, a text cell holding `=HYPERLINK(...)`, becomes a formula again once it is a CSV that someone opens in a spreadsheet. The owner ruled on 2026-09-26 that csvxl guards by default too, with a setting to turn it off, and chose to put that setting in an options struct together with the sheet list, which occupied the only trailing slot.

## What Changes

- **BREAKING**: `ExcelToCsv(excelFile, outputDir, csvNames, opts ...ExcelToCsvOptions)` with `Sheets` (empty means every sheet) and `AllowFormulas`; the trailing sheet names move into `Sheets`. `EachExcelToCsv(dir, outputDir, opts ...ExcelToCsvOptions)`.
- Both guard by default through `internal/csv.GuardFormula`, which the core writer now shares, so the two cannot disagree.
- CLI `convert` xlsx->csv takes `allowformulas true|false` and rejects an argument it does not understand.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `csv-formula-guard`: csvxl's Excel-to-CSV conversion is guarded the same way.

## Impact

- `internal/csv/formula_guard.go` (new), `datatable_csv.go`, `csvxl/convert.go`, `csvxl/convertDir.go`, `cli/commands/convert.go`; tests in `csvxl` and the CLI.
- `Docs/csvxl.md`, `Docs/cli-dsl.md`, the CLI skill, both changelogs.
