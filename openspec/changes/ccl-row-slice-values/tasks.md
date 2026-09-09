# Tasks: ccl-row-slice-values

## 1. Tests first

- [x] 1.1 `internal/ccl/batch8_test.go`：`A:B` 為當前列切片、`LAG(@,1)` 等於 `LAG(@.#,1)`、`LAG(A:B,1)`、算術類序列函數仍回錯、裸列範圍仍回錯
- [x] 1.2 `batch8_ccl_test.go`：同樣行為經 `AddColUsingCCL`，並加上範圍消費端的迴歸守衛（`SUM(A:C)`、`SUM((A:C).(2:5))`、`SUM(A.(0:1))`）

## 2. Implementation

- [x] 2.1 `ccl_evaluator.go`：`Evaluate` 把頂層 `ColumnRange` 展開成當前列切片；裸 `RowRange` 維持錯誤但改寫訊息
- [x] 2.2 `ccl_evaluator.go`：`IsRowDependent` 只對「兩邊都是欄位」的 `:` 回 true，列範圍維持原樣
- [x] 2.3 `ccl_evaluator.go`：`evaluateToColumn` 在問逐列與否之前先展開欄位範圍，聚合路徑不受影響
- [x] 2.4 `ccl_evaluator.go`：序列函數的 `@`／欄位範圍引數改成逐列一格
- [x] 2.5 `stdlib_sequences.go`：`SequenceFunctionTakesRows`，只有 LAG／LEAD

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/CCL.md`：改寫「範圍不是值」與序列函數兩段
- [x] 3.2 `skills/insyra/references/ccl-operators.md`：同步
- [x] 3.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：修正 batch 8 那條（尚未發布）
- [x] 3.4 在 #360 補說明修正方式已改變

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`go test -race` 於受影響套件；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate ccl-row-slice-values --strict` 通過
