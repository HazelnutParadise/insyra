# GPArotation_vgQ_target.go — 差異報告（審閱提案）

注意：此檔為審閱提案，原始 `GPArotation_vgQ_target.差異報告.md` 保持不變。

檔案: GPArotation_vgQ_target.go
對應 R 檔案: GPArotation_vgQ.target.R

## 摘要

- 目的：比較 target rotation 在 vgQ 系列的實作差異（mask 處理、penalty、convergence）。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/target`
- 建議建立 multiple target matrices（含 NA/mask）由 R 產生金標，並比對 Go 的 aligned loadings/diagnostics。
