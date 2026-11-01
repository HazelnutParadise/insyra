## Why

Four CCL functions had a second name doing the same thing (#414, the one-name rule of #211). The owner ruled on 2026-11-01 that CCL keeps the shorter name, and that names are deprecated in v0.3.6 and removed in 0.4.

## What Changes

- **BREAKING** `TONUM`, `TOSTR`, `SUBSTR` and `DAYOFMONTH` are removed in favour of `VALUE`, `TEXT`, `MID` and `DAY`. A formula still using one gets an error naming what to write.

## Impact

- `internal/ccl/`, tests, `Docs/CCL.md`, both changelogs, `ccl-type-semantics` spec.
