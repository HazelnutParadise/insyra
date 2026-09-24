# reader-writer-io Specification

## Purpose
每種格式都能從任意來源讀、寫到任意去處，不必先落地成檔案：`insyra` 的 CSV、JSON、Excel 與 `parquet` 各有收 `io.Reader`／`io.Writer`（Parquet 讀取為 `io.ReaderAt` 加大小）的入口，路徑版與字串版只是呼叫它們，所以同樣的位元組走哪個入口都得到同樣結果；CSV 另有逐批讀取、只佔一批記憶體的 `StreamCSV`。
## Requirements
### Requirement: Every format reads from a reader and writes to a writer

CSV, JSON and Excel in `insyra`, and Parquet in `parquet`, SHALL each offer a read entry point that takes a source rather than a path, and CSV, JSON and Parquet SHALL each offer a write entry point that takes a destination. The existing path-based entry points SHALL call them, so a path and a reader over the same bytes produce the same table.

#### Scenario: A CSV that is not on disk
- **WHEN** `ReadCSV` is given a reader over CSV bytes
- **THEN** the table equals what `ReadCSV_FileWithOptions` produces for a file holding the same bytes

#### Scenario: Encoding is detected on a reader
- **WHEN** `ReadCSV` is given Big5 bytes with `Encoding` left empty
- **THEN** the text is decoded as Big5

#### Scenario: Writing to a destination
- **WHEN** `WriteCSV`, `WriteJSON` or `parquet.WriteTo` writes to a buffer
- **THEN** the buffer holds the same bytes the path version writes to a file

#### Scenario: Parquet from memory
- **WHEN** `parquet.ReadFrom` is given a `*bytes.Reader` over a Parquet file and its size
- **THEN** the table equals what `parquet.Read` produces from the file

### Requirement: A CSV can be read in batches

`StreamCSV` SHALL return an `iter.Seq2[*DataTable, error]` yielding at most `batchSize` data rows per table. With `FirstRowToColNames`, the header SHALL name the columns of every batch. Leaving the loop early SHALL stop reading.

#### Scenario: Every row, in batches
- **WHEN** a 25-row CSV with a header is streamed at batch size 10
- **THEN** three tables arrive, of 10, 10 and 5 rows, each with the header's column names

#### Scenario: Stopping early
- **WHEN** the loop breaks after the first batch of a large input
- **THEN** the rest of the input is not read

