## MODIFIED Requirements

### Requirement: Appending to an existing sheet name rebuilds the sheet

`csvxl.AppendCsvToExcel` 遇到工作簿已有同名工作表時 SHALL 以新的空白工作表取代它再寫入 CSV：舊工作表的儲存格、公式、隱藏列與隱藏欄、列高、註解、超連結、合併範圍與只屬於該工作表的定義名稱 SHALL NOT 殘留，可見欄的欄寬與檢視（縮放、格線、凍結或分割窗格）SHALL 保留。活頁簿對該工作表的紀錄 SHALL 保留：位置、隱藏狀態與作用中的工作表不變。活頁簿其他地方的定義名稱與公式 SHALL 保留，並維持原本的所屬工作表。工作表名稱比對不分大小寫，取代後的工作表 SHALL 使用呼叫時給的名稱。工作簿只有那一張工作表時 SHALL 仍能完成替換。

#### Scenario: Layout stays, what hides data goes

- **WHEN** `Target` 的 A 欄寬為 40、B 欄被隱藏、A2:C2 合併、第 1 列凍結、縮放 150，之後對 `Target` 執行 `AppendCsvToExcel`
- **THEN** A 欄寬仍為 40，凍結與縮放不變，B 欄顯示，沒有合併範圍，且每一列都讀得回來
