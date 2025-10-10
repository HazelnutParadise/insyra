
檔案: `GPArotation_vgQ_bentler.go`

對應 R 檔案: `GPArotation_vgQ.bentler.R`

## 摘要（高階差異）

- 目的：驗證 Bentler criterion 的 Go 實作與 R 端數學公式與數值行為一致性。
- 高階結論：Go 在邏輯上重現 R 的運算（L^2、M、D、diff 與最終 Gq, f 的計算），但在 det/inverse 的數值穩定性處理與錯誤回饋上需加強（避免 panic，使用可靠的 log-det 或 SVD 解法）。

## 逐段重點差異

1. 基礎運算

- Go 與 R 都計算 L^2 與 M = t(L2) %*% L2，並以 D = diag(diag(M)) 進行差分。

2. 逆矩陣與行列式

- Go 直接嘗試 inverse(M) 與 inverse(D)，在失敗時 panic。
- R 的實作通常受 try/warning 包裝，且上層可能以其他策略處理 singular 的情況。

## 相容性風險與建議

- 使用 SVD-based 解法計算 log(det(M))（避免直接用 det 若 det 接近 0 或負值）以提升數值穩健性。
- 在 inverse 失敗時返回 error，並考慮 tiny-regularization（例如 M + eps*I）作為選項。

## 測試建議

1. 與 R 的 `vgQ.bentler` 比較小型矩陣與大型矩陣的 f 與 Gq 值（包含隨機 seed）。
2. 在接近奇異的情況下測試 fallback（regularization 與 error 返回的處理流程）。

## 優先次序

1. 數值穩健性（高），2. panic->error（中），3. 測試（中）。

## 下一步

- 在 `tests/fa/fixtures/bentler` 新增由 R 產生的測資，並執行 log-det / SVD-based 與直接 det 解法的數值比較，以決定在 Go 中採用的 fallback。
