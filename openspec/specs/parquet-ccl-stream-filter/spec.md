# parquet-ccl-stream-filter Specification

## Purpose
Lets a caller filter a Parquet file with CCL into a stream of tables, so a filter that keeps most of a large file can go to a sink without the whole result in memory.
## Requirements
### Requirement: StreamFilterWithCCL gives the rows FilterWithCCL keeps

`StreamFilterWithCCL` SHALL give, batch by batch and in file order, exactly the rows `FilterWithCCL` returns for the same file, expression and options, without holding the rows it has given. When the filter keeps nothing it SHALL give one table with the columns and no rows.

#### Scenario: Every route across batches
- **WHEN** a 2,500-row file is filtered with `A > 100`, `A > AVG(A)`, `A > MEDIAN(A)`, `LAG(A, 1)` and a row computed from the current one
- **THEN** the batches appended, and `FilterWithCCL`'s table, both equal the rows the loaded table's own CCL keeps

#### Scenario: Nothing kept
- **WHEN** the filter is `A < 0` on a file of non-negative values
- **THEN** the stream gives one table with columns A and B and no rows

### Requirement: The stream ends cleanly

A failure SHALL arrive once as a nil table and the error and end the sequence. With `MaxRows`, the rows within the limit SHALL be given before the error matching `ErrCCLRowLimit`. Leaving the loop early SHALL stop the read and close the file.

#### Scenario: A row limit
- **WHEN** a filter keeping every row of a 2,500-row file is streamed with `MaxRows` 1,500
- **THEN** 1,500 rows are given, then an error matching `ErrCCLRowLimit`

### Requirement: The result's columns can be chosen

`FilterWithCCLOptions.Columns` SHALL give the result the named columns in the order given, for `FilterWithCCL` and the stream alike, while the filter can read any column. A name the file does not have, or one given twice, SHALL be an error before any data is read.

#### Scenario: A column the filter reads and the result leaves out
- **WHEN** a file is filtered with `A > 2400` and `Columns` `["B"]`
- **THEN** the result has the one column B

