# GPArotation_GPForth.go — 差異報告

檔案: GPArotation_GPForth.go
對應 R 檔案: GPArotation_GPForth.R

## 摘要（高階差異）

- Go `GPForth` 為 R `GPArotation::GPForth` 的逐行翻譯，實作 orthogonal GPA 旋轉演算法。核心流程包含正規化 (Kaiser)、計算 Gq 與 f（透過 vgQ.* 函式）、迭代計算 M/S/Gp，以及使用 SVD 的內部 line-search 以維持正交性。

## 逐段差異（重點）

1. SVD 失敗的錯誤處理

- R：在某些狀況會以 warning + fallback 處理數值問題。
- Go：目前在 SVD 失敗時 `panic`，建議改為 `error` 返回或提供 fallback（tiny-regularization 或 alternative rotator）。

1. 回傳欄位與診斷

- 回傳結構與 R 類似（loadings, Th, Table, method, orthogonal, convergence, Gq, f），但建議補充 diagnostics 欄位（例如 `converged`, `iterations`, `residualNorm`）。

## 相容性風險與建議

- 優先：將 panic 改為 error 並設計 fallback 策略（高）。
- 數值穩健性相關改寫與測試（高）。

## 測試建議（具體）

1. 與 R 比對：挑選 5 組代表性 loadings（不同 p/k），比較 f、Gq、Phi/Th 與最終 loadings 的逐元素差。
1. 病態矩陣測試：模擬高 condition number，驗證 Go 在數值失敗時不會 panic，而是回傳 error 或以 regularization 退回。

## 下一步

- 在 `GPForth` 的 SVD/solve 呼叫周圍加入錯誤處理機制：捕捉錯誤並回傳 `error`，同時在上層提供嘗試 tiny-regularization（M+epsI）或 alternative rotator 的 fallback。
- 建立 `tests/fa/fixtures/gpforth` 並放入 R 產生的期望輸出，用於自動化比較。
