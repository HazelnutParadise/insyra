# GPArotation_vgQ_quartimax.go — 差異報告

檔案: GPArotation_vgQ_quartimax.go
對應 R 檔案: GPArotation_vgQ.quartimax.R

## 摘要（高階差異）

- 目的：比較 Go 與 R 在 quartimax 類型 vgQ 目標函數的前置處理（L 的正規化、delta/normalize）與 Gq/f 計算的一致性。
- 結論：建議在前處理階段對 delta/normalize 做嚴格檢查，並在極端情況下回傳 error 或使用 regularization。

## 逐段差異（重點）

1. 輸入與前置處理

- 確認 Go 是否在計算前正規化 L，以及 delta/normalize 參數是否與 R 相同。

2. 目標函數 f 與 Gq 的表達

- 檢查 Go 的 row/col 加權與常數縮放是否與 R 相符，包含中間矩陣運算順序。

# GPArotation_vgQ_quartimax.go — 差異報告

檔案: GPArotation_vgQ_quartimax.go
對應 R 檔案: GPArotation_vgQ.quartimax.R

## 摘要（高階差異）

- 目的：比較 Go 與 R 在 quartimax 類型 vgQ 目標函數的前置處理（L 的正規化、delta/normalize）與 Gq/f 計算的一致性。
- 結論：建議在前處理階段對 delta/normalize 做嚴格檢查，並在極端情況下回傳 error 或使用 regularization。

## 逐段差異（重點）

1. 輸入與前置處理

- 確認 Go 是否在計算前正規化 L，以及 delta/normalize 參數是否與 R 相同。

1. 目標函數 f 與 Gq 的表達

- 檢查 Go 的 row/col 加權與常數縮放是否與 R 相符，包含中間矩陣運算順序。

1. 數值穩健性與錯誤處理

- 建議在 division/log/det 等操作前加上 delta 或 regularization，並在 NA/Inf 或極端數值時回傳 error 而非 panic。

## 優先次序（建議）

1. 保護 delta 範圍並加入 fallback（高）
1. 與 R 建立 fixtures 做比對（中）
1. 改善錯誤處理（中）

## 測試建議（具體）

1. 建立 10 組不同 p/k 的 L（含零行/列、接近 0、NA/Inf）並比較 R 與 Go 的 f/Gq。
1. 測試 delta/normalize 不同設定下的行為差異。

## 下一步

- 在 `tests/fa/fixtures/quartimax` 放入由 R 產生的 10 組 L 並新增 Go 的比對測試。
