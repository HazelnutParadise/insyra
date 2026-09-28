# csvxl-excel-append Specification

## Purpose
Defines what `csvxl.AppendCsvToExcel` does when the target sheet already exists (rebuild it in its place, so nothing of the old sheet survives while it keeps its position, even for a single-sheet workbook) and requires every workbook the package opens to be closed.

## Requirements
### Requirement: Appending to an existing sheet name rebuilds the sheet

`csvxl.AppendCsvToExcel` 遇到工作簿已有同名工作表時 SHALL 以新的空白工作表取代它再寫入 CSV，使結果只含 CSV 的內容：舊工作表的儲存格、公式、隱藏列、列高、註解與超連結 SHALL NOT 殘留。該工作表 SHALL 保留它在工作簿中的位置，作用中的工作表 SHALL 不變。舊工作表的欄寬、檢視與合併範圍 SHALL NOT 保留。工作簿只有那一張工作表時 SHALL 仍能完成替換。

#### Scenario: Stale cells do not survive an append

- **WHEN** 工作表 `data` 原有 3 列，之後以 1 列的 CSV `AppendCsvToExcel` 到同名工作表
- **THEN** 重新開啟後 `data` 只有 1 列

#### Scenario: Replacing the only sheet

- **WHEN** 工作簿只有 `data` 一張工作表，對 `data` 執行 `AppendCsvToExcel`
- **THEN** 呼叫成功，工作簿仍只有 `data` 一張工作表且內容為新 CSV

#### Scenario: The sheet keeps its position and nothing else

- **WHEN** 工作表依序為 `First`、`Target`、`Last`，`Target` 的第 2、3 列被隱藏，B2 有註解、B3 有超連結，之後對 `Target` 執行 `AppendCsvToExcel`
- **THEN** 工作表順序仍是 `First`、`Target`、`Last`，新寫入的列都是顯示狀態，B2 沒有註解，B3 沒有超連結，`ExcelToCsv` 讀回的列數與 CSV 相同

### Requirement: Every opened workbook is closed

`AppendCsvToExcel`、`ExcelToCsv`、`EachExcelToCsv` SHALL 在函式（或每個檔案的處理）結束時關閉以 `excelize.OpenFile` 開啟的工作簿，包含錯誤路徑。

#### Scenario: Handles are released on the error path

- **WHEN** `ExcelToCsv` 因輸出目錄無法建立而回錯
- **THEN** 已開啟的工作簿仍被關閉（每個 `excelize.OpenFile` 緊接 `defer f.Close()`，以程式碼審查驗證）

