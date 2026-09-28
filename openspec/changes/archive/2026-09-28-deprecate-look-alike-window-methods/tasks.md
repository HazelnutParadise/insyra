# Tasks: deprecate-look-alike-window-methods

## 1. Tests first

- [x] 1.1 新測試：解析 `datalist.go`，四個方法的 doc comment 都有指名替代呼叫的 `Deprecated: use` 段落，`ExponentialSmoothing` 沒有（先紅）。原行為由既有的兩個特性測試檔固定

## 2. Implementation

- [x] 2.1 `datalist.go`：四個 doc comment 加 Deprecated 段落；`interfaces.go`：`IDataList` 的四個方法同樣標 Deprecated（比照 `IDataTable.ToJSON_Bytes`），測試一併檢查
- [x] 2.2 `cli/commands/timeseries.go`：`movavg`、`diff` 的呼叫加 `// nolint:staticcheck`
- [x] 2.3 兩個特性測試檔的檔頭註解改指新的文件章節

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`：章節「Methods that look alike but differ」說明四個已 Deprecated、表格即遷移說明；四個方法各加 Deprecated 註記
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core
- [x] 3.3 `AGENTS.md`：下一版移除的 follow-up，含 CLI `movavg`／`diff` 的去向
- [x] 3.4 `api-review.md`：D-14 標為已修正；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate deprecate-look-alike-window-methods --strict`
