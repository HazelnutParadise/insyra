# Tasks: csvxl-go-names

## 1. Tests

- [x] 1.1 `csvxl/names_test.go`：七個新名稱的 doc comment 以自己的名稱開頭（`TestExportedCSVXLNamesHaveGoDocComments`）；六個舊名稱的 `Deprecated:` 段落指名新名稱並寫明 `Removed in 0.4.`（`TestDeprecatedCSVXLNamesSayWhatReplacedThem`）
- [x] 1.2 `csvxl/new_names_behaviour_test.go`：`ExcelToCSV` 的工作表選項與公式防護（`TestExcelToCSVOptionsAndFormulaGuard`）

## 2. Implementation

- [x] 2.1 `csvxl/convert.go`、`csvxl/convertDir.go`、`csvxl/read_csv.go`：新名稱、舊名稱的 Deprecated 包裝、`ExcelToCSV` 與 `ExcelDirToCSV`（`ExcelToCSVOptions{Sheets, AllowFormulas}` 與公式防護）
- [x] 2.2 `cli/commands/convert.go`：CSV 轉 Excel 改呼叫 `CSVToExcel`；Excel 轉 CSV 仍呼叫 `ExcelToCsv`，輸出不變

## 3. Docs and changelog

- [x] 3.1 `Docs/csvxl.md`：新名稱與舊名稱的對照，以及 `ExcelToCSV` 的檔名與公式防護
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`csvxl` 段落
