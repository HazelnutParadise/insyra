# GPArotation_vgQ_simplimax.go — 差異報告

檔案: GPArotation_vgQ_simplimax.go
對應 R 檔案: GPArotation_vgQ.simplimax.R

## 摘要（高階差異）

- 目的：比較 Go 的 `vgQSimplimax` 與 R 的 `vgQ.simplimax` 在稀疏性/複雜度懲罰（penalty/thresholding）與權度計算上的一致性。
- 結論：需驗證 Go 在 thresholding/penalty 的實作細節（top-n penalty, thresholding rules）與 R 一致，並在極端情形下使用穩健化處理。

## 逐段差異（重點）

1. thresholding / penalty 的實作

- 確認 Go 與 R 在 penalize top-n 或 threshold 的具體行為是否一致。

1. 連續性與穩健性

- 檢查在極端稀疏或高罰值情況下結果的穩健性，若存在非微分點建議用穩健化處理（小 epsilon）。

1. 測試建議

- 建立包含稀疏與非稀疏 loadings 的測試組合，比對 R 與 Go 的輸出格式與稀疏性程度。

## 優先次序（建議）

1. 核心行為驗證（高）
2. 穩健化/epsilon 處理（中）

## 下一步

- 在 `tests/fa/fixtures/simplimax` 建立由 R 產生的 fixtures 並新增比較測試。
