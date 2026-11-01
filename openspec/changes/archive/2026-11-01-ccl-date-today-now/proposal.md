## Why

CCL can read and shift a date but cannot build one from its parts or ask what day it is, which leaves common spreadsheet formulas such as days since a date with no direct form (#414).

## What Changes

- Add `DATE(year, month, day)`, `TODAY()` and `NOW()`.
- `TODAY` and `NOW` read one moment per call, across every row, statement and batch.
- They hold the local clock's date and time with no zone, as CCL holds a date read from text without one.

## Impact

- `internal/ccl`, `ccl.go`, `datatable_ccl.go`, `parquet/ccl.go`, `Docs/CCL.md`, both changelogs.
