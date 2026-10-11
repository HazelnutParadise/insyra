# Tasks: core-utils-cleanup

## 1. Tests first

- [x] 1.1 `utils_test.go`：`SqrtRat` 對 -1 與 nil 回傳 nil，對 9/4 回傳 3/2；`PowRat` 對 (2/3, -2) 回傳 9/4，對 (-2, -3) 回傳 -1/8，對 (0, -1) 與 (nil, 2) 回傳 nil。
- [x] 1.2 `datalist_deprecated_test.go`：確認 Deprecated 註解存在。

## 2. Implementation

- [x] 2.1 `utils.go`：`SqrtRat` 與 `PowRat` 不再 panic 並正確處理負次方；`SqrtRat`、`PowRat`、`SortTimes`、`F64orRat` 標記為 Deprecated 並註記在 0.4 移除。`mkt/cai.go` 改用 `slices.SortFunc`。

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/utils.md`：標記四個輔助函式為 Deprecated。
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：Core。
- [x] 3.3 `delivery-status.md`。

## 4. Verification

- [x] 4.1 全套驗證。
