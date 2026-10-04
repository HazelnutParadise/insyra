# Proposal: ccl-date-part-functions-like-excel

## Why

CCL's `DAY`, `HOUR`, `MINUTE` and `SECOND` convert a duration to days, hours, minutes or seconds. Every tool CCL users come from gives those names another meaning: a part of a date. That holds for Excel, Google Sheets, LibreOffice, Power BI's DAX and Spark SQL; MySQL's `HOUR` on a time value is the one variant. So a formula carried over from Excel runs and answers something else.

The conversion also duplicates what CCL already has. `DAY(A - B)` and `DATEDIFF(A, B, 'day')` run the same formula, and the difference itself counts seconds, so `(A - B) / 3600` gives hours.

The names come from 90c9b052 (2026-01-09), which added date subtraction and named its converters after Excel's date-part functions. b050bcb4 (2026-05-08) then added `DATEDIFF`, and `DAYOFMONTH` because `DAY` was taken, working around the clash instead of removing it. On 2026-10-04 the owner decided to follow the industry. This completes #359 (CCL-22): `ccl-duration-functions-refuse-dates` made a date an error, and `ccl-datepart` added `DATEPART`.

## What Changes

- **BREAKING**: `DAY`, `HOUR`, `MINUTE` and `SECOND` take a `time.Time` or a date string and return its day of the month, hour (0–23), minute or second, read in the date's own time zone, as a `float64` like `YEAR` and `MONTH`. A fraction of a second is dropped, as in Excel.
- **BREAKING**: a `time.Duration`, a duration string or a number is an error. The error says the function takes a date and names `DATEDIFF(end, start, unit)`, or dividing the difference. A number is not read as an Excel serial date. `DAY('36h')` loses its conversion; `DATEDIFF` and arithmetic cover every duration that comes from two dates.
- `DAYOFMONTH` is Deprecated in the docs and keeps working as the same function as `DAY`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ccl-type-semantics`: "Duration functions refuse a date" is replaced by "DAY, HOUR, MINUTE and SECOND take a part of a date".

## Impact

- `internal/ccl/stdlib_datetime.go`: `datePart`, shared with `DATEPART`, and `datePartFunction`. `internal/ccl/stdlib.go`: the duration converters and `durationIn` are removed.
- Callers of `DAY(A - B)` and its siblings get an error naming `DATEDIFF`, never a different number.
- Tests: `duration_functions_test.go` is replaced by `date_part_functions_test.go`; `datatable_test.go` computes differences with `DATEDIFF`; `portable_integer_args_test.go` drops the duration-conversion cases, which now fail as not being dates.
- `Docs/CCL.md`, both changelogs (the `DAY` entry and the `DATEPART` entry describe the final behaviour), `api-review.md` CCL-22, `delivery-status.md`.
