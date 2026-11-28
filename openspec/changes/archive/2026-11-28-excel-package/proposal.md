# Proposal: excel-package (0.3.x)

## Why

On `0.4` the Excel readers and writers leave the root package for a package of their own, `excel`, so a program that only uses the core no longer compiles the Excel library (10.17 MB to 6.68 MB for a program calling only `insyra.NewDataList(1, 2).Sum()`, linux/amd64). The owner's rule is that a name `0.4` replaces ships on `0.3.x` too, with the old one deprecated, so `0.3.6` users can move before `0.4` removes it. The design, confirmed by the owner on 2026-11-28, is in `/mnt/project-files/453/split-design.md`.

Milestone: #453 root-package split, slice 1 of 3, the `dev` half. Waits for nothing; the `0.4` half is its own change on `0.4`.

## What Changes

- New package `excel` with the `0.4` API: `Read`, `ReadFrom`, `SheetNames`, `Write`, `WriteTo`, `ReadOptions`, `WriteOptions`, and `SheetExistsPolicy`, `SheetExistsFail`, `SheetExistsReplace`, `ErrSheetExists` as the same type and values as the core's.
- `excel` is implemented on the core's functions, so it reads and writes exactly as they do; it reads the first sheet when none is named, types columns unless `RawStrings` is set, and writes the header row unless `NoHeaderRow` is set.
- Deprecated, behaviour unchanged: `ReadExcelSheet`, `ReadExcel`, `DataTable.ToExcel`, `DataTable.WriteExcel` (and on `IDataTable`), `ExcelWriteOptions`, `SheetExistsPolicy` and its constants, `ErrSheetExists`, `csvxl.SheetNames`.
- `isr` and the CLI write through `excel`; their reads keep the deprecated reader so their 0.3.x default (text with a warning) does not change.

## Capabilities

### New Capabilities

- `excel-package`: where Excel reading and writing live and what they take.

## Impact

- Code: `excel/`, `datatable_excel.go`, `read.go`, `interfaces.go`, `csvxl/convert.go`, `isr/`, `cli/commands/`, `allpkgs/`.
- Docs: `Docs/excel.md` (new), `Docs/DataTable.md`, `Docs/csvxl.md`, `Docs/README.md`, `README.md`, `README_TW.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
