## Why

CCL has `ISNA` and no other way to ask what a cell holds, so a formula cannot branch on an empty cell, a word or a date without a workaround (#414).

## What Changes

- Add `ISBLANK`, `ISNUM`, `ISTEXT`, `ISBOOL`, `ISDATE` and `ISERROR`.
- `ISBLANK` counts `nil` and empty text; `NaN` stays with `ISNA`. `ISDATE` counts text that reads as a date.

## Impact

- `internal/ccl`, `Docs/CCL.md`, both changelogs.
