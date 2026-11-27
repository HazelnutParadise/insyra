# Proposal: parquet-ccl-stream-filter

## Why

`FilterWithCCL` returns every kept row as one table, so a filter that keeps most of a large file holds most of it, and `MaxRows` can only refuse it. #415 part 3 asks for a scan, select and filter path whose output is an iterator or a sink, reusing `Stream` and the CCL batch semantics, with a table only when asked for. `design.md` has the design, reviewed independently; the owner asked on 2026-11-27 to go ahead with it.

## What Changes

- `StreamFilterWithCCL` gives the kept rows batch by batch as an `iter.Seq2[*insyra.DataTable, error]`.
- `FilterWithCCL` and the stream share one producer of kept rows, so they keep the same rows in the same order; `FilterWithCCL` collects them.
- `FilterWithCCLOptions.Columns` chooses the result's columns by name.
- The same API ships on `0.4` and `dev`.

## Capabilities

### New Capabilities

- `parquet-ccl-stream-filter`: filtering a Parquet file with CCL into a stream of tables.

## Impact

- Code: `parquet/ccl.go`, `parquet/ccl_plan.go`, a new `parquet/ccl_stream.go`.
- Tests: `parquet/ccl_stream_test.go`.
- Docs: `Docs/parquet.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
