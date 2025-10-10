# 檔案: `psych_pinv.go` — 差異報告

對應 R 檔案: `psych_Pinv.R`

## 摘要（高階差異）

- 目的：比較 Go 版 `Pinv`（基於 SVD 的偽逆）與 R `Pinv` 在容差選取、API 設計與錯誤處理上的差異，並提出可驗證的測試與修正建議。
- 高階結論：核心數學等價；建議將容差（tol）暴露為可配置參數、避免 panic 而改回傳 error，並在重建邏輯中加入邊界檢查以貼近 R 的行為。

## 逐段重點差異

1. 容差（tol）定義與可配置性

   - R：預設使用 `sqrt(.Machine$double.eps)`，且通常允許使用者指定 `tol` 參數。
   - Go：目前使用硬編碼常數近似（來源於 IEEE double eps 的平方根），且未將 `tol` 暴露給呼叫者。

   影響：在不同平台或面對尺度極端的矩陣時，固定常數會造成與 R 不同的奇異值過濾結果。

2. 奇異值子集合與重建順序

   - 行為：兩邊皆選取 d_i > tol * d_1 作為保留集合，並使用子集合的 U、D、V 來重建偽逆。
   - 差異檢核：請確認 Go 實作的矩陣乘法次序與索引順序與 R 一致（微小差異會導致浮點誤差，但不影響整體正確性）。

3. API 與錯誤處理

   - R：單純回傳矩陣，並在極端情況以 warning/stop 處理（可由使用者捕捉）。
   - Go：目前實作回傳 `*mat.Dense`，遇到極端情況可能會 panic 或沒有足夠的錯誤訊息。

   建議：改為 `func Pinv(X *mat.Dense, tol float64) (*mat.Dense, error)` 或提供選項列（functional options）以支援更豐富的錯誤處理策略。

## 相容性風險與優先度建議

- 高優先：暴露 tol、避免 panic 並以 error 返回，以便上層採用 fallback（如正則化）。
- 中優先：在重建時處理 idx 為空的情況（回傳 error 或用最小正則化重建）。

## 測試建議（最小集合）

1. 與 R 的基準比對：準備 10 組由 R `Pinv` 產生的測資（包含正常、接近奇異與完全奇異），比較逐元素相對誤差。
2. tol 探索：測試多個 tol（R 的預設值、以及 ±1 個量級）並驗證 Go 在暴露 tol 後能複現 R 的行為。
3. 邊界情況：idx 為空、單一非零奇異值、高 condition number，檢查是否返回可理解的 error 或合理重建。

## 具體程式範例（API 建議）

```go
type PinvOption func(*pinvOpts)

func WithTol(tol float64) PinvOption { ... }

func Pinv(X *mat.Dense, opts ...PinvOption) (*mat.Dense, error) { ... }
```
