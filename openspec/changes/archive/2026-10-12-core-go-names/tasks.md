# Tasks: core-go-names

## 1. Tests first

- [x] 1.1 新測試：`ColIndexToNumber`／`ColNumberToIndex` 的已知值、0 到 20000 的來回轉換、非欄位輸入（舊程式碼沒有這兩個名稱，編譯失敗即為紅）
- [x] 1.2 新測試：`GetUseColoredOutput` 跟隨 `SetUseColoredOutput` 與 `SetDefaultConfig`（舊程式碼編譯失敗即為紅）
- [x] 1.3 新測試：三個舊名稱的 `Deprecated:` 段落指名新名稱與移除時程，回傳值與新名稱相同；`Dangerously_TurnOffThreadSafety` 的註解說明底線是刻意的（舊程式碼為紅）
- [x] 1.4 新測試：`isr` 的 `Of` 與 `From` 結果相同；`dl.Push` 可以接著串 `At`；`dt.Push(isr.Row{...})` 加一列（記錄裁定，舊程式碼即為綠）

## 2. Implementation

- [x] 2.1 `utils.go`：`ColIndexToNumber`、`ColNumberToIndex`，舊名稱改為 Deprecated 包裝
- [x] 2.2 `config.go`：`GetUseColoredOutput`，`GetDoesUseColoredOutput` 改為 Deprecated 包裝；`Dangerously_TurnOffThreadSafety` 註解說明刻意保留底線
- [x] 2.3 模組內所有呼叫者改用新名稱（`datatable_ccl.go`、`datatable_colname.go`、`describe_stats.go`、`utils.go`、與測試）

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/utils.md`、`Docs/Configuration.md`、`Docs/DataTable.md`：新名稱、舊名稱的對照
- [x] 3.2 `Docs/isr.md`：`Push` 與 `Append`／`AppendCols` 的差別，`Of` 與 `From` 為何並存
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core
- [x] 3.4 `AGENTS.md`：Key Conventions 記錄三個刻意例外與 `Push`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate core-go-names --strict`
