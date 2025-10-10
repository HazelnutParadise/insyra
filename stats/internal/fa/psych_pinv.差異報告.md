# psych_pinv.go — 差異報告

檔案: psych_pinv.go
對應 R 檔案: psych_Pinv.R

## 摘要

- 目的：比較 Go `Pinv` 與 R `Pinv` 在 SVD-based pseudo-inverse 的容差選取、API 與錯誤處理差異。
- 建議：核心數學等價，建議暴露 `tol`、避免 panic、改為 `(*mat.Dense, error)` 回傳，並加入 fixtures 與測試。

## 逐段差異（重點）

1. tol 可配置性

   - 建議暴露 tol（預設使用 `sqrt(machine eps)`），並允許呼叫者傳入自定義值以對齊 R 的行為。

1. 重建順序與索引一致性

   - 檢查 Go 實作的矩陣乘法順序與索引是否與 R 等價，確保浮點差異最小。

1. 錯誤處理

   - 建議改為 `Pinv(X, opts...) (*mat.Dense, error)`，並在 idx 為空或完全奇異時返回可解釋的 error，而不是 panic 或 undefined behavior。

## 測試建議（具體）

- fixtures 路徑：`tests/fa/fixtures/pinv`

1. 建立 10 組由 R 產生（正常/接近奇異/奇異）的測資與對應 R 結果。
2. 比較逐元素相對誤差，並測試不同 tol 設定。

## 下一步

- 我可以產生 `tests/fa/fixtures/pinv` 的 PoC fixtures 並新增 minimal Go 測試 harness 來驗證 API 與數值行為。

## 修正記錄

- **2025-10-11**: 將函數簽名從 `Pinv(X *mat.Dense) *mat.Dense` 改為 `Pinv(X *mat.Dense, tol float64) (*mat.Dense, error)`，暴露 tol 參數。
- 當 SVD 分解失敗時返回 error，而不是返回 nil。
- 當沒有奇異值超過容差時返回 error，表示矩陣過於奇異。
- 更新調用者 `psych_smc.go` 以處理新的 API，返回 error 時將 SMC 設為 1.0。
