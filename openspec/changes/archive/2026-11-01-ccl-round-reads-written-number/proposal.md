## Why

`ROUND(1.005, 2)` gives 1 because the float64 closest to 1.005 lies just below it, while Excel gives 1.01 and so does `TEXT(1.005, '0.00')`. One number should not have two rounded answers (#414).

## What Changes

- `ROUND` rounds half away from zero on the number's shortest decimal text.
- A number on such a boundary now gets a different, larger-magnitude result. Other numbers are unchanged.

## Impact

- `internal/ccl/stdlib_math.go`, `Docs/CCL.md`, both changelogs.
