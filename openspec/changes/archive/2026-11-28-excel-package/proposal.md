# Proposal: excel-package

## Why

Every program that imports the root `insyra` package compiles the Excel library into its binary, whether it reads a workbook or not: on linux/amd64 a program that only calls `insyra.NewDataList(1, 2).Sum()` is 10.17 MB, and 6.68 MB once the Excel readers and writers leave the root package. Go has no optional dependency inside one package, so the only way out is a package of its own, the way pandas, polars, R (`readxl`) and Julia (`XLSX.jl`) keep Excel apart from CSV. This is the first slice of #453 (moving functions that need not be `DataTable` methods out of the root directory). The design, reviewed independently and confirmed by the owner on 2026-11-28, is in `/mnt/project-files/453/split-design.md`; this change carries its `excel` part on `0.4`.

Milestone: #453 root-package split, slice 1 of 3 (`excel`, then `preprocess`, then `sqlio`). Waits for nothing.

## What Changes

- **BREAKING** New package `excel`: `Read(path, opts...)`, `ReadFrom(r, opts...)`, `SheetNames(path)`, `Write(dt, path, opts...)`, `WriteTo(dt, w, opts...)`, `ReadOptions`, `WriteOptions`, `SheetExistsPolicy`, `SheetExistsFail`, `SheetExistsReplace`, `ErrSheetExists`. An empty `ReadOptions.Sheet` reads the first sheet.
- **BREAKING** Removed from the root package: `ReadExcelSheet`, `ReadExcel`, `ExcelReadOptions`, `DataTable.ToExcel`, `DataTable.WriteExcel` (and from `IDataTable`), `ExcelWriteOptions`, `SheetExistsPolicy` and its constants, `ErrSheetExists`.
- **BREAKING** `csvxl.SheetNames` is removed; `excel.SheetNames` replaces it. The rest of `csvxl` is unchanged.
- The 512 MB unzip limit moves to `internal/excelsheet.Options`, shared by `excel` and `csvxl`; the CSV/Excel type-inference rule moves to `internal/csv.InferColumn`, shared by the core CSV readers and `excel`.
- `isr`, the CLI (`load`, `save`, `convert`) and `allpkgs` use `excel`. CLI commands are unchanged.

## Capabilities

### New Capabilities

- `excel-package`: where Excel reading and writing live and what they take.

### Modified Capabilities

- `excel-write`, `reader-writer-io`, `read-write-names`, `isr-short-syntax`, `dsl-commands`: the names they cite move to `excel`.
- `csvxl-sheet-pairing`: listing a workbook's sheets moves to `excel`.

## Impact

- Code: `excel/`, `internal/excelsheet/options.go`, `internal/csv/infer.go`, `read.go`, `interfaces.go`, `enum_names.go`, `csvxl/`, `isr/`, `internal/dsl/commands/`, `allpkgs/`; `datatable_excel.go` deleted.
- Tests: the root Excel tests move to `excel/`.
- Docs: `Docs/excel.md` (new), `Docs/DataTable.md`, `Docs/csvxl.md`, `Docs/README.md`, `README.md`, `README_TW.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- `dev` gets the same package with the old names marked Deprecated, in a change of its own.
