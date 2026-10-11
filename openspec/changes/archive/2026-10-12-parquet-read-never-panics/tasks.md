# Tasks: parquet-read-never-panics

## 1. Tests

- [x] 1.1 `parquet/read_damaged_test.go`：疊合產生的損壞檔案經 `ReadFrom`、`Read`、`Inspect`、`Stream`、`FilterWithCCL`、`ApplyCCL` 都不 panic，該報錯的都報錯，`ApplyCCL` 不動原檔；`io.ReaderAt` 的 panic 經 `ReadFrom` 與 `StreamFrom` 轉為錯誤

## 2. Implementation

- [x] 2.1 `parquet/api.go`：`readTableFrom`、`Inspect` 與 `panicSafeReaderAt` 以 recover 把 Arrow 或 `io.ReaderAt` 的 panic 轉成指名檔案的錯誤
- [x] 2.2 `parquet/internal.go`：讀檔的 goroutine 以 recover 把 panic 送進錯誤 channel；讀取維持平行

## 3. Changelog

- [x] 3.1 `CHANGELOG.md`／`CHANGELOG_TW.md`：`parquet`
