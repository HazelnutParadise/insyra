# Proposal: ccl-every-numeric-type

## Why

CCL reads a cell as a number through `toFloat64` and as a condition through `toBool`, and both know only `int`, `int32`, `int64`, `float32` and `float64`. A cell of any other Go integer type is not a number to CCL. Measured on 2026-10-02 with a two-row table whose column `A` holds the value 1, for each of `int8`, `int16`, `uint8`, `uint16`, `uint32`, `uint64` and `uint`:

- `SUM(A)` gives 0 and `MAX(A)` gives nil, with no error, because aggregates skip what they cannot read as a number.
- `A == 1` gives false with no error, because two values that are not both numbers compare as different types.
- `A * 2`, `IF(A, 'y', 'n')` and `ROUND(A, A)` fail with `invalid operands`, `cannot be converted to boolean` and `cannot convert int8 to number`.

`A.B`, which the docs describe as the value of `A` at the row `B` holds, accepts only an `int` or a `float64` row, so it fails with `invalid row index type` when `B` holds an `int32`, an `int64` or a `float32` as well. Columns of these types are ordinary: `parquet.Read` gives a file's `int8`, `int16` and unsigned columns as such cells, and DuckDB, Spark and pyarrow write them. The first two are wrong answers nobody is told about.

## What Changes

- Every Go integer type and `float32` is a number to CCL: arithmetic, comparison, conditions, functions and aggregates read it the way they read an `int64`. A `uint64` above 2^53 is read as the nearest `float64`, as an `int64` that large already is.
- A row index can be a cell of any numeric type and follows the whole-number rule the row index already has.
- `ISNA` and `IFNA` treat a `float32` NaN as missing, as they treat a `float64` one.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ccl-type-semantics`: every Go numeric type is a number.

## Impact

- `internal/ccl/stdlib.go` (`toFloat64`, `toBool`, `ISNA`, `IFNA`), `internal/ccl/ccl_evaluator.go` (`evaluateRowAccess`), `internal/ccl/stream_resolve.go` (`fixedRowIndices`). The streaming paths of `parquet.FilterWithCCL` and `ApplyCCL` read numbers through the same functions.
- Results change only where they were wrong or an error: an aggregate that answered 0 or nil answers the column's value, a comparison that answered false compares numbers.
- `Docs/CCL.md`, both changelogs, `AGENTS.md` (the follow-up is resolved), `delivery-status.md`.
