# Proposal: ccl-datepart

## Why

CCL's `HOUR`, `MINUTE` and `SECOND` convert durations, and since `ccl-duration-functions-refuse-dates` a date passed to them is an error. That error pointed to `TONUM(FORMAT_DATE(x, '15'))`, `'04'` and `'05'`, Go reference-layout digits that the owner found unintuitive on 2026-10-04 and asked to fix first. CCL had no function that gives the hour, minute or second of a date.

## What Changes

- Add `DATEPART(d, unit)`. It returns one part of `d` as a number: `year`, `month`, `day`, `hour`, `minute` or `second`, singular or plural, in any case, read in `d`'s own time zone. A fraction of a second is dropped, as Excel's `SECOND` drops it. An unknown unit, a value that is not a date, or the wrong number of arguments is an error.
- The name and argument order complete CCL's existing `DATEADD(d, n, unit)` and `DATEDIFF(d1, d2, unit)`. `DATEADD`, `DATEDIFF` and `DATEPART` are SQL Server's trio (PostgreSQL calls it `date_part`). Like YEAR, MONTH and DAYOFMONTH, the result is a `float64`.
- The errors from `HOUR`, `MINUTE` and `SECOND` on a date now name `DATEPART(x, 'hour')`, `'minute'` and `'second'`.
- `FORMAT_DATE` keeps its Go layout. Moving it to Excel format codes is a breaking change that belongs to the function review in #414.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ccl-type-semantics`: `DATEPART`; the duration functions' errors name it.

## Impact

- `internal/ccl/stdlib_datetime.go` (`DATEPART`) and `internal/ccl/stdlib.go` (the hints in `durationIn`).
- Not breaking: a new function, and a different suggestion in an error message.
- `Docs/CCL.md`, both changelogs (a new entry, and the `DAY`/`HOUR` entry's hint), `api-review.md` CCL-22, `delivery-status.md`.
