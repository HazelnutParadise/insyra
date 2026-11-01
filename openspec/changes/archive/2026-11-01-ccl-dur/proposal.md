## Why

`DAY`, `HOUR`, `MINUTE` and `SECOND` used to convert a duration to a number of units and now read a date. `DATEDIFF` covers two dates and dividing covers a `time.Duration`, but a duration string such as `'36h'` has no way to become a number, and the error those functions give points to `DATEDIFF`, which does not read one.

## What Changes

- New `DUR(x, unit)`: `x` is a `time.Duration` or a duration string, with `d` and `w` accepted as well as Go's units; `unit` is one of `DATEDIFF`'s and is required. An empty cell stays empty. A number and unreadable text are errors naming the value.
- The error `DAY`, `HOUR`, `MINUTE` and `SECOND` give for a duration also names `DUR`.
- Those functions say an empty cell is empty, where they said it was a number.

## Impact

- `internal/ccl/`, `Docs/CCL.md`, both changelogs, `ccl-type-semantics` spec.
