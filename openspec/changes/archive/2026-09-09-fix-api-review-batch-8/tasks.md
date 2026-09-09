# Tasks: fix-api-review-batch-8

## 1. Tests first

- [x] 1.1 `internal/ccl/batch8_test.go`：短路、CASE、逗號、字串序、nil 串接、`&` 優先級、Range／`@` 外洩、AND／OR 引數檢查、整數索引與視窗、日期小數天
- [x] 1.2 `batch8_ccl_test.go`（根套件）：經 `AddColUsingCCL`／`ExecuteCCL` 的端到端行為

## 2. Implementation

- [x] 2.1 `ccl_parser.go`：`&` 自成優先級；函數引數要求逗號分隔、拒絕尾逗號
- [x] 2.2 `ccl_evaluator.go`：`&&`／`||` 在 `cclBinaryOpNode` 與 `cclFoldChainNode` 短路；`CASE` 特判路徑短路
- [x] 2.3 `ccl_evaluator.go`：`applyOperator` 字串字典序、混型比較回錯、`&` 用 `toString`
- [x] 2.4 `ccl_evaluator.go`：`wholeIndex` 共用檢查，套用於 `evaluateRowAccess`、`evaluateRange`；`stdlib_sequences.go` 的 `scalarInt` 比照
- [x] 2.5 `ccl_evaluator.go`：`AND`／`OR` 特判路徑補引數檢查；序列函數拒絕 `@`
- [x] 2.6 `ccl_evaluator.go`／`ccl.go`：頂層結果為 Range 或整列時回錯
- [x] 2.7 `ccl_evaluator.go`：日期加減小數天保留小時以下精度
- [x] 2.8 `stdlib.go`：`CONCAT` 用 `toString`

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/CCL.md`：優先級表、型別強制轉換表更正（布林強制轉換、字串比較、nil 串接）、短路說明、`%` 運算子、整數索引規則
- [x] 3.2 `skills/insyra/references/`：CCL 使用說明同步
- [x] 3.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：CCL 區段，標示 BREAKING
- [x] 3.4 `api-review.md`：CCL-9～14、20、23～25、30、39 標已修正；關閉 #349、#350、#351、#352、#353、#357、#360、#361、#362

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate fix-api-review-batch-8 --strict` 通過
