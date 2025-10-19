# GPArotation_vgQ_quartimin.go — 差異報告

檔案: GPArotation_vgQ_quartimin.go
對應 R 檔案: GPArotation_vgQ.quartimin.R

## 摘要

- 目的：比對 quartimin 的中間運算（crossprod、diag、rowSums）與 scaling 是否與 R 一致。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/quartimin`
- 建議建立多組 p/k 與 edge-cases 進行逐元素比對。
