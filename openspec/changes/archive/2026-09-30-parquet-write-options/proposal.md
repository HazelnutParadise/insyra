# Proposal: parquet-write-options

## Why

[#272](https://github.com/HazelnutParadise/insyra/issues/272) holds two findings of the API review.

- **Q-2.** `Read`, `ReadFrom`, `Stream` and `ReadColumn` take a `context.Context`, but `Write(dt, path)` and `WriteTo(dt, w)` do not, so a large write cannot be cancelled. Neither takes settings: every file is written uncompressed with up to 1,048,576 rows in a row group, and a caller who wants Snappy or Zstd, or smaller row groups for a reader that skips by row group, has no way to ask.
- **Q-6.** `FilterWithCCL` and `ApplyCCL` read the file 1,000 rows at a time, and the number is a literal in each function. Checking whether it could be made settable showed that it cannot be treated as a tuning knob: each batch is evaluated on its own, so an expression that reads beyond the current row sees only its batch. Measured on 2026-09-30 with a file holding `A` = 1 to 2,500: `FilterWithCCL(ctx, path, "A > AVG(A)")` keeps 1,250 rows starting at 501, because each batch compares against its own average (500.5, 1,500.5, 2,250.5), where the same filter on the loaded table keeps the 1,250 rows from 1,251; `"# == 0"` and `"A == A.0"` keep rows 1, 1,001 and 2,001; `ApplyCCL(ctx, path, "NEW('i') = #")` numbers the rows 0 to 999 and starts again at 0. A settable batch size would let the same call return different rows depending on a performance setting. The same check found that a sequence function does not work in these paths at all: it is not evaluated row by row, so `ApplyCCL(ctx, path, "NEW('c') = CUMSUM(A)")` wrote the whole batch's running sum, as text, into every cell of `c`, where `AddColUsingCCL` on the loaded table gives each row its own sum. Nothing in `Docs/parquet.md` says any of this.

## What Changes

- New `WriteOptions{Compression, RowGroupSize}`, taken as an optional trailing `opts ...WriteOptions` by `Write` and `WriteTo`. `Compression` is a new `Compression` type with `CompressionNone` (the zero value), `CompressionSnappy`, `CompressionGzip`, `CompressionBrotli` and `CompressionZstd`, the codecs the Arrow library insyra writes with can encode. `RowGroupSize` is the most rows in one row group; zero means 1,048,576. The zero value writes byte for byte what `Write` wrote before. An unknown `Compression`, a negative `RowGroupSize` or more than one `WriteOptions` is an error, returned before anything is written.
- New `WriteContext(ctx, dt, path, opts...)` and `WriteToContext(ctx, dt, w, opts...)`, following the library's `...Context` pattern (`ReadSQLContext`, the `datafetch` fetches); `Write` and `WriteTo` call them with `context.Background()`. The context is checked before the conversion, between columns while converting, before each row group and, for `WriteContext`, before the finished file replaces `path`. A cancelled write returns the context's error, and `WriteContext` leaves `path` as it was. A nil context is an error rather than a panic, as in `datafetch`.
- `WriteContext`, and so `Write`, writes through the core package's temporary-file-and-rename helper, which moves to `internal/utils.WriteFileAtomically` so `parquet` and `lpgen` can share it, instead of the fixed `<path>.tmp`. Its temporary file has a name of its own. Reviewing this change showed that two writes to one path at once both reported success while leaving the other's data or a corrupt file, and that a file the caller kept at `<path>.tmp` was overwritten and then deleted.
- `FilterWithCCL` and `ApplyCCL` keep their batch of 1,000 rows as one named constant whose comment says why it is not a setting. `Docs/parquet.md` states that the rows are read in batches of 1,000, that an aggregate, the row index `#` or a fixed-row reference such as `A.0` sees only the current batch, and that a sequence function does not work there, and points to `Read` plus the `DataTable` CCL methods when an expression needs the whole column.
- Making the streaming CCL paths give whole-column answers, or refuse the expressions they cannot evaluate, sequence functions included, changes results and is recorded as an `AGENTS.md` follow-up for the owner to decide.

Adding the variadic parameter changes the type of `Write` and `WriteTo` as function values; every call written as a call still compiles.

## Capabilities

### New Capabilities

- `parquet-write`: the settings and cancellation of the Parquet writers.
- `parquet-ccl-batches`: how `FilterWithCCL` and `ApplyCCL` batch the file and what that means for an expression.

### Modified Capabilities

None.

## Impact

- `parquet/api.go`, `parquet/internal.go` (`dataTableToArrowTable` takes the context), `parquet/ccl.go`, new tests; `write_atomic.go` and the new `internal/utils/atomic_file.go`.
- `Docs/parquet.md`, `Docs/tutorials/parquet-inspection-streaming-and-filter.md` if it describes the batching, both changelogs, `AGENTS.md` (CCL batch follow-up), `api-review.md` (Q-2, Q-6), `delivery-status.md`.
