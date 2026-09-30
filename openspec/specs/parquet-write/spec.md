# parquet-write Specification

## Purpose
The settings and cancellation of the Parquet writers: `WriteOptions` chooses the compression and row group size with a zero value that writes what `Write` always wrote, the `...Context` forms stop a write between columns and row groups, and every write to a path goes through a temporary file of its own so a failed or concurrent write never leaves a truncated or mixed file.

## Requirements

### Requirement: The writers take optional settings whose zero value changes nothing

`Write`, `WriteTo`, `WriteContext` and `WriteToContext` SHALL take an optional trailing `opts ...WriteOptions`. `WriteOptions.Compression` SHALL select the codec for every column: `CompressionNone` (the zero value), `CompressionSnappy`, `CompressionGzip`, `CompressionBrotli` or `CompressionZstd`. `WriteOptions.RowGroupSize` SHALL be the most rows one row group holds, with zero meaning 1,048,576. With no options or the zero `WriteOptions` the output SHALL be byte for byte what the writer produced before options existed. An unknown `Compression`, a negative `RowGroupSize` and more than one `WriteOptions` SHALL each return an error before anything is written, so `Write` leaves no file behind.

#### Scenario: Defaults are the old output
- **WHEN** 同一張表以 `WriteTo(dt, w)`、`WriteTo(dt, w, WriteOptions{})` 與未壓縮、每個 row group 1,048,576 列的 Arrow 寫入器各寫一次
- **THEN** 三份位元組完全相同

#### Scenario: Row groups
- **WHEN** 25 列的表以 `RowGroupSize: 10` 寫入後以 `Inspect` 讀取
- **THEN** 有三個 row group，列數依序為 10、10、5

#### Scenario: Compression
- **WHEN** 以 `CompressionZstd` 寫入後讀取檔案中繼資料
- **THEN** 每個欄位區塊的壓縮格式都是 ZSTD，且 `Read` 讀回的表與原表相同

#### Scenario: Settings that cannot be used
- **WHEN** `Write` 收到 `RowGroupSize: -1`、未知的 `Compression`，或兩個 `WriteOptions`
- **THEN** 回傳錯誤，目標路徑沒有檔案

### Requirement: A write can be cancelled

`WriteContext(ctx, dt, path, opts...)` and `WriteToContext(ctx, dt, w, opts...)` SHALL stop and return the context's error when `ctx` is done: before the table is converted, between columns while converting, before each row group and, for `WriteContext`, before the finished temporary file replaces `path`. A cancelled `WriteContext` SHALL leave the file at `path` as it was and no temporary file behind. A nil `ctx` SHALL be an error, not a panic. `Write` and `WriteTo` SHALL be `WriteContext` and `WriteToContext` with `context.Background()`.

#### Scenario: Cancelled before the write
- **WHEN** 以已取消的 context 呼叫 `WriteContext`，而 `path` 已有一個檔案
- **THEN** 回傳的錯誤 `errors.Is(err, context.Canceled)`，`path` 的內容不變，目錄裡沒有暫存檔

#### Scenario: A nil context
- **WHEN** 以 nil context 呼叫 `WriteContext` 或 `WriteToContext`
- **THEN** 回傳錯誤而不 panic，目錄裡沒有檔案，writer 沒有收到位元組

#### Scenario: Cancelled between row groups
- **WHEN** 以 `RowGroupSize: 1` 寫入多列的表，寫入目的地在收到第三次寫入時取消 context，此時已寫出至少一個 row group
- **THEN** `WriteToContext` 回傳 `context.Canceled`，不再寫完其餘 row group

### Requirement: Writes to one path do not mix

`Write` and `WriteContext` SHALL write through a temporary file with a name of its own in the directory of `path`, so two writes to the same path at once each leave either their own complete file or the other's at `path`, never a mix, a write the operating system refuses to rename leaves no temporary file, and a file the caller keeps at `<path>.tmp` is not touched.

#### Scenario: Two writers
- **WHEN** 兩個 goroutine 各自對同一路徑連續寫入 20 次，一個寫 1,000 列、一個寫 3,000 列
- **THEN** 每一邊至少有一次寫入回傳 nil（Windows 可能拒絕兩個同時改名到同一目標的其中一個），最後的檔案可讀，列數是 1,000 或 3,000，目錄裡沒有其他檔案

#### Scenario: A file named like the old temporary file
- **WHEN** `path + ".tmp"` 已有使用者的檔案，再呼叫 `Write(dt, path)`
- **THEN** 該檔案內容不變
