# GPArotation_vgQ_geomin.go — 差異報告

檔案: GPArotation_vgQ_geomin.go
對應 R 檔案: GPArotation_vgQ.geomin.R

## 摘要（高階差異）

- 目的：確認 Go 與 R 在 geomin 目標函數中 delta 的處理、log/division 邊界情形與數值穩健性的一致性。
- 結論：需保護 delta 範圍、避免在 L 接近零時產生未定義行為，並在極端情況下回傳可處理的 error 或啟用 fallback。

## 逐段差異（重點）

1. L2 與 delta 的處理

- 建議在 Go 中限制 delta 最小值（例如 1e-8）並在回傳中記錄使用的 delta。

1. 權度計算與中間運算順序

- 檢查 Go 的 (2/k)*(L/L2)*proMat 等中間運算與 R 向量化實作是否等價（row/col 方向敏感）。

1. 錯誤處理

- 在 NA/Inf 或極端條件下應返回 error 或採可預期 fallback（非 panic），並加入 diagnostics。

## 相容性風險與建議

- 優先保護 delta 範圍與加入 fallback（高）；建立 fixtures 與 R 做逐元素比對（高）。

## 測試建議（具體）

1. 比較 R 的 `vgQ.geomin` 在多個 delta 值下的 f/Gq 與 Go 的差異。
2. 構造含 0、極小值、NaN/Inf 的 L，驗證 Go 的 fallback 行為。

## 下一步

- 在 `tests/fa/fixtures/geomin` 建立由 R 產生的多組 delta 參數 fixtures，並新增 Go 的比對測試。
