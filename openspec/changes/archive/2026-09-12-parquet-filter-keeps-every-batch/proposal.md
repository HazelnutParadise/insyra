# Proposal: parquet-filter-keeps-every-batch

## Why

`parquet.FilterWithCCL` silently returns the first 1000 matching rows and throws the rest away. Measured on 2026-09-12, on a one-column file filtered with a condition every row satisfies:

| rows in the file | rows returned | error |
| --- | --- | --- |
| 500 | 500 | none |
| 1000 | 1000 | none |
| 1001 | 1000 | none |
| 2500 | 1000 | none |
| 5000 | 1000 | none |

The cause is one line. The function reads the file in batches of 1000 and, from the second batch on, appends into `result.GetColByNumber(i)` — and `GetColByNumber` returns `dt.columns[index].Clone()`, a copy. Every batch after the first is appended to a value nothing keeps. `ApplyCCL` is not affected: it writes each batch straight to the parquet writer and was measured correct at 2500 rows.

This was found by the first test ever written against `parquet/ccl.go`, which was at 0% coverage until `test-unpinned-behaviour`. Nothing in the repository reads a Parquet file larger than one batch through this function.

Two smaller defects in the same read loop go with it, because they are the same three lines and leaving either behind would mean a follow-up entry for a two-line fix:

- **An error the producer reports after the last batch can be dropped.** `streamAsArrowRecord` sends the error into a buffered channel and then closes both that channel and the record channel. Once both are closed the consumer's `select` has two ready cases and the Go spec says it picks one at random, so a read error arriving after the consumer took its last batch is lost about half the time and the call returns a partial table with a nil error. Demonstrated in isolation: 1032 of 2000 runs lost the error when the producer finished first. In the shipped code the consumer usually reaches the `select` first, so 300 runs against a missing file did not reproduce it — the window is real but narrow, and the fix removes it rather than relying on the scheduler.
- **Every streaming call logs a warning it should not.** `Read` and `Inspect` ignore an `os.ErrClosed` from their second close, because Arrow's reader already closed the file. `streamAsArrowRecord` does not, so `Stream`, `FilterWithCCL` and `ApplyCCL` each print `failed to close file …: file already closed` on every successful call.

## What Changes

- `FilterWithCCL` collects matching rows in local slices across the whole stream and builds the result table once at the end. The first-batch and later-batch branches go away with it, so there is no longer a path where the two disagree. Peak memory is unchanged: the matching rows were always going to be held, since they are what the function returns.
- When the record channel closes, `FilterWithCCL` and `ApplyCCL` read the error channel before returning instead of letting `select` choose between them. `Stream` does the same before closing its own output.
- `streamAsArrowRecord` ignores `os.ErrClosed` on its close, matching `Read` and `Inspect`.

## Capabilities

### New Capabilities

- `parquet-streaming-read`: what the batch-by-batch read path guarantees — every row the filter matched, and the error when a read fails part-way. `parquet-read-column-limit` covers only `ReadColumn`'s `MaxValues` guard.

### Modified Capabilities

(none)

## Impact

- `parquet/ccl.go` (`FilterWithCCL`, `ApplyCCL`), `parquet/api.go` (`Stream`), `parquet/internal.go` (`streamAsArrowRecord`), `parquet/ccl_test.go`.
- User-visible: a filter over a file of more than 1000 rows returns a different, correct answer. Both changelogs get an entry under `### parquet`.
- `Docs/parquet.md` does not promise a row limit, so nothing there is wrong today — but it does not say what `FilterWithCCL` does with a read error either, and that is now worth one line.
- No API signature changes.
