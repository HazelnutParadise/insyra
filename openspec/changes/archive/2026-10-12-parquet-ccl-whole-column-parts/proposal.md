# Proposal: parquet-ccl-whole-column-parts

## Why

`FilterWithCCL` and `ApplyCCL` read a Parquet file 1,000 rows at a time. `parquet-ccl-whole-file-aggregates` and `parquet-ccl-streamed-sequences` made the built-in aggregates other than `MEDIAN`, the row index, fixed rows and a top-level built-in sequence function give the loaded table's answer without holding whole columns. Whatever else reads beyond the current row is refused: `MEDIAN`, an aggregate a caller registered with `engine/ccl.RegisterAggregateFunction` or a built-in name re-registered, a row reference computed from the current row such as `A.(# - 1)`, a column range inside an expression an aggregate is computed over, and a sequence function anywhere but as a whole expression, `LAG(CUMSUM(A), 1)` and `SUM(CUMSUM(A))` included, which a loaded table answers. On 2026-10-01 the owner chose to compute these by reading into memory only the columns they use; this is the third of the three changes.

## What Changes

- `ApplyCCL`'s `@` gives each row its own values. `GetCurrentRow` in `parquet/ccl.go` returned the context's own slice, so every row of a batch ended up holding the batch's last row's values; it now returns a copy.

Left on 0.4: the whole-column path for the parts the streaming path cannot compute batch by batch (`MEDIAN`, a row reference computed from the current row, a sequence function outside a whole expression), and the `internal/ccl` helpers it needs. On this line those parts are evaluated batch by batch, as the batch requirement in `parquet-write-options` records.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `parquet-ccl-batches`, with a new requirement for `@` in `ApplyCCL`. It is added rather than modified because the requirement it would modify is not in the main specs yet.

## Impact

- `parquet/ccl.go`, `parquet/ccl_test.go`.
- `CHANGELOG.md`, `CHANGELOG_TW.md`.
