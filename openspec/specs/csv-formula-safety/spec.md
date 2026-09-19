# csv-formula-safety Specification

## Purpose
`DataTable` 的 CSV 匯出可選擇讓 Excel、LibreOffice 等試算表軟體安全開啟：開啟 `SanitizeFormulas` 選項時，為可能被當成公式執行的儲存格加上單引號前綴；未開啟時輸出與 `ToCSV` 完全相同。

## Requirements
### Requirement: A CSV export can be made spreadsheet-safe

`ToCSVWithOptions` 的 `SanitizeFormulas` 為 true 時，開頭為 `=`、`+`、`-`、`@`（含其前的空白）的儲存格 SHALL 加上單引號前綴；其他儲存格 SHALL 原樣寫出。預設 SHALL 為 false，`ToCSV` 的輸出 SHALL 不變。

#### Scenario: Formula cell with sanitisation on
- **WHEN** 儲存格為 `=cmd|' /C calc'!A0` 且 `SanitizeFormulas` 為 true
- **THEN** 檔案中該欄位不以 `=` 開頭

#### Scenario: Default is unchanged
- **WHEN** 以 `ToCSV` 寫出同一張表
- **THEN** 檔案中仍含原始的 `=cmd`
