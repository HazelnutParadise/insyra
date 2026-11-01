## Why

Counting or adding up the rows that meet a condition takes `SUM(IF(A > 0, 1, 0))`. Excel users reach for `COUNTIF`, `SUMIF` and `AVERAGEIF`, whose conditions are written as text. In CCL quoted text is the text itself, and a condition is already an expression, so a second condition syntax would add nothing (#414).

## What Changes

- Add `COUNTIF(condition)`, `SUMIF(condition, values)` and `AVGIF(condition, values)`, taking the condition as an expression.
- `FilterWithCCL` and `ApplyCCL` compute them by holding the columns they read, as they do `MEDIAN`.

## Impact

- `internal/ccl`, `Docs/CCL.md`, `Docs/parquet.md`, both changelogs.
