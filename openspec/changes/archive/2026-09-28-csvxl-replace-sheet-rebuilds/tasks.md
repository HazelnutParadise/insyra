# Tasks

## 1. One way to replace a sheet

- [x] 1.1 Tests in `csvxl/append_replace_test.go`: a replaced sheet keeps its position and the active sheet, and carries no hidden row, comment or hyperlink of the old sheet; they fail against the in-place clear (`go test ./csvxl/`)
- [x] 1.2 Move `replaceSheetInPlace` from `datatable_excel.go` into `internal/excelsheet` and call it from `DataTable.ToExcel` and `csvxl.replaceSheet`; remove `paddedSheetCells`; the tests in 1.1 and the existing `ToExcel` and csvxl tests pass (`go test . ./csvxl/`)
- [x] 1.3 `Docs/csvxl.md`, the `csvxl-excel-append` spec's Purpose and both changelogs say what a replaced sheet keeps

## 2. Verification

- [x] 2.1 `go test ./...`, `golangci-lint run` and `openspec validate csvxl-replace-sheet-rebuilds --strict` pass
