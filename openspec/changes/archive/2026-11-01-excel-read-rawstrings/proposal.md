## Why

From 0.4 `ReadExcelSheet` and `ReadExcel` type columns the way CSV reading does. A program should be able to choose before it upgrades, and to be told the default is changing (#414).

## What Changes

- Both readers take an optional `rawStrings ...bool`: `true` keeps text, `false` types columns as 0.4 will.
- Left out, every cell stays text as it does today, with one warning.
- `isr.Excel_inOpts` gains `RawStrings`, and the CLI's `load` accepts `infer` for Excel.

## Impact

- `read.go`, `isr`, `cli/commands/load.go`, `Docs/DataTable.md`, `Docs/cli-dsl.md`, both changelogs.
