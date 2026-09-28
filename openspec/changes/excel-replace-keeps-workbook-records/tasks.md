# Tasks

## 1. Keep what the workbook records about a replaced sheet

- [x] 1.1 Tests: `TestAppendCsvToExcelKeepsTheNamesOfTheSheetsAfterIt` and `TestAppendCsvToExcelKeepsAHiddenSheetHidden` in `csvxl/append_replace_test.go`, ported from `dev` 851f3114, and `TestToExcelReplaceKeepsOtherSheetsNamesAndHiddenState` in `datatable_excel_test.go`; all three fail before the change (`go test -run ... ./csvxl/ .`)
- [x] 1.2 `excelsheet.Replace` raises every sheet-scoped name at the rebuilt sheet's index or above by one after moving it back, and writes the old `state` back; 1.1 and `go test ./csvxl/ ./internal/excelsheet/` and the root `Excel` tests pass
- [x] 1.3 `Docs/csvxl.md`, `Docs/DataTable.md`, the `csvxl-excel-append` Purpose and both changelogs say what a replaced sheet keeps; the stale comment above `TestAppendCsvToExcelKeepsTheReplacedSheetsPosition` no longer says the sheet keeps its column width

## 2. Verification

- [x] 2.1 `go test ./...`, `golangci-lint run` and `openspec validate excel-replace-keeps-workbook-records --strict` pass
