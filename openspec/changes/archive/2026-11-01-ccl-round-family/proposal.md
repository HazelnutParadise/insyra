## Why

CCL has `ROUND`, `FLOOR`, `CEIL` and `TRUNC`, but nothing that rounds away from or toward zero at a given number of decimal places, and the owner asked for Excel's `INT`, `ROUNDUP` and `ROUNDDOWN` (#414).

## What Changes

- Add `INT(x)`, `ROUNDUP(x, n)` and `ROUNDDOWN(x, n)`, with the behaviour on Microsoft's function pages. `n` is required in both, as there.
- `INT` gives what `FLOOR` gives. The owner chose to have it despite that.
- All read the number as written, as `ROUND` does.

## Impact

- `internal/ccl/stdlib_math.go`, `Docs/CCL.md`, both changelogs.
