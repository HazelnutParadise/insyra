# GPArotation_vgQ_varimax.go — 差異報告（審閱提案）

注意：此檔為審閱提案，原始 `GPArotation_vgQ_varimax.差異報告.md` 保持不變。

檔案: GPArotation_vgQ_varimax.go
對應 R 檔案: GPArotation_vgQ.varimax.R

## 摘要

- 目的：確認 varimax 目標函數、縮放因子與 diagnostics 在 Go/R 的一致性。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/varimax`
- 建議由 R 產生數組測資並比較 f/Gq 與 diagnostics。
