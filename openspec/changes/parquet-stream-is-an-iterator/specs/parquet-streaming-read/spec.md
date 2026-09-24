## ADDED Requirements

### Requirement: Stopping a stream early leaves nothing running

`parquet.Stream` SHALL return an `iter.Seq2[*insyra.DataTable, error]`. Leaving the range loop before the last batch SHALL stop the reading goroutines without the caller cancelling anything. A read failure SHALL arrive once, as a nil table with the error, and end the sequence. Cancelling the context SHALL end the sequence with the context's error.

#### Scenario: Break after the first batch
- **WHEN** a caller ranges over a file of several batches and breaks after the first, without cancelling its context
- **THEN** every goroutine the stream started has returned

#### Scenario: Every batch
- **WHEN** a caller ranges to the end
- **THEN** it receives every row of the file and no error

#### Scenario: A file that cannot be opened
- **WHEN** the path does not exist
- **THEN** the sequence yields one nil table with the error and ends
