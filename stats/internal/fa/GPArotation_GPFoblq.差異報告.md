# GPArotation_GPFoblq.go — 差異報告

檔案: GPArotation_GPFoblq.go
對應 R 檔案: GPArotation::GPFoblq

## 摘要（高階差異）

- 目的：核對 Go 在 GPFoblq（oblique GPF）算法細節、收斂條件與數值 fallback 的實作是否與 R 等價。
- 建議：避免 panic，將錯誤變為可捕捉的 error，並在 diagnostics 提供 iterations/converged/penalty。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/gp_foblq`
- 建議建立 6 組測試（含邊界條件與病態矩陣），由 R 產生金標並比較 Go 的輸出與 diagnostics。

## 下一步

- 我可以產生 PoC fixtures 並新增 minimal Go 比對測試。

## 修正記錄

- **2025-10-11**: 將函數簽名從 `GPFoblq(...) map[string]any` 改為 `GPFoblq(...) (map[string]any, error)`，避免 panic，返回 error 供上層處理。
- 補齊 diagnostics 欄位：在返回結果中添加 `"iterations"` 和 `"penalty"` 欄位。
- 更新所有調用者（psych_faRotations.go 中的 Quartimin、Oblimin、Simplimax、GeominQ、BentlerQ 和 Promax 函數）以處理 error，返回 identity rotation 作為 fallback。
- 修改 `obliqueCriterion` 函數以返回 error，並在內部處理 `vgQOblimin` 的 error。
