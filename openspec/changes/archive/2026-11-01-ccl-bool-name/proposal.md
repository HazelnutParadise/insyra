## Why

`VALUE` and `TEXT` carry no `TO`, and CCL prefers short names, so `TOBOOL` is the odd one out (#414).

## What Changes

- **BREAKING** on 0.4: `TOBOOL` becomes `BOOL`, and a formula using the old name gets an error naming the new one.
- On 0.3.6: `BOOL` is added, and `TOBOOL` keeps working, deprecated, with one warning.

## Impact

- `internal/ccl/stdlib_typeconv.go`, `Docs/CCL.md`, both changelogs.
