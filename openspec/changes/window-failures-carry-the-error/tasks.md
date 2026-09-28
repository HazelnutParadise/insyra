# Tasks: window-failures-carry-the-error

## 1. Tests first

- [x] 1.1 `DataList`：Rolling 三種不合法選項、`Apply(nil)`、`Corr`/`Cov`/`Beta(nil)`、EWM 不合法衰減參數，結果都帶錯誤；合法但觀察數不足仍是整列 nil 且無錯誤（先紅）
- [x] 1.2 `DataTable`：`RollingCol`／`EWMCol` 不合法選項同時記在表與結果；七個逐欄轉換與三個 builder 找不到欄時結果帶錯誤（先紅）
- [x] 1.3 Grouped：`As` 的錯誤結果帶錯誤；組內不合法選項（Rolling、Diff、Shift）不再產生整欄 nil（先紅）
- [x] 1.4 CLI：`rolling x 0 mean`、`rolling x 2 mean minobs 3` 回錯誤且不存變數（先紅）

## 2. Implementation

- [x] 2.1 `datalist_window.go`、`datalist_ewm.go`：view 的 `err` 改存 `*ErrorInfo`，reducer 的失敗結果帶著它
- [x] 2.2 `datatable_window.go`：找不到欄的結果帶錯誤；`RollingCol`／`EWMCol` 的選項錯誤以 `setError` 記到表
- [x] 2.3 `datatable_groupby_window.go`：`As` 的失敗結果帶錯誤，組內失敗即停並記到表
- [x] 2.4 `cli/commands/timeseries.go`：三個命令檢查結果的錯誤

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`：Rolling／EWM／Expanding 的失敗說明與 Error Handling 段落
- [x] 3.2 `Docs/DataTable.md`：逐欄與分組轉換的 Errors 段落
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core 與 CLI
- [x] 3.4 `api-review.md`：D-18 標為已修正，註明未採用「同長度 nil」的理由
- [x] 3.5 `delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate window-failures-carry-the-error --strict`
