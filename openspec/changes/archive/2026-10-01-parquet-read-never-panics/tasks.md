# Tasks: parquet-read-never-panics

## 1. Tests first

- [x] 1.1 新測試：疊合產生的損壞檔案經 `ReadFrom`、`Read`、`Inspect`、`Stream`、`FilterWithCCL`、`ApplyCCL` 都不 panic，該報錯的都報錯，`ApplyCCL` 不動原檔（舊程式碼會 panic，即為紅）

## 2. Implementation

- [x] 2.1 `parquet/api.go`：`readTableFrom` 與 `Inspect` 以 recover 把 Arrow 的 panic 轉成指名檔案的錯誤
- [x] 2.2 `parquet/internal.go`：讀檔的 goroutine 以 recover 把 panic 送進錯誤 channel

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/parquet.md`：損壞的檔案回傳錯誤
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`parquet`
- [x] 3.3 `AGENTS.md`：移除已解決的 follow-up；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate parquet-read-never-panics --strict`
