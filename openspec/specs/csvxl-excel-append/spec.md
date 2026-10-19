# csvxl-excel-append Specification

## Purpose
Defines what `csvxl.AppendCSVToExcel` does when the target sheet already exists (rebuild it in its place, so no cell, hidden row or column, merged range, comment or link survives while column widths and the view carry over, and what the workbook records about it, its position, hidden state and the active sheet, stays, and names and formulas elsewhere keep their sheets, even for a single-sheet workbook) and requires every workbook the package opens to be closed.

## Requirements

### Requirement: Appending to an existing sheet name rebuilds the sheet

`csvxl.AppendCSVToExcel` 遇到工作簿已有同名工作表時 SHALL 以新的空白工作表取代它再寫入 CSV：舊工作表的儲存格、公式、隱藏列與隱藏欄、列高、註解、超連結、合併範圍與只屬於該工作表的定義名稱 SHALL NOT 殘留，可見欄的欄寬與檢視（縮放、格線、凍結或分割窗格）SHALL 保留。活頁簿對該工作表的紀錄 SHALL 保留：位置、隱藏狀態與作用中的工作表不變。活頁簿其他地方的定義名稱與公式 SHALL 保留，並維持原本的所屬工作表。工作表名稱比對不分大小寫，取代後的工作表 SHALL 使用呼叫時給的名稱。工作簿只有那一張工作表時 SHALL 仍能完成替換。

#### Scenario: Stale cells do not survive an append

- **WHEN** 工作表 `data` 原有 3 列，之後以 1 列的 CSV `AppendCSVToExcel` 到同名工作表
- **THEN** 重新開啟後 `data` 只有 1 列

#### Scenario: Replacing the only sheet

- **WHEN** 工作簿只有 `data` 一張工作表，對 `data` 執行 `AppendCSVToExcel`
- **THEN** 呼叫成功，工作簿仍只有 `data` 一張工作表且內容為新 CSV

#### Scenario: The sheet keeps its position and nothing else

- **WHEN** 工作表依序為 `First`、`Target`、`Last`，`Target` 的第 2、3 列被隱藏，B2 有註解、B3 有超連結，之後對 `Target` 執行 `AppendCSVToExcel`
- **THEN** 工作表順序仍是 `First`、`Target`、`Last`，新寫入的列都是顯示狀態，B2 沒有註解，B3 沒有超連結，`ExcelToCSV` 讀回的列數與 CSV 相同

#### Scenario: Layout stays, what hides data goes

- **WHEN** `Target` 的 A 欄寬為 40、B 欄被隱藏、A2:C2 合併、第 1 列凍結、縮放 150，之後對 `Target` 執行 `AppendCsvToExcel`
- **THEN** A 欄寬仍為 40，凍結與縮放不變，B 欄顯示，沒有合併範圍，且每一列都讀得回來

#### Scenario: Names of the sheets after it stay with them

- **WHEN** 工作表依序為 `First`、`Target`、`Last`，`Last` 有只屬於自己的定義名稱 `Rate`（指向 `Last!$B$2`，值為 7），`Last!C1` 的公式是 `Rate*10`，之後對 `Target` 執行 `AppendCSVToExcel`
- **THEN** `Rate` 仍屬於 `Last`，`Last!C1` 算出 70

#### Scenario: A hidden sheet stays hidden

- **WHEN** 工作表 `Data` 被隱藏、`Secret` 被設為 veryHidden，之後對兩者執行 `AppendCSVToExcel`
- **THEN** `Data` 仍是隱藏，`Secret` 仍是 veryHidden，作用中的工作表不變

### Requirement: Every opened workbook is closed

`AppendCSVToExcel`、`ExcelToCSV`、`ExcelDirToCSV` SHALL 在函式（或每個檔案的處理）結束時關閉以 `excelize.OpenFile` 開啟的工作簿，包含錯誤路徑。

#### Scenario: Handles are released on the error path

- **WHEN** `ExcelToCSV` 因輸出目錄無法建立而回錯
- **THEN** 已開啟的工作簿仍被關閉（每個 `excelize.OpenFile` 緊接 `defer f.Close()`，以程式碼審查驗證）
