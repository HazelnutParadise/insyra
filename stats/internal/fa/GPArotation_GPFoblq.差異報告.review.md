# GPArotation_GPFoblq.go — 差異報告（審閱提案）

注意：此檔為審閱提案，原始 `GPArotation_GPFoblq.差異報告.md` 保持不變。

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
