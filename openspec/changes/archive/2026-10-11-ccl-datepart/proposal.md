# Proposal: ccl-datepart

## Why

CCL's `HOUR`, `MINUTE` and `SECOND` convert durations; a date string reaches them as the time since that date's own midnight. CCL had no function that gives the hour, minute or second of a date.

## What Changes

- Add `DATEPART(d, unit)`. It returns one part of `d` as a number: `year`, `month`, `day`, `hour`, `minute` or `second`, singular or plural, in any case, read in `d`'s own time zone. A fraction of a second is dropped, as Excel's `SECOND` drops it. An unknown unit, a value that is not a date, or the wrong number of arguments is an error.
- The name and argument order complete CCL's existing `DATEADD(d, n, unit)` and `DATEDIFF(d1, d2, unit)`. `DATEADD`, `DATEDIFF` and `DATEPART` are SQL Server's trio (PostgreSQL calls it `date_part`). Like `YEAR`, `MONTH` and `DAYOFMONTH`, the result is a `float64`.

Left on 0.4: `DAY`, `HOUR`, `MINUTE` and `SECOND` refusing a date, and their errors naming `DATEPART`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ccl-type-semantics`: `DATEPART`.

## Impact

- `internal/ccl/stdlib_datetime.go` (`DATEPART`).
- Not breaking: a new function.
- `Docs/CCL.md`, both changelogs (a new entry).
