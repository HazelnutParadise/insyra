# Tasks: applyccl-keeps-file-layout

## 1. Tests first

- [x] 1.2 新測試：在第一批出錯的 `FilterWithCCL`／`ApplyCCL` 不留下卡住的讀檔 goroutine
- [x] 1.1 新測試：Zstd 單一 row group、Snappy 700 列一組、各欄不同壓縮、指定 `WriteOptions`、設定無效、`<path>.tmp` 不受影響（舊程式碼寫成未壓縮、1,000 列一組，即為紅）

## 2. Implementation

- [x] 2.1 `parquet/ccl.go`：讀原檔中繼資料得到各欄壓縮與 row group 大小；`ApplyCCL` 接受 `opts ...WriteOptions`；以 `WriteBuffered` 依 row group 大小寫入；經 `utils.WriteFileAtomically` 寫檔
- [x] 2.2 `parquet/ccl.go`：`FilterWithCCL` 與 `ApplyCCL` 返回時取消讀檔 goroutine 的 context（審查發現）

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/parquet.md`：`ApplyCCL` 的簽名、沿用原檔設定、`WriteOptions`、記憶體
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`parquet`
- [x] 3.3 `AGENTS.md`：移除已解決的 follow-up；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate applyccl-keeps-file-layout --strict`
