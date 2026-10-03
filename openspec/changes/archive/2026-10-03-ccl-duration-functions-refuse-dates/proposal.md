# Proposal: ccl-duration-functions-refuse-dates

## Why

CCL's `DAY`, `HOUR`, `MINUTE` and `SECOND` convert a duration to days, hours, minutes or seconds. Excel's functions of the same names take a part of a date, and CCL is meant to read like Excel, so a formula carried over from Excel runs and answers something else. A date reaching these functions was read as the time since midnight. Measured on 2026-10-03: `DAY('2024-01-02T06:00:00Z')` gave `0.25` where Excel gives `2`, `HOUR('2024-01-02T06:30:00Z')` gave `6.5`, `MINUTE('2024-01-02')` gave `0`, and a `time.Time` cell failed with `unsupported type for DAY: time.Time`, which does not say what to use instead. None of it was documented. Issue #359 (CCL-22).

## What Changes

- **BREAKING**: a date string or a `time.Time` value passed to `DAY`, `HOUR`, `MINUTE` or `SECOND` is an error. The error says the function converts a duration and names the function for the date part: `DAYOFMONTH(x)`, `TONUM(FORMAT_DATE(x, '15'))`, `TONUM(FORMAT_DATE(x, '04'))` or `TONUM(FORMAT_DATE(x, '05'))`.
- A `time.Duration`, a duration string such as `'36h'` and a number of seconds convert exactly as before, to the last bit.
- `Docs/CCL.md` gains a "Differences from Excel" table: the functions that share an Excel name and behave differently, and the CCL names for Excel functions that are spelled differently. Every row was checked against CCL on 2026-10-03.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ccl-type-semantics`: the duration functions refuse a date.

## Impact

- `internal/ccl/stdlib.go`: the four registrations share one builder, `durationIn`.
- A formula that relied on `DAY(date)` meaning the fraction of the day elapsed now fails with an error that names the replacement; nothing gives a different value silently.
- `Docs/CCL.md`, `skills/insyra/SKILL.md` (one sentence on CCL not being Excel), both changelogs, `api-review.md`, `delivery-status.md`.
