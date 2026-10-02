# Proposal: applyccl-keeps-every-column-type

## Why

`ApplyCCL` rebuilds every column of the file from Go values and builds only `int64`, `float64`, text, boolean, timestamp and binary arrays, so a file holding a column of any other type fails the whole call, even when no statement touches that column. Since `parquet-ccl-streamed-sequences` it fails with an error; before, it panicked. Such files are common: DuckDB, Spark and pyarrow write a `DATE` column as `Date32`, Spark's `IntegerType` and DuckDB's `INTEGER` as `int32`, and money as a decimal. A list or a struct column, which `Read` turns into `nil` cells, could never be written back even in principle. The owner approved making `ApplyCCL` take every column type on 2026-10-02.

## What Changes

- A column no statement writes is written back from the file's own Arrow data, unchanged, whatever its type: integer and float widths, dates, decimals, large strings, lists, structs and the rest.
- A column a statement assigns keeps its type, now also `int8` to `int32`, the unsigned widths, `float32`, `Date32` and `Date64`, when every value written into it fits that type without loss; otherwise it takes the type `Write` gives its values, as before.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `parquet-ccl-output`: a column `ApplyCCL` does not write keeps its data and type exactly; more types are kept for a column it writes.

## Impact

- `parquet/ccl.go`: runs carry the file's arrays beside their Go values, the stages keep them in step, the writer uses them for an unwritten column; the lossless check and the builder learn the narrower types.
- `Docs/parquet.md`, both changelogs, `AGENTS.md` (item 3 of the follow-up on what `ApplyCCL` does that a loaded table does not is resolved), `delivery-status.md`.
