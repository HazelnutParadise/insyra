# Tasks: parquet-write-options

## 1. Tests first

- [x] 1.1 新測試：預設輸出與 `WriteOptions{}`、與直接以 Arrow 未壓縮、1,048,576 列一組寫出的位元組相同；`RowGroupSize: 10` 寫 25 列得到 10、10、5 三組；每種 `Compression` 寫出的欄位區塊帶對應的壓縮格式且讀回相同；負的 `RowGroupSize`、未知的 `Compression`、兩個 `WriteOptions` 回傳錯誤且不留檔（舊程式碼沒有 `WriteOptions`，不能編譯，即為紅）
- [x] 1.2 新測試：已取消的 context 讓 `WriteContext` 回傳 `context.Canceled`、原檔不變、沒有暫存檔；`WriteToContext` 在寫出第一批位元組後取消即停止
- [x] 1.4 新測試：nil context 回傳錯誤；兩個 goroutine 同時寫同一路徑不會混在一起；`<path>.tmp` 的既有檔案不受影響；在寫出 row group 之後才取消也會停下（前三項在舊程式碼上為紅）
- [x] 1.3 新測試：`FilterWithCCL` 的 `A > AVG(A)` 與 `ApplyCCL` 的 `NEW('i') = #` 依 1,000 列一批計算，釘住文件所述的行為

## 2. Implementation

- [x] 2.1 `parquet/api.go`：`Compression`、`WriteOptions`、`WriteContext`、`WriteToContext`；`Write`、`WriteTo` 改為呼叫 Context 版本；設定在建立暫存檔前驗證；逐 row group 寫入並檢查 context
- [x] 2.2 `parquet/internal.go`：`dataTableToArrowTable` 接收 context，每欄之前檢查
- [x] 2.4 `internal/utils/atomic_file.go`：核心套件的 `writeFileAtomically` 搬成 `utils.WriteFileAtomically`，`write_atomic.go` 改為呼叫它
- [x] 2.5 `parquet/api.go`：`WriteContext` 改經 `utils.WriteFileAtomically` 寫入（暫存檔名各自不同）；nil context 回傳錯誤（審查發現）
- [x] 2.3 `parquet/ccl.go`：批次大小改為具名常數，註解說明它不是設定的理由

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/parquet.md`：`WriteOptions`、`Compression`、`WriteContext`／`WriteToContext`；CCL 的批次與逐批計算
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`parquet`
- [x] 3.3 `AGENTS.md`：串流 CCL 逐批計算、`ApplyCCL` 不保留原檔壓縮與 row group 的 follow-up
- [x] 3.4 `api-review.md`：Q-2、Q-6 標為已修正；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate parquet-write-options --strict`
