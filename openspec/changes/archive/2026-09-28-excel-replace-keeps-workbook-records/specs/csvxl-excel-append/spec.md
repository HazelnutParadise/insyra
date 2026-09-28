# Spec Delta

## MODIFIED Requirements

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
