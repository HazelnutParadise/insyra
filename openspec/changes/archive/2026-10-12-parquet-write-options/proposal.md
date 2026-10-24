# Proposal: parquet-write-options

## Why

[#272](https://github.com/HazelnutParadise/insyra/issues/272) holds two findings of the API review.

- **Q-2.** `Read`, `ReadFrom`, `Stream` and `ReadColumn` take a `context.Context`, but `Write(dt, path)` and `WriteTo(dt, w)` do not, so a large write cannot be cancelled. Neither takes settings: every file is written uncompressed with up to 1,048,576 rows in a row group, and a caller who wants Snappy or Zstd, or smaller row groups for a reader that skips by row group, has no way to ask.
- **Q-6.** `FilterWithCCL` and `ApplyCCL` read the file 1,000 rows at a time, and the number is a literal in each function. Checking whether it could be made settable showed that it cannot be treated as a tuning knob: each batch is evaluated on its own, so an expression that reads beyond the current row sees only its batch. Measured on 2026-09-30 with a file holding `A` = 1 to 2,500: `FilterWithCCL(ctx, path, "A > AVG(A)")` keeps 1,250 rows starting at 501, because each batch compares against its own average (500.5, 1,500.5, 2,250.5), where the same filter on the loaded table keeps the 1,250 rows from 1,251; `"# == 0"` and `"A == A.0"` keep rows 1, 1,001 and 2,001; `ApplyCCL(ctx, path, "NEW('i') = #")` numbers the rows 0 to 999 and starts again at 0. A settable batch size would let the same call return different rows depending on a performance setting. The same check found that a sequence function does not work in these paths at all: it is not evaluated row by row, so `ApplyCCL(ctx, path, "NEW('c') = CUMSUM(A)")` wrote the whole batch's running sum, as text, into every cell of `c`, where `AddColUsingCCL` on the loaded table gives each row its own sum. Nothing in `Docs/parquet.md` says any of this.

## What Changes

- New `WriteOptions{Compression, RowGroupSize}`, taken as an optional trailing `opts ...WriteOptions` by `WriteTo`, `WriteContext` and `WriteToContext`. `Compression` is a new `Compression` type with `CompressionNone` (the zero value), `CompressionSnappy`, `CompressionGzip`, `CompressionBrotli` and `CompressionZstd`, the codecs the Arrow library insyra writes with can encode. `RowGroupSize` is the most rows in one row group; zero means 1,048,576. The zero value writes the bytes the default writer wrote before options existed. An unknown `Compression`, a negative `RowGroupSize` or more than one `WriteOptions` is an error, returned before anything is written.
- `Write(dt, path)` keeps its signature and still writes in place. It takes no options on this line.
- New `WriteTo(dt, w, opts...)`, which writes to any `io.Writer` without closing it.
- New `WriteContext(ctx, dt, path, opts...)` and `WriteToContext(ctx, dt, w, opts...)`, following the library's `...Context` pattern (`ReadSQLContext`, the `datafetch` fetches). The context is checked before the table is converted, before each row group and, for `WriteContext`, before the finished file replaces `path`. A cancelled write returns the context's error, and `WriteContext` leaves `path` as it was. A nil context is an error rather than a panic, as in `datafetch`.
- `WriteContext` writes through `internal/utils.WriteFileAtomically`, which writes to a temporary file with a name of its own in the directory of `path` and renames it into place only once complete. Two writes to one path at once no longer mix, and a file the caller keeps at `<path>.tmp` is left alone.
- `FilterWithCCL` and `ApplyCCL` keep their batch of 1,000 rows as the named constant `cclBatchSize`.

Left on 0.4: the context check between columns while the table is converted (this line's `dataTableToArrowTable` takes no context), sharing the atomic helper with `lpgen`, the statement in `Docs/parquet.md` of what a batch does to aggregates, `#` and sequence functions, and giving whole-column answers from the streaming CCL paths, which stays the owner's decision.

## Capabilities

### New Capabilities

- `parquet-write`: the settings and cancellation of the Parquet writers.
- `parquet-ccl-batches`: how `FilterWithCCL` and `ApplyCCL` batch the file and what that means for an expression.

### Modified Capabilities

None.

## Impact

- `parquet/api.go`, `parquet/ccl.go` (the batch constant), `internal/utils/atomic_file.go` and its test (new).
- New tests `parquet/write_options_test.go`, `parquet/write_context_test.go`, `parquet/ccl_batch_test.go`, `parquet/reader_writer_test.go`.
- `Docs/parquet.md`, both changelogs.
