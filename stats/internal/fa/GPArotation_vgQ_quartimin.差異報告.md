# GPArotation_vgQ_quartimin.go — 差異報告

檔案: GPArotation_vgQ_quartimin.go
對應 R 檔案: GPArotation_vgQ.quartimin.R

## 摘要（高階差異）

- 目的：比較 Go 的 `vgQQuartimin` 與 R 的 `vgQ.quartimin` 在 f/Gq 的計算、常數因子與中間矩陣運算順序上的一致性。
- 結論：需驗證 Go 在計算 Gq 與 f 時使用的中間運算（rowSums/crossprod/diag 等）與 R 完全等價，並在數值邊界處使用 stable fallback。

## 逐段差異（重點）

1. 輸入與前置處理

- 確認 Go 是否對 L 做與 R 相同的 normalization/前處理（delta/normalize 參數的應用順序）。

1. 目標函數與 Gq 計算

- 檢查中間運算（crossprod、diag、rowSums）順序與常數因子是否一致，避免逐元素差異。

1. 數值穩健性

- 若遇 NaN/Inf 或接近零的元素，建議在 Go 中加入 delta 或 SVD-based fallback，並回傳 diagnostics。

## 優先次序（建議）

1. 與 R 的逐元素比對（高）
2. 補強數值穩健性（高）

## 測試建議（具體）

1. 建立數組包含不同 p/k 與 edge-cases（zero rows、NaN/Inf），比較 R 與 Go 的 f/Gq。

## 下一步

- 在 `tests/fa/fixtures/quartimin` 放入由 R 產生的範例資料並新增比對測試。
