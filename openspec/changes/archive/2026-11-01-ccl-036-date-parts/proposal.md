## Why

The owner ruled that a replacement must exist on 0.3.6 before the old name is deprecated (#414). `DATEPART` and `DAYOFMONTH` could not warn, because `DAY`, `HOUR`, `MINUTE` and `SECOND` did not read dates on this line.

## What Changes

- `DAY`, `HOUR`, `MINUTE` and `SECOND` read a date as 0.4 does. A date string's result changes, and a `time.Time` no longer fails.
- A duration, a duration string and a number convert as before, with the warning pointing to `DUR`.
- `DATEPART` and `DAYOFMONTH` log one deprecation warning each.

## Impact

- `internal/ccl/stdlib_datetime.go`, `Docs/CCL.md`, both changelogs.
