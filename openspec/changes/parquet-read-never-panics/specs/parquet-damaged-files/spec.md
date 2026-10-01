## ADDED Requirements

### Requirement: A damaged file is an error, never a panic

`Read`, `ReadFrom`, `ReadColumn`, `Inspect`, `Stream`, `StreamFrom`, `FilterWithCCL` and `ApplyCCL` SHALL return an error, and SHALL NOT panic, when the Arrow reader panics on the file. The error SHALL name the file and say it is not a readable Parquet file. `ApplyCCL` SHALL leave the original file as it was.

#### Scenario: Pages from one file, footer from another
- **WHEN** 把 1,000 列檔案的前 k 個位元組蓋到 3,000 列檔案上（k 從 100 起每次加 53），再以 `ReadFrom` 讀取
- **THEN** 每一次都回傳表格或錯誤，沒有任何一次 panic，且至少一次回傳錯誤

#### Scenario: The same file through every reader
- **WHEN** 以 `Read`、`Inspect`、`Stream`、`FilterWithCCL`、`ApplyCCL` 讀取一個會讓 Arrow panic 的檔案
- **THEN** 都不 panic；`Read`、`Stream`、`FilterWithCCL` 回傳錯誤，`ApplyCCL` 回傳錯誤且原檔位元組不變
