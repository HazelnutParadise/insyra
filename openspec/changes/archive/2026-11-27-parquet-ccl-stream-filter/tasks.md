# Tasks: parquet-ccl-stream-filter

## 1. 實作

- [x] 1.1 `parquet/ccl.go`：`FilterWithCCL` 改由 `filterKept` 產生保留的列再收集
- [x] 1.2 `parquet/ccl_stream.go`：`StreamFilterWithCCL`、`tableOf`、`projectColumns`
- [x] 1.3 `parquet/ccl_plan.go`：`FilterWithCCLOptions.Columns`，註解說明計畫也適用於串流

## 2. 測試

- [x] 2.1 `parquet/ccl_stream_test.go`：與載入後的表格比對、什麼都沒保留、`Columns`、`MaxRows`、提早離開迴圈、取消、nil context、CSV 寫出再讀回
- [x] 2.2 量測 1,000,000 列檔案的記憶體峰值

## 3. 文件

- [x] 3.1 `Docs/parquet.md`、`CHANGELOG.md`、`CHANGELOG_TW.md`

## 4. 驗證與移植

- [x] 4.1 `gofmt -s -l`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate --strict`
- [ ] 4.2 cherry-pick 到 dev 並重跑 4.1
