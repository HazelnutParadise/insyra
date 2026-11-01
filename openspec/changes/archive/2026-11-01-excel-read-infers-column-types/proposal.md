## Why

`ReadExcelSheet` and `ReadExcel` load every cell as text, while CSV reading infers column types, so the two spreadsheet formats have opposite defaults. It also makes `COUNT`, which counts numbers only since #640, return 0 on every column read from Excel (#414).

## What Changes

- **BREAKING**: `ReadExcelSheet` and `ReadExcel` infer column types the way CSV reading does.
- Both take an optional trailing `rawStrings ...bool` to keep every cell as text. `isr.Excel_inOpts` gains `RawStrings`, and the CLI's `load` accepts `infer false` for Excel.

## Impact

- `read.go`, `isr`, `internal/dsl/commands/load.go`, `Docs/DataTable.md`, `Docs/cli-dsl.md`, both changelogs.
