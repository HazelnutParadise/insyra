## Why

One row that cannot be computed fails a whole CCL formula, and guarding every kind of failure with `IF` is long when the data is messy (#414).

## What Changes

- Add `IFERROR(value, valueIfError)`, which computes `valueIfError` only when `value` fails.
- A misspelt function or a missing column still fails, because that is a mistake in the formula.

## Impact

- `internal/ccl`, `Docs/CCL.md`, both changelogs.
