## ADDED Requirements

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
