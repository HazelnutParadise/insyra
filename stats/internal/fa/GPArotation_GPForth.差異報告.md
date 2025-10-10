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

## 優先次序（建議）

1. panic->error 與 fallback 策略（高），2. 數值穩健性（SVD/log-det 改寫）與測試（高），3. 文件化與 diagnostics（中）。

## 測試建議（具體）

1. 與 R 比對：選取 5 個代表性 loadings（p/k 不同），在 R 與 Go 執行 `GPForth`，比較收斂的 f、Gq、Phi/Th 與最終 loadings 的逐元素差。
2. 病態矩陣測試：構造接近奇異的矩陣（condition number 很高），驗證 Go 不會 panic 且能以 error 或 regularization 退回。

## 下一步

- 修改 `GPForth` 中 SVD 呼叫的錯誤處理：捕捉錯誤後返回 `error`，並在上層提供 fallback（例如嘗試加入 tiny regularization M+epsI 或使用 alternative rotator）。
- 建立 `tests/fa/fixtures/gpforth` 並放入 R 產生的期望輸出以供自動化比較（我可以幫忙生成 fixtures）。
