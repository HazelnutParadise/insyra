# GPArotation_vgQ_varimax.go — 差異報告

檔案: GPArotation_vgQ_varimax.go
對應 R 檔案: GPArotation_vgQ.varimax.R

## 摘要（高階差異）

- 目的：比較 Go 的 `vgQVarimax` 與 R 的 `vgQ.varimax` 在目標函數 (f) 與權度 (Gq) 的計算、常數縮放與回傳 metadata 上的一致性。
- 結論：兩者應數值接近；需驗證常數因子、矩陣運算順序與數值穩健性（NaN/Inf、division by zero）。

## 逐段差異（重點）

1. f 與 Gq 的計算式

- 檢查 Go 的公式是否與 R 完全等價（注意常數因子與矩陣運算順序）。

1. 常數縮放與 optimizer 行為

- 確認 (2/k)、1/4 等縮放因子在 Go/R 兩邊一致，因為會影響收斂行為與最終值。

1. 數值穩健性

- 檢查在 NaN/Inf 或零行/列情形下的處理（應有 delta/regularize 或 SVD/pinv fallback），避免 panic。

1. 回傳格式與 diagnostics

- 建議回傳 method 字串、Gq/f、以及 diagnostics（scaling、converged、iterations）以便比對。

## 相容性風險與建議

- 優先以 R 為金標撰寫單元測試；若發現常數縮放差異，應修正以避免逐元素偏移。

## 測試建議（具體）

1. 產生 10 組由 R 建立的小型 L 矩陣（不同 p/k），比較 f 與 Gq 的逐元素差。
2. 建立邊界案例（零行/列、NaN/Inf、極端條件）驗證 fallback 行為。

## 優先次序（建議）

1. 與 R 的逐元素比較測試（高）
2. 數值穩健性補強（高）
3. 文件化與 diagnostics（中）

## 下一步

- 在 `tests/fa/fixtures/varimax` 放入由 R 產生的測試資料與期望 f/Gq，並新增 minimal Go 比對 harness。
