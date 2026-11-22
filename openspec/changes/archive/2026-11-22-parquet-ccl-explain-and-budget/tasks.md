# Tasks: parquet-ccl-explain-and-budget

## 1. 測試先行

- [x] 1.1 `parquet/ccl_plan_test.go`：2,500 列檔案上 `A > AVG(A)`、`A > MEDIAN(A)`、全保留條件、`@`、跨批次的 `LAG`、`LAG(A, COUNT(A))`；比對 Explain 的模式、欄位與次數和實際呼叫讀檔的次數；`ExplainApplyCCL` 的逐句計畫
- [x] 1.2 同一檔案：`MaxRows` 超出與未超出、負數、兩個選項、`RefuseWholeColumns`；context 取消與提前停止後檔案已關閉

## 2. 實作

- [x] 2.1 `parquet/ccl.go`：把 `FilterWithCCL` 與 `ApplyCCL` 決定路徑的邏輯抽成共用函式
- [x] 2.2 `parquet/ccl_plan.go`：`CCLPlan`、`CCLMode`、`ExplainFilterWithCCL`、`ExplainApplyCCL`、`String`、`Show`
- [x] 2.3 `FilterWithCCLOptions`、`ErrCCLRowLimit`、`ErrCCLWholeColumnsRefused`

## 3. 文件與紀錄

- [x] 3.1 `Docs/parquet.md`：Explain、選項、`ApplyCCL` 先 Explain 再執行之間的空檔
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### parquet` 段落末尾新增條目

## 4. 驗證與移植

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate parquet-ccl-explain-and-budget --strict`
- [x] 4.2 cherry-pick 到 dev 並重跑 4.1
