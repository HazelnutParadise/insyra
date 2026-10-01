# Proposal: applyccl-keeps-file-layout

## Why

`ApplyCCL` rewrites a Parquet file in place, and it wrote the new file with Arrow's defaults whatever the original used: uncompressed, one row group per 1,000-row batch. Measured on 2026-10-01: a 200,000-row file written with `CompressionZstd` in one row group was 1.7 MB, and after `ApplyCCL(ctx, path, "NEW('b') = ['id'] * 2")` it was 9.3 MB in 200 row groups; the added column accounts for about 1.6 MB of that uncompressed. It also wrote through the fixed `<path>.tmp`, the pattern `parquet-write-options` removed from `Write` because two writers to one path mixed their bytes. The owner chose on 2026-10-01 to keep the original's layout by default and let the caller set it, the `AGENTS.md` follow-up recorded on 2026-09-30.

## What Changes

- `ApplyCCL` writes each column with the codec the original file used for it, a column the script adds with the codec of the original's first column, and row groups as large as the original's largest one.
- It takes an optional trailing `opts ...WriteOptions`. Given one, it writes the whole file with that `Compression` and `RowGroupSize`, exactly as `Write` would; more than one, or one `Write` would refuse, is an error before anything is read or written.
- It writes through a temporary file with a name of its own (`internal/utils.WriteFileAtomically`), so a file the caller keeps at `<path>.tmp` is left alone. An input with no rows still leaves the original untouched.
- `FilterWithCCL` and `ApplyCCL` stop their reader when they return. Reviewing this change found that a call returning early with an error left the goroutine reading the file blocked on its next batch, with the file open, for as long as the caller's context lived, which for `context.Background()` is the rest of the program.
- To build row groups larger than one batch, it holds the row group being written in memory, as the reader already holds one row group of the input.

## Capabilities

### New Capabilities

- `parquet-ccl-output`: how `ApplyCCL` writes the file it rewrites.

### Modified Capabilities

None.

## Impact

- `parquet/ccl.go`, new `parquet/applyccl_layout_test.go` and `parquet/ccl_reader_leak_test.go`.
- `Docs/parquet.md`, both changelogs, `AGENTS.md` (the follow-up is resolved and removed), `delivery-status.md`.
