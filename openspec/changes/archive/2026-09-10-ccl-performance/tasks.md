# Tasks: ccl-performance

## 1. Tests first

- [x] 1.1 `internal/ccl/perf_correctness_test.go`：折疊後的聚合結果與手算相同；含 `#` 的聚合不被折疊；rolling 結果與測試內的樸素參考實作逐值相同；日期字串仍可運算、非日期字串行為不變；不同 regex 樣式在同一次求值中各自正確
- [x] 1.2 `internal/ccl/ccl_bench_test.go`：`A / SUM(A)`、z-score、字串運算、REGEX_MATCH、ROLLING_MEAN 的 benchmark

## 2. Implementation

- [x] 2.1 `ccl_evaluator.go`：`FoldRowInvariantAggregates`；求值失敗時不折疊，讓原本的錯誤路徑照常回報
- [x] 2.2 `ccl.go`：逐列迴圈前先折疊；`GetColData` 不再每次 copy
- [x] 2.3 `ccl_evaluator.go`：`parseTimeLike` 對字串先檢查首字元
- [x] 2.4 `stdlib_string.go`：有上限的編譯後 regex 快取
- [x] 2.5 `stdlib_sequences.go`：`seqRollingReduce` 先轉換整欄一次並重複使用緩衝區

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/CCL.md`：Performance 一節說明哪些會被折疊、`ROLLING_*` 為何維持 O(n·w)
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`
- [x] 3.3 `api-review.md`：CCL-18、CCL-38 標已修正；關閉 #355

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 0 issues
- [x] 4.2 改動前後同一組運算式的輸出逐值比對，確認完全相同
- [x] 4.3 `openspec validate ccl-performance --strict` 通過
