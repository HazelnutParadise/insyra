# Proposal: parquet-ccl-explain-and-budget

## Why

`FilterWithCCL` and `ApplyCCL` read a file in batches of 1,000 rows, but whether an expression reads the file again for an aggregate, loads whole columns into memory, or keeps every matching row is decided inside the call, where a caller cannot see it or cap it. On a large file the first sign is the memory running out. #415 part 2 asks for a summary that evaluates no data and for limits on what a filter may hold. `design.md` has the API, revised after an independent review and confirmed by the owner on 2026-11-22.

## What Changes

- `ExplainFilterWithCCL` and `ExplainApplyCCL` return a `CCLPlan` for an expression, or one per statement, read from the file's metadata and the expression alone: the mode (`CCLStreamed`, `CCLStreamedSequence`, `CCLWholeColumns`), the reasons, the columns needed and held, the minimum number of data passes, the values held, whether the result is held, whether the data can make the call read more, and an estimate of the held columns' size. `CCLPlan` has `String()` and `Show()`; `CCLMode` has `String()`.
- `FilterWithCCL` takes an optional `FilterWithCCLOptions{MaxRows, RefuseWholeColumns}`. Exceeding `MaxRows` returns `nil` and an error matching `ErrCCLRowLimit`; an expression needing whole columns under `RefuseWholeColumns` fails before any data pass with `ErrCCLWholeColumnsRefused`.
- `FilterWithCCL` and `ApplyCCL` decide their path through the same function Explain uses, so the plan and the call cannot disagree.
- The change ships on `0.4` and on `dev` with the same API.

## Capabilities

### New Capabilities

- `parquet-ccl-explain`: planning a CCL filter or script on a Parquet file without evaluating rows, and limiting what a filter holds.

## Impact

- Code: `parquet/ccl.go`, a new `parquet/ccl_plan.go`, `internal/ccl` for the structural checks the plan needs.
- Tests: `parquet/ccl_plan_test.go`.
- Docs: `Docs/parquet.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- Compatibility: `FilterWithCCL` gains a trailing variadic; existing calls are unchanged.
