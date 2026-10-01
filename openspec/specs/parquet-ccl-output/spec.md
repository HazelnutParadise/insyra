# parquet-ccl-output Specification

## Purpose
How `ApplyCCL` writes the file it rewrites: with the original's compression and row group size unless `WriteOptions` says otherwise, with every column a statement writes typed so that no value is lost, through a temporary file of its own, and with its reader stopped whenever the call returns, as `FilterWithCCL`'s is.

## Requirements

### Requirement: ApplyCCL keeps the file's layout unless told otherwise

Without options, `ApplyCCL` SHALL write each column the original file has with the codec the original used for it, a column the script adds with the codec of the original's first column, and row groups holding at most as many rows as the original's largest row group. With one `WriteOptions`, it SHALL write every column with that `Compression` and row groups of that `RowGroupSize`, as `Write` does. More than one `WriteOptions`, or one `Write` refuses, SHALL be an error returned before the file is read, leaving it unchanged.

#### Scenario: A Zstd file in one row group
- **WHEN** 2,500 列、以 `CompressionZstd` 寫成一個 row group 的檔案經 `ApplyCCL(ctx, path, "NEW('b') = ['id'] * 2")` 改寫
- **THEN** 每個欄位區塊（含新欄位）都是 ZSTD，仍是一個 2,500 列的 row group，值正確

#### Scenario: The original's row group size
- **WHEN** 以 `CompressionSnappy`、`RowGroupSize: 700` 寫入 2,500 列後執行 `ApplyCCL`
- **THEN** row group 依序為 700、700、700、400，壓縮格式為 SNAPPY

#### Scenario: Columns with different codecs
- **WHEN** 原檔欄位 `a` 用 ZSTD、`b` 用 SNAPPY，`ApplyCCL` 新增欄位 `c`
- **THEN** `a` 仍是 ZSTD、`b` 仍是 SNAPPY、`c` 是 ZSTD

#### Scenario: Settings given
- **WHEN** `ApplyCCL(ctx, path, script, WriteOptions{Compression: CompressionGzip, RowGroupSize: 1000})` 改寫 2,500 列
- **THEN** 每個欄位區塊都是 GZIP，row group 為 1,000、1,000、500

#### Scenario: Settings that cannot be used
- **WHEN** `ApplyCCL` 收到 `RowGroupSize: -1` 或兩個 `WriteOptions`
- **THEN** 回傳錯誤，檔案位元組不變

### Requirement: ApplyCCL writes through a temporary file of its own

`ApplyCCL` SHALL write the new file through a temporary file with a name of its own in the file's directory and rename it into place only when the whole file was written, so a failure leaves the original as it was, a file named `<path>.tmp` is not touched, and no temporary file is left behind. An input with no rows SHALL leave the original untouched.

#### Scenario: A file named like the old temporary file
- **WHEN** `path + ".tmp"` 已有使用者的檔案，再執行 `ApplyCCL`
- **THEN** 該檔案內容不變，目錄裡除了這兩個檔案沒有其他檔案

### Requirement: A CCL call that returns stops its reader

`FilterWithCCL` and `ApplyCCL` SHALL stop the goroutine that reads the file when they return, an early return on an error included, so no reader stays blocked with the file open after the call.

#### Scenario: An expression that fails on the first batch
- **WHEN** 以 `context.Background()` 對 5,000 列的檔案連續 10 次呼叫會在第一批出錯的 `FilterWithCCL` 或 `ApplyCCL`
- **THEN** 每次都回傳錯誤，之後 goroutine 數量回到呼叫前的水準

### Requirement: ApplyCCL loses no value to a column's type

`ApplyCCL` SHALL write a column a statement assigns with the type the file gave it when every value written into it can be held by that type without loss, and otherwise, like a column a script creates, with the type `Write` gives a table column holding those values, settled from every value the column receives, not from the first batch. Every column a statement writes SHALL be nullable, so a missing value reads back as `nil`. A value that cannot be written, or a column type `ApplyCCL` cannot write back, SHALL be an error that leaves the file as it was, never a panic.

#### Scenario: A fraction in an integer column
- **WHEN** 對 int64 欄 `num` 為 `[1, 2, 3, 4]` 的檔案執行 `ApplyCCL(ctx, path, "['num'] = ['num'] * 1.5")`
- **THEN** 讀回的 `num` 是 float64 的 `[1.5, 3, 4.5, 6]`，不是截斷後的整數

#### Scenario: Whole numbers keep an integer column
- **WHEN** 對只有一個 int64 欄 `n`、值為 1 到 5 的檔案執行 `ApplyCCL(ctx, path, "['n'] = A * 2")`
- **THEN** 讀回的 `n` 仍是 int64 的 `[2, 4, 6, 8, 10]`

#### Scenario: A created column that starts with missing values
- **WHEN** 對 `A` 為 1 到 2,500、以 `WriteOptions{RowGroupSize: 1000}` 寫出的檔案執行 `ApplyCCL(ctx, path, "NEW('r') = ROLLING_MEAN(A, 1500)")`
- **THEN** 讀回的 `r` 前 1,499 列為 nil、其後是 float64，不是文字

#### Scenario: A missing value in a created column
- **WHEN** 對 `A` 為 `[1, nil, 3]` 的檔案執行 `ApplyCCL(ctx, path, "NEW('c') = A")`
- **THEN** 讀回的 `c` 是 `[1, nil, 3]`
