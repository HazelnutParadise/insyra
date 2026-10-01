# Design: parquet-ccl-whole-file-aggregates

## Context

The CCL evaluator works on a `ccl.Context`. The DataTable path gives it the whole table; `parquet` gives it one 1,000-row batch at a time (`parquetContext`), so an aggregate's `GetColData`, `#`'s `GetRowIndex` and a fixed row's `GetCell` all see only the batch. Three changes make the streaming paths answer as the DataTable path does without holding whole columns; this design covers all three so the later two fit, and this change implements the first.

## Decisions

### 1. Resolve the whole-file parts first, then evaluate row by row

An expression is split into parts whose value depends on more than the current row ("whole-file parts") and the rest. `ccl.ResolveWholeTable` computes every whole-file part by reading the table in passes and replaces it with its value (`literalNode`), the same substitution `FoldRowInvariantAggregates` already makes on the DataTable path. What is left reads only the current row and is evaluated batch by batch exactly as before. An expression with no whole-file part costs no extra pass.

Whole-file parts in this change:
- an aggregate call whose arguments do not mention `#` (one that does is evaluated per row by the evaluator already, so it is row-local);
- a row access `X.r` whose row operand `r` is not row-dependent: a fixed row, a fixed row range, or one computed from other whole-file parts (`A.(COUNT(A) - 1)`).

`X.#` is the current row and stays row-local. Parts are resolved innermost first: a part is ready when nothing whole-file is left inside it; every ready part is resolved, substituted, and the walk repeats.

### 2. The row index through an optional interface

`ccl.GlobalRowContext` is a `Context` that holds part of a larger table: `GlobalRowIndex()` is the current row's position in the whole table. A row position written in an expression (`#`, the row in `A.#` or `A.5`) is a whole-table position; the evaluator reads `#` through `GlobalRowIndex` and, in `evaluateRowAccess`, turns a written row into the part's own row before it asks the context. Every context method (`GetCell`, `GetRowAt`, `GetRowIndex`, `SetRowIndex`, `GetRowCount`, …) works on the part's own rows, because the evaluator also walks a context's rows through them (`evaluateToColumn`, `rowShapedColumn`, `rowSliceForRange`). A context without the interface is unchanged.

### 3. Fixed rows are captured, then evaluated by the real evaluator

The resolver evaluates each fixed-row part's row operand against a context that reports the table's total row count (so a range is checked against the file, not the batch), collects the row positions, captures those whole rows in one pass, and evaluates the row access against a context serving the captured rows. The real `evaluateRowAccess` produces the value, so `A.0`, `@.3`, `A:C.5` and `A.0:1999` mean exactly what they mean on a loaded table. Memory is the captured rows only.

### 4. Built-in aggregates stream, in the order the aggregate function sees its values

`ccl.NewStreamingAggregate(name)` returns a reducer for `SUM`, `AVG`, `COUNT`, `MIN`, `MAX`, `VAR`, `VARP`, `STDEV` and `STDEVP` that is fed values in order and returns what the aggregate function returns for the same values given at once, bit for bit: it runs the same additions in the same order. The variance family needs the mean before the squared deviations, so it takes two passes over its values. A name a caller has re-registered with `RegisterAggregateFunction` has no streaming form, because its arithmetic is the caller's.

An aggregate's arguments are fed in order, each one whole before the next, because the aggregate function sees all of its first argument's values before any of its second's (`forEachValue`); a column range or `@` is fed one column at a time for the same reason. A row-varying argument is evaluated per batch with `evaluateToColumn` against the batch context; a constant one is fed once, in its turn. At each nesting level every ready aggregate advances one step per file pass, so independent aggregates share passes.

### 5. ApplyCCL resolves statement by statement

A statement may read a column an earlier statement created or replaced (`NEW('c') = A - AVG(A); NEW('d') = ['c'] / SUM(['c'])`). Statement k is therefore resolved against batches to which the already-resolved statements 0..k-1 have been applied, which is what `ExecuteCCL` sees on a loaded table.

The row-by-row half has to agree with that. Before this change, a statement evaluated in a batch read only the record the batch arrived as: a column an earlier `NEW` created was not found, and a column an earlier assignment replaced still held the file's values. Left that way, `['A'] = A - 1000; NEW('b') = A - MIN(A)` would take `MIN` from the replaced column and `A` from the file's, giving 1,001 where `ExecuteCCL` gives 1. The batch context therefore keeps the values each statement wrote, by column, and every accessor reads them before the record.

### 6. What this change refuses, and what the next two add

The resolver refuses, naming the part, anything it cannot compute yet: a sequence function call, `MEDIAN`, an aggregate or sequence function without a streaming form, and a row access whose row operand is row-dependent other than `#` itself. The second change streams the built-in sequence functions by carrying the last rows of each batch into the next (bounded look-back, look-ahead for `LEAD`, the running value for the cumulative functions). The third change computes everything else by loading only the columns the part reads into a whole-column context and evaluating the part with the real evaluator.

## Risks

- Extra passes cost time: the variance family reads the file twice per argument, and each nesting level adds passes. An expression without whole-file parts is unaffected.
- `NEW` columns still take their type from the first batch, as before.
