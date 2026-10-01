# Proposal: parquet-ccl-whole-file-aggregates

## Why

`FilterWithCCL` and `ApplyCCL` read a Parquet file 1,000 rows at a time and evaluate each batch on its own, so anything that reads beyond the current row sees only its batch. Measured on 2026-09-30 on a column `A` holding 1 to 2,500: `A > AVG(A)` keeps the rows from 501, where the same filter on the loaded table keeps the rows from 1,251; `SUM(A) > 1000000` keeps 1,500 rows; `# == 0` and `A == A.0` keep rows 1, 1,001 and 2,001; `ApplyCCL(ctx, path, "NEW('i') = #")` restarts at 0 every 1,000 rows. `parquet-write-options` documented this and recorded it as an `AGENTS.md` follow-up. On 2026-10-01 the owner chose to make these answers right without holding whole columns in memory, in three changes: aggregates, the row index and fixed rows first (this change), sequence functions second, and `MEDIAN` and caller-registered functions, which load only the column they use, third.

## What Changes

- The row index `#` is the row's position in the file in both functions, not in its batch.
- A reference to a fixed row or range of rows, such as `A.0`, `@.1500` or `A.0:1999`, reads those rows of the file, in whichever batch they are.
- `SUM`, `AVG`, `COUNT`, `MIN`, `MAX`, `VAR`, `VARP`, `STDEV` and `STDEVP` are computed over the whole file before the rows are evaluated, by reading the file in extra passes that keep only a few numbers, so memory still holds one batch. An aggregate nested in another (`AVG(A - AVG(A))`), one over a column range or `@`, one with several arguments and, in `ApplyCCL`, one over a column an earlier statement created, all give what the same script gives on the loaded table, bit for bit.
- An expression that needs none of this is read once, as before.
- Until the next two changes land, `FilterWithCCL` and `ApplyCCL` refuse, with an error naming it, an expression holding a sequence function (`LAG`, `CUMSUM`, …), `MEDIAN`, an aggregate or sequence function a caller registered, or a reference to a row computed from the current one (`A.(# - 1)`), where they used to return a per-batch answer without a word. A column range inside an expression an aggregate is computed over (`COUNT(IF(A > 0, A:B, 0))`) is refused too, because it expands to the batch rather than the column.
- In `ApplyCCL`, a statement reads what the statements before it wrote, as `ExecuteCCL` does; it used to fail on a column an earlier `NEW` created and read the file's values for one an earlier assignment replaced.
- A part that cannot be computed fails only the rows that reach it, as on the loaded table.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `parquet-ccl-batches`: aggregates, the row index and fixed rows are computed over the whole file; what is not yet supported is refused.

## Impact

- `internal/ccl`: the row index through a context that holds part of a table, streaming forms of the built-in aggregates, and the resolver that computes the whole-file parts of an expression.
- `parquet/ccl.go`: batch contexts that know their offset in the file, the passes the resolver asks for, and the two functions wired to them.
- `Docs/parquet.md`, `Docs/CCL.md` if it describes the streaming paths, both changelogs, `AGENTS.md` (the follow-up is narrowed to what the next two changes cover), `delivery-status.md`.
