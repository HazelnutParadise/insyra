# Design: parquet-ccl-whole-column-parts

## Context

The streaming path computes the parts of an expression that read beyond the current row by passes that keep running totals (`ResolveWholeTable`) or by streams that keep a window (`TopSequence`). `MEDIAN` needs every value at once, a caller's aggregate is a function of a whole slice, a row reference computed from the current row can reach any row, and a sequence function inside another expression is defined by the whole column it produces. None of these has a bounded streaming form.

## Decisions

### 1. Evaluate the whole expression, not just the part

When any part of an expression cannot be streamed, the whole expression is evaluated on a context holding the whole of the columns it reads. Replacing only the offending part would still need the streaming machinery for the rest, and a row reference computed from the current row has no single value to replace it with. Evaluating the whole expression with the ordinary evaluator gives the loaded table's answer by construction, its error messages included.

### 2. Exactly the table's procedure

The context is evaluated the way `applyCCLOnDataTable` and `executeNewColumn`/`executeAssignment` do: `ccl.Bind` against the file's column names, the column-past-the-end check, `FoldRowInvariantAggregates` (without it `A > MEDIAN(A)` sorts the column once per row), then per row when the expression depends on the row and once otherwise, a slice as long as the table spread over the rows. In `ApplyCCL` the existing `applyStatement` already follows the statement rules and runs on that context.

### 3. Only the columns it reads

`ccl.ReferencedColumns` lists the column positions a bound expression reads: column letters, bound names, every column of a range, and every column for `@`. A first pass reads those columns, whole; the context reports the file's full column count, so letters keep their meaning, and a column it did not load is an error rather than empty values. For `ApplyCCL` the pass runs the earlier statements' pipeline, so a column an earlier statement created or replaced is read as it then is.

### 4. When to take this path

The streaming path stays the default. An expression takes the whole-column path when `ResolveWholeTable` would refuse it, or when it is a top-level sequence function `NewTopSequence` cannot stream (a re-registered name, a column argument that does not change from row to row, an argument that changes). In those last cases the table's own procedure gives the table's answer or error, so falling back is always right.

## Risks

- Memory holds the columns such an expression reads, whole, and in `ApplyCCL` the column it writes until the file is written. `@` reads every column.
- One more read of the file per such filter or statement.
