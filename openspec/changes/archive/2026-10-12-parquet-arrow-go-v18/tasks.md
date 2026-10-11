# Tasks: parquet-arrow-go-v18

## 1. Dependency

- [x] 1.1 `parquet` 改以 `github.com/apache/arrow-go/v18` v18.8.0 讀寫；`go.mod`、`go.sum` 已更新；`arrow.Record`、`array.NewRecord` 改為 `arrow.RecordBatch`、`array.NewRecordBatch`（`parquet/ccl.go`）
- [x] 1.2 Arrow 讀取器對某個 row group 回報錯誤時，回傳指名檔案與未完整讀出之 row group、並包住 Arrow 錯誤的錯誤（`readerFailed`）
- [x] 1.3 損壞的 Snappy 頁是錯誤而不結束程式，`ApplyCCL` 不改動原檔（`parquet/damaged_snappy_test.go`）
- [x] 1.4 duration 欄仍讀成 `int64`；page 大小上限設為 `1 << 40`，讓 v17 讀得進的檔案仍讀得進（`parquet/reader_writer_test.go`）

## 2. Changelog

- [x] 2.1 `CHANGELOG.md`／`CHANGELOG_TW.md`：`parquet`
