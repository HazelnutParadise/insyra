# reader-writer-io Specification

## Purpose
How `insyra` reads CSV, JSON and Excel from an `io.Reader` and writes CSV and JSON to an `io.Writer`, how the path-based entry points relate to them, and how a CSV is read in batches.
## Requirements
### Requirement: Every format reads from a reader and writes to a writer

CSV, JSON and Excel in `insyra` SHALL each offer a read entry point that takes a source rather than a path, and CSV and JSON SHALL each offer a write entry point that takes a destination. The existing path-based entry points SHALL call them, so a path and a reader over the same bytes produce the same table.

#### Scenario: A CSV that is not on disk
- **WHEN** `ReadCSV` is given a reader over CSV bytes
- **THEN** the table equals what `ReadCSV_FileWithOptions` produces for a file holding the same bytes

#### Scenario: Encoding is detected on a reader
- **WHEN** `ReadCSV` is given Big5 bytes with `Encoding` left empty
- **THEN** the text is decoded as Big5

#### Scenario: Writing to a destination
- **WHEN** `WriteCSV` or `WriteJSON` writes to a buffer
- **THEN** the buffer holds the same bytes the path version writes to a file

#### Scenario: ReadJSON with a value implementing json.Marshaler and io.Reader
- **WHEN** `ReadJSON` is given a value that implements both `json.Marshaler` and `io.Reader`
- **THEN** the value is marshaled first rather than being read as a stream

#### Scenario: File reader preserves error text on detection read failure
- **WHEN** `ReadCSV_FileWithOptions` encounters a read error while auto-detecting the encoding
- **THEN** the original file read error text is preserved

### Requirement: A CSV can be read in batches

`StreamCSV` SHALL return an `iter.Seq2[*DataTable, error]` yielding at most `batchSize` data rows per table. With `FirstRowToColNames`, the header SHALL name the columns of every batch. Leaving the loop early SHALL stop reading.

#### Scenario: Every row, in batches
- **WHEN** a 25-row CSV with a header is streamed at batch size 10
- **THEN** three tables arrive, of 10, 10 and 5 rows, each with the header's column names

#### Scenario: Stopping early
- **WHEN** the loop breaks after the first batch of a large input
- **THEN** the rest of the input is not read

### Requirement: Excel readers let the caller choose text or typed columns

`ReadExcelSheet` and `ReadExcel` SHALL take an optional `rawStrings`: true SHALL keep every cell as text, false SHALL type columns by the rule CSV reading follows, and more than one value SHALL be an error. Left out, every cell SHALL stay text and one warning SHALL say columns are typed by default from 0.4.

#### Scenario: Left out

- **WHEN** a sheet holds a column of numbers and `rawStrings` is left out
- **THEN** the cells stay text

#### Scenario: False

- **WHEN** a sheet holds a column of whole numbers and `rawStrings` is false
- **THEN** the cells are `int64`

