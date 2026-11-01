## Why

`DATEPART` was added while `HOUR`, `MINUTE` and `SECOND` converted durations, so the hour of a date had no short form. Those functions read dates now, so `DATEPART` only repeats them and `YEAR`, `MONTH` and `DAY` under a longer name. The owner ruled on 2026-11-01 that it is deprecated in v0.3.6 and removed in 0.4.

## What Changes

- **BREAKING** `DATEPART` is removed. A formula still using it gets an error naming the functions to use.

## Impact

- `internal/ccl/`, `Docs/CCL.md`, both changelogs, `ccl-type-semantics` spec.
