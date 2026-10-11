# parquet-damaged-files Specification

## Purpose
What every Parquet reader does with a file it cannot read: an error instead of a panic, and a damaged row group reported as an error naming it instead of a short read.

## Requirements

### Requirement: A damaged row group is an error, never a short read

`Read`, `ReadFrom`, `ReadColumn`, `Stream`, `StreamFrom`, `FilterWithCCL` and `ApplyCCL` SHALL return an error, naming the file, both row counts and the row groups that did not read in full, when the rows they read differ from the row count the file's metadata gives for the row groups they read.

#### Scenario: A damaged third row group
- **WHEN** 一個以 `WriteOptions{RowGroupSize: 1000}` 寫出的 3,000 列檔案，第三個 row group 的頁標頭損壞，對它執行 `Read`
- **THEN** 回傳錯誤，說明檔案損壞並指名第 2 個 row group（從 0 數）沒能完整讀出；讀到的列數與 metadata 不符時，錯誤同時列出 metadata 的列數與實際讀到的列數，不回傳表格

#### Scenario: ApplyCCL on a damaged file
- **WHEN** 對同一個檔案執行 `ApplyCCL(ctx, path, "NEW('c') = A")`
- **THEN** 回傳錯誤，檔案位元組不變，目錄裡沒有暫存檔

#### Scenario: Reading only the intact row groups
- **WHEN** 對同一個檔案執行 `Read(ctx, path, ReadOptions{RowGroups: []int{0, 1}})`
- **THEN** 回傳 2,000 列，沒有錯誤

### Requirement: A damaged file is an error, never a panic

`Read`, `ReadFrom`, `ReadColumn`, `Inspect`, `Stream`, `StreamFrom`, `FilterWithCCL` and `ApplyCCL` SHALL turn a panic inside the Arrow reader, or inside an `io.ReaderAt` the caller passed in, into an error, so that no such panic ends the program.

#### Scenario: Pages from one file, footer from another
- **WHEN** 把 1,000 列檔案的前 k 個位元組蓋到 3,000 列檔案上（k 從 100 起每次加 53），再以 `ReadFrom` 讀取
- **THEN** 每一次都回傳表格或錯誤，沒有任何一次 panic，且至少一次回傳錯誤

#### Scenario: The same file through every reader
- **WHEN** 以 `Read`、`Inspect`、`Stream`、`FilterWithCCL`、`ApplyCCL` 讀取一個會讓 Arrow panic 的檔案
- **THEN** 都不 panic；`Read`、`Stream`、`FilterWithCCL` 回傳錯誤，`ApplyCCL` 回傳錯誤且原檔位元組不變

#### Scenario: A reader that panics
- **WHEN** `ReadFrom` 或 `StreamFrom` 的 `io.ReaderAt` 在讀取 column chunk 時 panic
- **THEN** `ReadFrom` 回傳錯誤，`StreamFrom` 由錯誤通道送出錯誤，程式不結束

### Requirement: An Arrow reader failure on a row group is an error that names the row groups

`Read`, `ReadFrom`, `ReadColumn`, `Stream`, `StreamFrom`, `FilterWithCCL` and `ApplyCCL` SHALL return an error that names the file and the row groups that did not read in full, and wraps the error the Arrow reader reported, when that reader reports an error on a row group.

#### Scenario: A damaged third row group
- **WHEN** 一個以 `WriteOptions{RowGroupSize: 1000}` 寫出的 3,000 列檔案，第三個 row group 的頁標頭損壞，對它執行 `Read`
- **THEN** 回傳錯誤，說明檔案損壞、指名第 2 個 row group（從 0 數）沒能完整讀出，並以 `%w` 包住 Arrow 回報的錯誤，不回傳表格

#### Scenario: A context that ended during the read
- **WHEN** 讀取途中 context 已結束，而 Arrow 因此回報錯誤
- **THEN** 回傳的錯誤不包成檔案損壞，原樣回傳

#### Scenario: ApplyCCL on a damaged file
- **WHEN** 對同一個檔案執行 `ApplyCCL(ctx, path, "NEW('c') = A")`
- **THEN** 回傳錯誤，檔案位元組不變，目錄裡沒有暫存檔

#### Scenario: Reading only the intact row groups
- **WHEN** 對同一個檔案執行 `Read(ctx, path, ReadOptions{RowGroups: []int{0, 1}})`
- **THEN** 回傳 2,000 列，沒有錯誤

#### Scenario: A damaged Snappy page
- **WHEN** 一個以 `WriteOptions{Compression: CompressionSnappy, RowGroupSize: 1000}` 寫出的 3,000 列檔案，第三個 row group 第一頁的壓縮內容損壞，對它執行 `Read`、`Stream`、`FilterWithCCL` 與 `ApplyCCL`
- **THEN** 都回傳錯誤，程式不會結束，`ApplyCCL` 不改動原檔；只讀前兩個 row group 時回傳 2,000 列
