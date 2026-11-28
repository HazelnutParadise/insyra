# Tasks: excel-package

## 1. Package

- [x] 1.1 `excel/read.go`: `Read`, `ReadFrom`, `SheetNames`, `ReadOptions`; an empty `Sheet` reads the first sheet; more than one options value is an error.
- [x] 1.2 `excel/write.go`: `Write`, `WriteTo`, `WriteOptions`, `SheetExistsPolicy`, `ErrSheetExists`, moved from `datatable_excel.go`; a nil table is an error.
- [x] 1.3 `excel/enum_names.go` and its test.
- [x] 1.4 `internal/excelsheet.Options` and `internal/csv.InferColumn`, used by the core, `excel` and `csvxl`.

## 2. Callers

- [x] 2.1 Remove the root names and the `IDataTable` methods; delete `datatable_excel.go`.
- [x] 2.2 `csvxl` uses `excelsheet.Options`; remove `csvxl.SheetNames`.
- [x] 2.3 `isr`, `internal/dsl/commands` (`load`, `save`, `convert`) and `allpkgs` use `excel`.

## 3. Tests

- [x] 3.1 Move the root Excel tests to `excel/` under the new names.
- [x] 3.2 New tests: the first sheet by default, a named sheet, `SheetNames`, two options values, a nil table, a write-then-read round trip.
- [x] 3.3 `go list -deps .` lists no `xuri/excelize` package; the probe program is 6.68 MB on linux/amd64.

## 4. Docs

- [x] 4.1 `Docs/excel.md`; pointers in `Docs/DataTable.md` and `Docs/csvxl.md`; rows in `README.md`, `README_TW.md`, `Docs/README.md`.
- [x] 4.2 `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`, BREAKING marked.

## 5. Verify

- [x] 5.1 `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate excel-package --strict`.
