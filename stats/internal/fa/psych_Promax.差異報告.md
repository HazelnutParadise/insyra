# psych_Promax.go — 差異報告

檔案: psych_Promax.go
對應 R 檔案: psych_Promax.R

## 摘要

- 目的：比較 Go 與 R 在 Promax（oblique）旋轉的差異，包含 power transform、參數對應、錯誤處理與回傳資訊。
- 結論（建議）：確認 `power` / `normalize` 的預設與 API，並避免 panic（改為返回 error），同時加入 fixtures 做比對。

## 逐段差異（重點）

1. 演算法步驟與參數對齊

- 建議確認 Go 是否暴露 `power` 與 `normalize` 參數，並在文件中標註預設值以便與 R 一致。

2. 數值穩健性

- 建議在 power transform 與逆矩陣計算時加入 fallback 與 diagnostics（converged / iterations / residualNorm）。

## 測試建議（具體）

- fixtures 路徑：`tests/fa/fixtures/promax`

1. 選取 5 組代表性 loadings，在 R 與 Go 上以不同 `power` 值（2,3,4）與 `normalize` true/false 做比較。

## 下一步

- 我可以建立 `tests/fa/fixtures/promax` 的 PoC fixtures 並新增 minimal Go test harness。
