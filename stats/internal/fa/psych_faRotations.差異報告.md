
檔案: `psych_faRotations.go`

對應 R 檔案: `psych_faRotations.R`

## 摘要（高階差異）

- 目的：此檔案為 Go 端的 fa rotation dispatcher，負責把使用者的 `method` 與 `RotOpts` 映射到具體的 rotation 演算法實作（如 varimax、geomin、promax、GPArotation 等）。
- 高階結論：功能上與 R 的 `psych::faRotations` 對應；但在錯誤處理、參數預設與回傳診斷資訊（converged/iterations/restarts）三方面存在差異，可能導致在錯誤情境或邊界案例中行為不一致。

## 逐段重點差異

1) Dispatch 與 method mapping

- Go: 使用 typed `RotOpts` 與內建 dispatch（switch / map）直接呼叫對應的 rotation 函式。實作偏向內建所有子演算法以移除外部依賴。
- R: `psych::faRotations` 在需要時可依賴 `GPArotation` 套件，並在缺失套件時採取 fallback 或發出 warning。

2) 參數預設與型別

- Go: `RotOpts` 為 struct，型別嚴謹但部分預設值可能以零值表示；需明確文檔標示哪些欄位為必填或會被覆寫成預設值。
- R: 參數預設直接在函式宣告中給定，且較常用 `NULL` / `missing` 行為與檢查函數。

3) 錯誤處理與 fallback

- Go: 許多子函式遇到數值錯誤（例如奇異矩陣、SVD 失敗）目前會 `panic`（或隱含崩潰風險），上層若不捕捉會終止整個流程。
- R: 傾向使用 `try` / `warning` / fallback（例如改用其他 rotation 方法或回傳警告）來保持流程不中斷。

4) 回傳內容與診斷資訊

- Go: 回傳以核心資料（loadings 矩陣、method 字串）為主，缺少 `converged`、`iterations`、`restarts`、`fit` 等詳細診斷欄位。
- R: 回傳物件中包含豐富診斷資訊，便於使用者判斷 rotation 結果的可靠性。

## 相容性風險與建議

- 高優先：將 panic 行為改為 error 回傳或讓上層可捕捉（match R 的非致命錯誤處理），以避免在單一路徑失敗時中斷整個分析流程。
- 中優先：統一 `RotOpts` 的預設值來源（集中初始化），並在文件中明確列出與 R 預設不同之處。
- 低優先：擴充回傳結構，加入 `converged`、`iterations`、`fit` 等欄位，便於後續分析與比對 R 的結果。

## 測試建議（最小集合）

1. 功能驗證：選取 5 組具有代表性的 loadings 矩陣（不同 p/k、含 sparse 與 dense），在 R 與 Go 上執行相同 method，比對 loadings（逐元素相對誤差）與 fit 指標。
2. 邊界測試：輸入包含 NaN/Inf、或接近奇異的矩陣，確認 Go 不會 panic，且會以 error 返回或提供可處理的 fallback。
3. 效能/隨機性：對比多次 restart 行為（若實作 restart 機制），確認隨機種子控制與結果再現性。
