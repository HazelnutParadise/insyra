# Tasks: fix-api-review-batch-4

## 1. Tests first

- [x] 1.1 `internal/ccl/batch4_test.go`：MapContext 順序、Duration 比較、巢狀序列、巨大引數不 panic、註冊併發、聚合跳 NaN、關鍵字大小寫
- [x] 1.2 `batch4_ccl_test.go`：越界欄位回錯、`@` 複本；`batch4_core_test.go`：ToCSV 寫入錯誤、原子輸出、巢狀 AtomicDoAll 不死鎖
- [x] 1.3 `csvxl/traversal_batch4_test.go`：手工改 workbook.xml 的 `../important`
- [x] 1.4 `cli/commands/batch4_test.go`、`cli/env/state_nan_test.go`、`cli/root_flags_test.go`
- [x] 1.5 `datalist_test.go` 11 個 TODO 測試補斷言；`stats/verify_more_test.go` 兩個邊界測試斷言

## 2. Implementation

- [x] 2.1 internal/ccl：tokenizer 關鍵字、`MaxResolvedColIndex`、`toFloat64` Duration、MapContext 排序、`evaluateToColumn` 巢狀、`scalarInt`／`runeSlice`／`REPEAT` 範圍、registry RWMutex 與 recover、SUM／AVG／collectFloats 跳 NaN
- [x] 2.2 `ccl.go`／`datatable_ccl.go`：`checkCCLColRange` 於三個 Bind 點；`GetCurrentRow` 複本
- [x] 2.3 `write_atomic.go`、`datatable_csv.go`（`writeCSV`）、`datatable_json.go`
- [x] 2.4 `internal/core/atomic.go`：AtomicDoN 持鎖再入走內聯
- [x] 2.5 `csvxl/convert.go`、`convertDir.go`：`safeSheetFileName`、先讀後寫、暫存檔
- [x] 2.6 cli：`col`／`row`／`timeseries` nil 檢查；`run` 深度與 OpenREPL；`root.go` 根旗標；`db_conn.go` SanitizeHistoryLine；`repl.go`／`api.go`／`registry.go` 寫入遮罩；`env/state.go` NaN 標記與新表格式；history 0600
- [x] 2.7 `.github/workflows/reference-verification.yml` `-run "ScikitLearn"`；`git rm insyra.test`；`.gitignore` `*.test`；`delivery-status.md` 重寫

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/CCL.md`、`Docs/cli-dsl.md`、`Docs/DataTable.md`、`Docs/csvxl.md`、`Docs/DataList.md`、`AGENTS.md`
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：Core、CLI、`csvxl`
- [x] 3.3 `api-review.md`：RP-1、RP-2、SEC-1、SEC-2、SEC-7、SEC-11、CLI-1～5、TS-1～3、IN-1、CCL-1（範圍與關鍵字）、CCL-2～8 標已修正；對應 issue 關閉

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`go test -race` core／internal/ccl／cli 無 race；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate fix-api-review-batch-4 --strict` 通過
