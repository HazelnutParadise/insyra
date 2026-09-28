# csvxl-excel-append Specification

## Purpose
Defines what `csvxl.AppendCsvToExcel` does when the target sheet already exists (rebuild it in its place, so nothing stored in the old sheet survives while what the workbook records about it, its position, hidden state and the active sheet, stays, and names and formulas elsewhere keep their sheets, even for a single-sheet workbook) and requires every workbook the package opens to be closed.

## Requirements

### Requirement: Every opened workbook is closed

`AppendCsvToExcel`、`ExcelToCsv`、`EachExcelToCsv` SHALL 在函式（或每個檔案的處理）結束時關閉以 `excelize.OpenFile` 開啟的工作簿，包含錯誤路徑。

#### Scenario: Handles are released on the error path

- **WHEN** `ExcelToCsv` 因輸出目錄無法建立而回錯
- **THEN** 已開啟的工作簿仍被關閉（每個 `excelize.OpenFile` 緊接 `defer f.Close()`，以程式碼審查驗證）

### Requirement: Appending to an existing sheet name rebuilds the sheet

`csvxl.AppendCsvToExcel` 遇到工作簿已有同名工作表時 SHALL 以新的空白工作表取代它再寫入 CSV，使結果只含 CSV 的內容：舊工作表的儲存格、公式、隱藏列、列高、註解、超連結、欄寬、檢視、合併範圍與只屬於該工作表的定義名稱 SHALL NOT 殘留。活頁簿對該工作表的紀錄 SHALL 保留：它在工作簿中的位置與隱藏狀態不變，作用中的工作表不變。活頁簿其他地方的定義名稱與公式 SHALL 保留，並維持原本的所屬工作表。工作表名稱比對不分大小寫，取代後的工作表 SHALL 使用呼叫時給的名稱。工作簿只有那一張工作表時 SHALL 仍能完成替換。

#### Scenario: Stale cells do not survive an append

- **WHEN** 工作表 `data` 原有 3 列，之後以 1 列的 CSV `AppendCsvToExcel` 到同名工作表
- **THEN** 重新開啟後 `data` 只有 1 列

#### Scenario: Replacing the only sheet

- **WHEN** 工作簿只有 `data` 一張工作表，對 `data` 執行 `AppendCsvToExcel`
- **THEN** 呼叫成功，工作簿仍只有 `data` 一張工作表且內容為新 CSV

#### Scenario: The sheet keeps its position and nothing else

- **WHEN** 工作表依序為 `First`、`Target`、`Last`，`Target` 的第 2、3 列被隱藏，B2 有註解、B3 有超連結，之後對 `Target` 執行 `AppendCsvToExcel`
- **THEN** 工作表順序仍是 `First`、`Target`、`Last`，新寫入的列都是顯示狀態，B2 沒有註解，B3 沒有超連結，`ExcelToCsv` 讀回的列數與 CSV 相同

#### Scenario: Names of the sheets after it stay with them

- **WHEN** 工作表依序為 `First`、`Target`、`Last`，`Last` 有只屬於自己的定義名稱 `Rate`（指向 `Last!$B$2`，值為 7），`Last!C1` 的公式是 `Rate*10`，之後對 `Target` 執行 `AppendCsvToExcel`
- **THEN** `Rate` 仍屬於 `Last`，`Last!C1` 算出 70

#### Scenario: A hidden sheet stays hidden

- **WHEN** 工作表 `Data` 被隱藏、`Secret` 被設為 veryHidden，之後對兩者執行 `AppendCsvToExcel`
- **THEN** `Data` 仍是隱藏，`Secret` 仍是 veryHidden，作用中的工作表不變

### Requirement: A CSV that cannot be read leaves its sheet untouched

`csvxl.AppendCsvToExcel` SHALL read each CSV in full before it creates or replaces that CSV's sheet. When a CSV cannot be opened, its encoding cannot be detected, or its content is not valid CSV, the sheet it targets SHALL keep its content and settings, or SHALL NOT be created when the workbook had no such sheet, the other CSVs of the call SHALL still be appended, and the call SHALL return the `<n> files failed to append` error it returns today.

#### Scenario: A missing CSV does not empty the sheet

- **WHEN** 工作表 `data` 原有 3 列，對 `data` 執行 `AppendCsvToExcel`，指定的 CSV 不存在
- **THEN** 呼叫回傳 `1 files failed to append`，重新開啟後 `data` 仍是原本的 3 列

#### Scenario: Invalid CSV does not empty the sheet

- **WHEN** 工作表 `data` 原有 3 列，對 `data` 執行 `AppendCsvToExcel`，CSV 內容有未閉合的引號
- **THEN** 呼叫回傳 `1 files failed to append`，重新開啟後 `data` 仍是原本的 3 列

#### Scenario: One bad CSV in a batch

- **WHEN** 工作表 `good` 與 `bad` 各有內容，一次 `AppendCsvToExcel` 兩個 CSV，給 `good` 的可讀、給 `bad` 的不存在
- **THEN** 呼叫回傳 `1 files failed to append`，`good` 只有新 CSV 的內容，`bad` 保留原本的內容

#### Scenario: No empty sheet for a CSV that cannot be read

- **WHEN** 工作簿沒有 `extra` 工作表，對 `extra` 執行 `AppendCsvToExcel`，指定的 CSV 不存在
- **THEN** 呼叫回傳 `1 files failed to append`，重新開啟後工作表清單不變
