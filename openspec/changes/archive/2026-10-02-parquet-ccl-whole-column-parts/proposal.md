# Proposal: parquet-ccl-whole-column-parts

## Why

`FilterWithCCL` and `ApplyCCL` read a Parquet file 1,000 rows at a time. `parquet-ccl-whole-file-aggregates` and `parquet-ccl-streamed-sequences` made the built-in aggregates other than `MEDIAN`, the row index, fixed rows and a top-level built-in sequence function give the loaded table's answer without holding whole columns. Whatever else reads beyond the current row is refused: `MEDIAN`, an aggregate a caller registered with `engine/ccl.RegisterAggregateFunction` or a built-in name re-registered, a row reference computed from the current row such as `A.(# - 1)`, a column range inside an expression an aggregate is computed over, and a sequence function anywhere but as a whole expression, `LAG(CUMSUM(A), 1)` and `SUM(CUMSUM(A))` included, which a loaded table answers. On 2026-10-01 the owner chose to compute these by reading into memory only the columns they use; this is the third of the three changes.

## What Changes

- An expression, or an `ApplyCCL` statement, holding a part that cannot be computed batch by batch is computed by reading the whole of the columns it reads, and only those, into memory, and evaluating it the way the `DataTable` CCL methods do: columns bound by name, a column past the last one refused, row-invariant aggregates folded, then row by row, or once and spread over the rows when the expression does not depend on the row. The answer, or the error, is the loaded table's.
- `@` reads every column, so an expression that needs whole columns and uses `@` holds the whole file.
- In `ApplyCCL`, such a statement is computed before the file is written, against the file as the statements before it leave it, and its column is held until it is written.
- Nothing is refused any more for being impossible to compute in batches.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `parquet-ccl-batches`: what used to be refused is computed by holding the columns it reads.

## Impact

- `internal/ccl`: telling whether an expression can be computed batch by batch, and listing the columns an expression reads.
- `parquet/ccl.go`: a whole-column evaluation for `FilterWithCCL` and a statement stage for `ApplyCCL` that writes a precomputed column.
- `Docs/parquet.md`, both changelogs, `AGENTS.md` (the follow-up on refused parts is resolved), `delivery-status.md`.
