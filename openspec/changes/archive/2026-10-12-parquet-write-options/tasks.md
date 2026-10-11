# Tasks: parquet-write-options

## 1. Tests

- [x] 1.1 `parquet/write_options_test.go`：預設輸出與 `WriteOptions{}`、與直接以 Arrow 未壓縮、1,048,576 列一組寫出的位元組相同；`RowGroupSize: 10` 寫 25 列得到 10、10、5 三組；每種 `Compression` 寫出的欄位區塊帶對應的壓縮格式且讀回相同；負的 `RowGroupSize`、未知的 `Compression`、兩個 `WriteOptions` 回傳錯誤且不留檔
- [x] 1.2 `parquet/write_context_test.go`：已取消的 context 讓 `WriteContext` 回傳 `context.Canceled`、原檔不變、沒有暫存檔；`WriteToContext` 在寫出 row group 之間取消即停止
- [x] 1.3 `parquet/ccl_batch_test.go`：`FilterWithCCL` 的 `A > AVG(A)` 與 `ApplyCCL` 的 `NEW('i') = #` 依 1,000 列一批計算
- [x] 1.4 `parquet/write_context_test.go`：nil context 回傳錯誤；兩個 goroutine 同時寫同一路徑不會混在一起（測試實際呼叫的是 `Write`，不是 `WriteContext`）；`<path>.tmp` 的既有檔案不受影響（測試實際呼叫的是 `Write`）

## 2. Implementation

- [x] 2.1 `parquet/api.go`：`Compression`、`WriteOptions`、`WriteContext`、`WriteToContext`、`WriteTo`；`Write(dt, path)` 保持原簽名與原地寫入；設定在建立任何輸出前驗證；逐 row group 寫入並檢查 context
- [x] 2.2 `internal/utils/atomic_file.go`：`WriteFileAtomically`；`WriteContext` 經它寫入，暫存檔名各自不同
- [x] 2.3 `parquet/ccl.go`：批次大小改為具名常數 `cclBatchSize`
- [x] 2.4 `WriteContext` 與 `WriteToContext` 收到 nil context 回傳錯誤，不 panic

## 3. Docs and changelog

- [x] 3.1 `Docs/parquet.md`：`WriteOptions`、`Compression`、`WriteContext`、`WriteToContext`、`WriteTo`
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`parquet`
