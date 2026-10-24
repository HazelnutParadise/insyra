# Tasks: diag-typed-functions

## 1. 四個具型別的函式

- [x] 1.1 測試：`DiagOf`（方陣、長方矩陣、nil 介面、nil `*mat.Dense`）、`DiagMatrix`、`DiagMatrixSize`（補 0、長方單位矩陣、`v` 過長、大小小於 1）、`IdentityMatrix`（含 0 與負數）、不修改輸入。
- [x] 1.2 `stats/diag.go` 實作四個函式。

## 2. Diag 標為 Deprecated

- [x] 2.1 `Diag` 加上 Deprecated 註解與各用法的對應，註明在 0.4 移除，保留原本行為。

## 3. 驗證與文件

- [x] 3.1 全套驗證通過。
- [x] 3.2 `Docs/stats.md`、CHANGELOG、`delivery-status.md`。
