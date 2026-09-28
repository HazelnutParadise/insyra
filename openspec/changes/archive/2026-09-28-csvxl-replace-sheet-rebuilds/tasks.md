# Tasks

## 1. Rebuild a replaced sheet

- [x] 1.1 Tests in `csvxl/append_replace_test.go`, ported from `0.4` 7e9b431e: a replaced sheet keeps its position and the active sheet, and carries no hidden row, row height, comment, hyperlink or column width of the old sheet, and every row reads back through `ExcelToCsv`; the only-sheet test pins the default column width. They fail against the in-place clear (`go test -run TestAppendCsvToExcel ./csvxl/`)
- [x] 1.2 `replaceSheet` in `csvxl/convert.go` deletes, re-creates and moves back an existing sheet and restores the active sheet, with the body of `0.4`'s `excelsheet.Replace`; `paddedSheetCells` is removed; the tests in 1.1 and `go test ./csvxl/` pass
- [x] 1.3 `TestAppendCsvToExcelKeepsTheNamesOfTheSheetsAfterIt`: with sheets First, Target, Last, Tail, the names defined for Last and Tail stay with them, a workbook name stays, the name defined for Target goes, and `Last!C1 = Rate*10` still calculates 70 after appending to Target; it fails against `0.4`'s routine as ported. `replaceSheet` raises every sheet-scoped name at the rebuilt sheet's index or above by one after moving the sheet back (`go test ./csvxl/`)
- [x] 1.4 `TestAppendCsvToExcelKeepsAHiddenSheetHidden`: a hidden sheet stays hidden and a veryHidden one stays veryHidden, and the active sheet does not change; it fails against `0.4`'s routine as ported. `replaceSheet` writes the old `state` back (`go test ./csvxl/`)
- [x] 1.5 `Docs/csvxl.md`, the `csvxl-excel-append` spec's Purpose and both changelogs say what a replaced sheet keeps and what it no longer keeps against v0.3.3, and why

## 2. Verification

- [x] 2.1 The probe measured on `origin/dev` before the change, rerun on the branch: no hidden row, comment or link survives, `ExcelToCsv` reads back all five rows, and the time to replace a sheet of 20,000 formulas is recorded next to the 106 ms before: 2.9 ms after, best of 3, measured on 2026-09-28
- [x] 2.2 `go test ./...`, `golangci-lint run` and `openspec validate csvxl-replace-sheet-rebuilds --strict` pass
