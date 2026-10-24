## ADDED Requirements

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
