## ADDED Requirements

### Requirement: Appending to an existing sheet name replaces the sheet

`csvxl.AppendCsvToExcel` 遇到工作簿已有同名工作表時 SHALL 先移除該工作表再建立新的，使結果只含 CSV 的內容，舊工作表超出 CSV 範圍的儲存格 SHALL NOT 殘留。工作簿只有那一張工作表時 SHALL 仍能完成替換。

#### Scenario: Stale cells do not survive an append

- **WHEN** 工作表 `data` 原有 3 列，之後以 1 列的 CSV `AppendCsvToExcel` 到同名工作表
- **THEN** 重新開啟後 `data` 只有 1 列

#### Scenario: Replacing the only sheet

- **WHEN** 工作簿只有 `data` 一張工作表，對 `data` 執行 `AppendCsvToExcel`
- **THEN** 呼叫成功，工作簿仍只有 `data` 一張工作表且內容為新 CSV

### Requirement: Every opened workbook is closed

`AppendCsvToExcel`、`ExcelToCsv`、`EachExcelToCsv` SHALL 在函式（或每個檔案的處理）結束時關閉以 `excelize.OpenFile` 開啟的工作簿，包含錯誤路徑。

#### Scenario: Handles are released on the error path

- **WHEN** `ExcelToCsv` 因輸出目錄無法建立而回錯
- **THEN** 已開啟的工作簿仍被關閉（每個 `excelize.OpenFile` 緊接 `defer f.Close()`，以程式碼審查驗證）
