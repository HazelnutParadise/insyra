## MODIFIED Requirements

### Requirement: Stopping a SQL stream early releases its connection

`sqlio.Stream(ctx, db, dialect, table, batchSize, opts...)` and `sqlio.StreamQuery(ctx, db, dialect, query, batchSize, opts...)` SHALL return an `iter.Seq2[*insyra.DataTable, error]` that reads in the caller's goroutine. Leaving the loop before the last chunk SHALL close the query's rows and return its connection to the pool, without the caller cancelling anything. A failure SHALL arrive once, as a nil table with the error, and end the sequence; a `batchSize` below 1 SHALL be such a failure.

#### Scenario: Break after the first chunk
- **WHEN** a caller breaks out of the loop after the first chunk, without cancelling
- **THEN** the pool reports no connection in use

#### Scenario: Every chunk
- **WHEN** a caller ranges to the end of a 25-row table with `batchSize` 10
- **THEN** it receives chunks of 10, 10 and 5 rows and no error

#### Scenario: No database
- **WHEN** `db` is nil
- **THEN** the sequence yields one nil table with an error and ends

#### Scenario: A batch size below one
- **WHEN** `batchSize` is 0
- **THEN** the sequence yields one nil table with an error saying `batchSize must be positive` and ends
