# GPArotation_vgQ_bentler.go — 差異報告

檔案: GPArotation_vgQ_bentler.go
對應 R 檔案: GPArotation_vgQ.bentler.R

## 摘要（高階差異）

- 目的：驗證 Bentler criterion 的 Go 實作在 log-det / inverse / det 的數值穩健性與 R 的等價性。
- 結論：建議使用 SVD 或 log-det 的數值方法替代直接 det/inverse；在 inverse 失敗時回傳 error 或使用 tiny-regularization（M+epsI）。

## 逐段差異（重點）

1. 基礎運算與公式等價性

- 檢查 Go 與 R 在計算 L^2、M、D、diff 與最終 Gq/f 的步驟是否等價，特別是 det/inverse 的處理順序與數值方法。

2. 逆矩陣與錯誤處理

- 建議以 SVD-based 計算 log(det(M))，並在 inverse 失敗時回傳 error 或考慮 tiny-regularization（M + eps*I）。

## 相容性風險與建議

- 優先避免在 det 接近 0 或條件數太高時直接使用 det/inverse，改用更穩健的數值方法。

## 測試建議（具體）

1. 比較 R 與 Go 在多個樣本（含接近奇異情形）計算出的 f/Gq。
2. 測試在 inverse 失敗時 Go 的 fallback（regularization vs error）。

## 下一步

- 在 `tests/fa/fixtures/bentler` 放入由 R 產生的多組測試資料，並比較 Go 的 log-det / SVD-based 與直接 det/inverse 的差異。
