# parquet-damaged-files Specification

## Purpose
What the Parquet readers do with a file they cannot make sense of: every reader, the streaming ones and the CCL ones included, returns an error naming the file instead of letting the Arrow reader's panic reach the caller, passing its bare error on, or reading a damaged row group as an empty one, and `ApplyCCL` leaves such a file as it was.

## Requirements

### Requirement: A damaged file is an error, never a panic

`Read`, `ReadFrom`, `ReadColumn`, `Inspect`, `Stream`, `StreamFrom`, `FilterWithCCL` and `ApplyCCL` SHALL return an error, and SHALL NOT panic, when the Arrow reader panics on the file. The error SHALL name the file and say it is not a readable Parquet file. `ApplyCCL` SHALL leave the original file as it was.

#### Scenario: Pages from one file, footer from another
- **WHEN** 把 1,000 列檔案的前 k 個位元組蓋到 3,000 列檔案上（k 從 100 起每次加 53），再以 `ReadFrom` 讀取
- **THEN** 每一次都回傳表格或錯誤，沒有任何一次 panic，且至少一次回傳錯誤

#### Scenario: The same file through every reader
- **WHEN** 以 `Read`、`Inspect`、`Stream`、`FilterWithCCL`、`ApplyCCL` 讀取一個會讓 Arrow panic 的檔案
- **THEN** 都不 panic；`Read`、`Stream`、`FilterWithCCL` 回傳錯誤，`ApplyCCL` 回傳錯誤且原檔位元組不變

### Requirement: A damaged row group is an error, never a short read

`Read`, `ReadFrom`, `ReadColumn`, `Stream`, `StreamFrom`, `FilterWithCCL` and `ApplyCCL` SHALL return an error naming the file and the row groups that did not read in full when the Arrow reader reports an error reading them, wrapping that error, or when the rows they read differ from the row count the file's metadata gives for the row groups they read, naming both counts too. A cancelled or expired context SHALL be returned as it is. `ApplyCCL` SHALL then leave the file as it was. Reading only row groups that are intact SHALL still succeed.

#### Scenario: A damaged third row group
- **WHEN** 一個以 `WriteOptions{RowGroupSize: 1000}` 寫出的 3,000 列檔案，第三個 row group 的頁標頭損壞，對它執行 `Read`
- **THEN** 回傳錯誤，說明檔案損壞、第 2 個 row group（從 0 數）沒能完整讀出，並以 `%w` 包住 Arrow 回報的錯誤，不回傳表格

#### Scenario: ApplyCCL on a damaged file
- **WHEN** 對同一個檔案執行 `ApplyCCL(ctx, path, "NEW('c') = A")`
- **THEN** 回傳錯誤，檔案位元組不變，目錄裡沒有暫存檔

#### Scenario: Reading only the intact row groups
- **WHEN** 對同一個檔案執行 `Read(ctx, path, ReadOptions{RowGroups: []int{0, 1}})`
- **THEN** 回傳 2,000 列，沒有錯誤

#### Scenario: A damaged Snappy page
- **WHEN** 一個以 `WriteOptions{Compression: CompressionSnappy, RowGroupSize: 1000}` 寫出的 3,000 列檔案，第三個 row group 第一頁的壓縮內容損壞，對它執行 `Read`、`Stream`、`FilterWithCCL` 與 `ApplyCCL`
- **THEN** 都回傳錯誤，程式不會結束，`ApplyCCL` 不改動原檔；只讀前兩個 row group 時回傳 2,000 列
