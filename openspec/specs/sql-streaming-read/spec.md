# sql-streaming-read Specification

## Purpose
串流讀取 SQL 查詢結果時，呼叫端中途停止也不會留下任何東西：`ReadSQLStream` 在呼叫端自己的 goroutine 裡逐批讀取，迴圈怎麼結束都會立刻關閉查詢結果、把連線還給連線池，不需要呼叫端記得 cancel。
## Requirements
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

