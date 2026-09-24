## Why

`ReadSQLStream` returned a channel of `ReadSQLChunk` and documented the same contract `parquet.Stream` used to: a caller who stops reading early must cancel `ctx`, or a goroutine and its database connection leak. Measured on 2026-09-24, the documented remedy does not work. After a caller breaks out of the loop and cancels, the reader goroutine reaches its `ctx.Done()` branch and sends `ReadSQLChunk{Err: ctx.Err()}` on the unbuffered channel, which nobody reads any more, so it parks for the rest of the process holding the `*sql.Rows` open — one connection lost from the pool per abandoned stream. A probe that broke after the first chunk and then cancelled still had the reader running two seconds later. The review had marked the function as a template for `parquet.Stream` (E-9), and it was the one with the worse defect.

The owner chose on 2026-09-24 to give it the same shape `parquet.Stream` now has.

## What Changes

- **BREAKING**: `ReadSQLStream(ctx, db, tableName, options...)` returns `iter.Seq2[*DataTable, error]`, used as `for dt, err := range insyra.ReadSQLStream(...)`. It reads chunks in the caller's own goroutine, so there is no goroutine to leak, and the rows are closed the moment the loop ends, however it ends.
- **BREAKING**: `ReadSQLChunk` is removed; the iterator's two values carry what it carried.
- The query runs when the loop starts rather than when the function is called, so a failure to build or run it arrives as the first value of the loop instead of as a second return value.

## Capabilities

### New Capabilities
- `sql-streaming-read`: what a streamed SQL read guarantees when the caller stops early.

### Modified Capabilities
None.

## Impact

- `datatable_from_sql.go`, `datatable_from_sql_test.go`, `interfaces.go` if it lists the function; `Docs/DataTable.md`; both changelogs; `skills/insyra/` if it shows the function.
- `api-review.md` E-9 and the `ReadSQLStream` / `ReadSQLChunk` checklist lines, `delivery-status.md`.
