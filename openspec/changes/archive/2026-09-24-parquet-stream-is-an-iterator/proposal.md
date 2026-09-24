## Why

`parquet.Stream` returned two channels. A consumer that left the loop early without cancelling its context left the producing goroutine parked on its next send for the life of the process (Q-7). The contract was written into the doc comment, but the trap was still there, and every caller had to know about goroutines and channels to use the function safely (#273, Q-9).

Go 1.23 added range-over-func: a function of type `iter.Seq2[K, V]` can be ranged over directly, and leaving the loop tells the function to stop. `0.4` requires Go 1.26. The owner chose on 2026-09-24 to replace `Stream` with that shape on `0.4`, with no deprecated channel version, the same way `lp`'s `SolveModel` was replaced when its old shape carried the defect.

## What Changes

- **BREAKING**: `Stream(ctx, path, opt, batchSize)` returns `iter.Seq2[*insyra.DataTable, error]`. Each batch arrives as a table with a nil error; a failure arrives once, as a nil table and the error, and ends the sequence. Leaving the loop stops the reader, so nothing is left running whether or not the caller cancels. Cancelling `ctx` ends the sequence with the context's error.
- The doc comment's warning about draining or cancelling goes, because there is nothing left to warn about.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `parquet-streaming-read`: gains the requirement that stopping early leaves nothing running.

## Impact

- `parquet/api.go`, `parquet/foreign_types_test.go`; `Docs/parquet.md`; both changelogs; `skills/insyra/` if it shows `Stream`.
- `api-review.md` Q-7 and Q-9, `delivery-status.md`, issue #273.
