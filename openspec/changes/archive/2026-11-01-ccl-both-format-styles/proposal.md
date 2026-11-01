## Why

`TEXT` and `FORMAT_DATE` only read Go's formats. An Excel format errors under `TEXT` and comes back as its own text under `FORMAT_DATE`: `FORMAT_DATE(d, 'yyyy-mm-dd')` gives the string `yyyy-mm-dd`. The owner ruled on 2026-11-01 that both spellings work (#414).

## What Changes

- `TEXT` and `FORMAT_DATE` read a Go format or an Excel one, told apart from the format alone.
- A date format with no readable code is an error, where `FORMAT_DATE` echoed it.

## Impact

- `internal/ccl/`, `Docs/CCL.md`, both changelogs, `ccl-type-semantics` spec.
