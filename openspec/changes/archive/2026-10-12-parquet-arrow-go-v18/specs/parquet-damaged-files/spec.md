## ADDED Requirements

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
