# GPArotation_vgQ_simplimax.go — 差異報告（審閱提案）

注意：此檔為審閱提案，原始 `GPArotation_vgQ_simplimax.差異報告.md` 保持不變。

檔案: GPArotation_vgQ_simplimax.go
對應 R 檔案: GPArotation_vgQ.simplimax.R

## 摘要

- 目的：檢視 simplimax 在稀疏性與 penalty 設計上的相容性。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/simplimax`
- 建議建立稀疏/非稀疏的 loadings 測試並比較稀疏化程度與 diagnostics。
