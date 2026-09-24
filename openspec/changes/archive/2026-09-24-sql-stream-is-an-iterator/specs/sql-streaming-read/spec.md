## ADDED Requirements

### Requirement: Stopping a SQL stream early releases its connection

`ReadSQLStream` SHALL return an `iter.Seq2[*DataTable, error]` that reads in the caller's goroutine. Leaving the loop before the last chunk SHALL close the query's rows and return its connection to the pool, without the caller cancelling anything. A failure SHALL arrive once, as a nil table with the error, and end the sequence.

#### Scenario: Break after the first chunk
- **WHEN** a caller breaks out of the loop after the first chunk, without cancelling
- **THEN** the pool reports no connection in use

#### Scenario: Every chunk
- **WHEN** a caller ranges to the end of a 25-row table at chunk size 10
- **THEN** it receives chunks of 10, 10 and 5 rows and no error

#### Scenario: No database
- **WHEN** `db` is nil
- **THEN** the sequence yields one nil table with an error and ends
