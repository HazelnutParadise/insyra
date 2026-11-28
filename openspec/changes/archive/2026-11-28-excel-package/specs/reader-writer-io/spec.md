## MODIFIED Requirements

### Requirement: Every format reads from a reader and writes to a writer

CSV and JSON in `insyra`, Excel in `excel` and Parquet in `parquet` SHALL each offer a read entry point that takes a source rather than a path, and CSV, JSON and Parquet SHALL each offer a write entry point that takes a destination. The existing path-based entry points SHALL call them, so a path and a reader over the same bytes produce the same table.

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

### Requirement: Excel readers infer column types

`excel.Read` and `excel.ReadFrom` SHALL infer each column's type by the rule CSV reading follows, after the header row and row names are taken out, and SHALL read a cell the file left out as empty. `ReadOptions.RawStrings` set to true SHALL keep every cell as text.

#### Scenario: Numeric and text columns

- **WHEN** a sheet has a column of 1, 2, 3, a column of 1.5, 2.5, 3 and a column of apple, 12, pear
- **THEN** the columns load as `int64`, `float64` and text, in that order

#### Scenario: A gap in a numeric column

- **WHEN** a numeric column has cells Excel left out
- **THEN** those cells read as `NaN` and the column is `float64`

#### Scenario: Keeping text

- **WHEN** `RawStrings` is true
- **THEN** every cell stays the text Excel displays
