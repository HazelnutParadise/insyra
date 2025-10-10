# GPArotation_GPForth.go — 差異報告（審閱提案）

注意：此檔為審閱提案，原始 `GPArotation_GPForth.差異報告.md` 保持不變。

檔案: GPArotation_GPForth.go
對應 R 檔案: GPArotation::GPForth

## 摘要

- 目的：檢查 Go GPForth 的迭代規則、梯度計算與收斂判準是否與 R 等效，並處理數值不穩定時的 fallback。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/gp_forth`
- 建議建立 6 組測試矩陣（包含病態與正常情況），並比較 R/Go 的 fit 指標與 rotations。
