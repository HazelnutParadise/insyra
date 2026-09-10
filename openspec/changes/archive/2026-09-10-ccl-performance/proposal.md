# Proposal: ccl-performance

## Why

An aggregate inside a per-row expression is recomputed for every row, over a fresh copy of the whole column. Measured on this branch, 20,000 rows: `A / 1` takes 0.9 ms, `A / SUM(A)` takes 2.1 s, and the z-score from `Docs/CCL.md` — `(A - AVG(A)) / STDEV(A)` — takes 6.0 s. That is quadratic work for a value that is the same on every row, and it makes the expression the documentation recommends unusable past a few thousand rows.

Three smaller costs sit on the same path. A string operand is offered to four `time.Parse` layouts before anything else happens, so a text column costs 45 ms per 100,000 rows where a numeric one costs 3.5 ms. `REGEX_MATCH` compiles its pattern once per row: 106 ms against `CONTAINS`'s 15 ms. And `ROLLING_*` converts every element out of `any` once per window it appears in and allocates a fresh slice for each window — at 100,000 rows and a window of 5,000 that is 500 million interface unboxings and 100,000 allocations, for 2.2 s.

Closes #355.

## What Changes

- **A row-invariant aggregate is computed once.** Before the row loop, aggregate subtrees that do not mention `#` are evaluated and replaced by their value. An aggregate that does mention `#` is row-dependent and is left alone.
- **The context stops copying a column per read.** `GetColData` hands back the snapshot it already holds; the snapshot is taken once per call and never written to.
- **A string is only offered to the date parsers when it could be a date.** All four layouts begin with a digit, so a string that does not is skipped without calling `time.Parse`.
- **`REGEX_MATCH` caches compiled patterns**, bounded so a generated pattern cannot grow the cache without limit.
- **`ROLLING_*` converts its column once** instead of once per window per element, and reuses one buffer instead of allocating per window.

## Capabilities

### New Capabilities

- `ccl-performance`: an expression does not repeat work whose answer cannot change.

### Modified Capabilities

(none)

## Impact

- `internal/ccl/ccl_compiler.go`, `ccl_evaluator.go`, `stdlib_string.go`, `stdlib_sequences.go`; `ccl.go`.
- **No result changes.** Every optimisation here preserves the values and their order: the folded aggregate is the same value the loop computed each time; the rolling window still receives the identical `[]float64` in the identical order, so its sums are bit-identical.
- **`ROLLING_*` stays O(n·window) on purpose.** A running accumulator would make it linear, but `sum -= leaving; sum += entering` accumulates rounding error that recomputation does not have, so a long window over values of mixed magnitude would return a different number than today. This change takes the constant factor and leaves the arithmetic alone.
