檔案: GPArotation_GPForth.go
對應 R 檔案: GPArotation_GPForth.R

摘要（高階差異）

- Go `GPForth` 為 R `GPArotation::GPForth` 的逐行翻譯，實作了 orthogonal GPA 旋轉演算法。
- 核心流程包括：正規化 (Kaiser)、計算 Gq 與 f (透過 vgQ.* 函式)、迭代 loop（計算 M, S, Gp）、內部 line-search 以 SVD 正交化 Tmatt，最後更新 Tmat、f、Gq。

逐行重要差異摘錄

- Go 對 SVD 失敗使用 `panic("SVD failed")`；R 在嘗試 do.call 時常以 try/error 處理並使用 warning/fallback。此差異會改變在病態矩陣上的健壯性。
- 回傳欄位與 R 類似：`loadings, Th, Table, method, orthogonal, convergence, Gq, f`。

相容性風險與建議

- 建議將 panic 換成返回 error 或至少捕捉轉為 warning，並在上層決定 fallback 行為。
- 建議增加單元測試：以 R 的 `GPArotation::GPForth` 為基準比較收斂結果、Phi/Th 與 f 值。
