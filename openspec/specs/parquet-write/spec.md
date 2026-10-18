# parquet-write Specification

## Purpose
How a DataTable is written as a Parquet file: to a path or an io.Writer, with `WriteOptions` for compression and row group size, with or without a context, and which writes replace the target atomically.
## Requirements
### Requirement: The writers take optional settings whose zero value changes nothing

`WriteTo`, `WriteContext` and `WriteToContext` SHALL take an optional trailing `opts ...WriteOptions`; with no options or the zero `WriteOptions` they write the same bytes the default writer wrote before options existed.

#### Scenario: Defaults are the old output
- **WHEN** 同一張表以 `Write(dt, path)`、`WriteTo(dt, w)` 與 `WriteTo(dt, w, WriteOptions{})` 寫出，並與未壓縮、每個 row group 1,048,576 列的 Arrow 寫入器寫出的位元組比較
- **THEN** 所有位元組完全相同

#### Scenario: Row groups
- **WHEN** 25 列的表以 `WriteContext(ctx, dt, path, WriteOptions{RowGroupSize: 10})` 寫入後以 `Inspect` 讀取
- **THEN** 有三個 row group，列數依序為 10、10、5

#### Scenario: Compression
- **WHEN** 以 `WriteOptions{Compression: CompressionZstd}` 寫入後讀取檔案中繼資料
- **THEN** 每個欄位區塊的壓縮格式都是 ZSTD，且 `Read` 讀回的表與原表相同

#### Scenario: Settings that cannot be used
- **WHEN** `WriteContext` 收到 `RowGroupSize: -1`、未知的 `Compression`，或兩個 `WriteOptions`
- **THEN** 回傳錯誤，目標路徑沒有檔案；`WriteTo` 收到同樣的設定時，writer 沒有收到任何位元組

### Requirement: A write can be cancelled

`WriteContext` and `WriteToContext` SHALL return the context's error and stop before the next step once `ctx` is done.

#### Scenario: Cancelled before the write
- **WHEN** 以已取消的 context 呼叫 `WriteContext`，而 `path` 已有一個檔案
- **THEN** 回傳的錯誤 `errors.Is(err, context.Canceled)`，`path` 的內容不變，目錄裡沒有暫存檔

#### Scenario: Cancelled between row groups
- **WHEN** 以 `RowGroupSize: 1` 寫入多列的表，寫入目的地在收到第三次寫入時取消 context，此時已寫出至少一個 row group
- **THEN** `WriteToContext` 回傳 `context.Canceled`，不再寫完其餘 row group

#### Scenario: A nil context
- **WHEN** 以 nil context 呼叫 `WriteContext` 或 `WriteToContext`
- **THEN** 回傳錯誤而不 panic，目錄裡沒有檔案，writer 沒有收到位元組

### Requirement: WriteContext replaces its target only with a complete file

`WriteContext` SHALL write through a temporary file with a name of its own in the directory of `path` and replace `path` only once that file is complete.

#### Scenario: Two writers
- **WHEN** 兩個 goroutine 各自以 `WriteContext` 對同一路徑連續寫入 20 次，一個寫 1,000 列、一個寫 3,000 列
- **THEN** 每一邊至少有一次寫入回傳 nil（Windows 可能拒絕兩個同時改名到同一目標的其中一個），最後的檔案可讀，列數是 1,000 或 3,000，目錄裡沒有其他檔案

#### Scenario: A file named like the old temporary file
- **WHEN** `path + ".tmp"` 已有使用者的檔案，再呼叫 `WriteContext(ctx, dt, path)`
- **THEN** 該檔案內容不變

#### Scenario: Write stays in place
- **WHEN** 一個硬連結指向 `path`，再以 `Write(dt, path)` 寫入新表
- **THEN** 透過硬連結讀到的是新表，因為 `Write` 原地寫入、不換檔，與 `WriteContext` 不同

### Requirement: A column of byte slices is written as Binary
When every non-nil cell of a column is a `[]byte`, `Write` SHALL store the column as Parquet `Binary` and a later read SHALL return the same bytes.

#### Scenario: Bytes round trip
- **WHEN** a column holding `A-01`, `ff 00` and a nil is written and read back
- **THEN** the cells are the same bytes and the nil is nil

### Requirement: A byte slice among other kinds is written as text
When a column holds `[]byte` cells together with other kinds of value, `Write` SHALL store a `[]byte` that is valid UTF-8 as that text.

#### Scenario: Bytes beside text
- **WHEN** a column holding `x` and the bytes of `A-01` is written and read back
- **THEN** the second cell is `A-01`

