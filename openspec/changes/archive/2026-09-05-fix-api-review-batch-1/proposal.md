# Proposal: fix-api-review-batch-1

## Why

The production-readiness review in `api-review.md` (core foundations, DataList, csvxl, parallel, parquet) found eight behaviours that are wrong rather than merely inconsistent, two of them confirmed by execution: `ReplaceLast(5, 0)` on `[5, NaN]` rewrites the NaN and leaves the 5, and `Normalize()` on `[1, "x", 3]` returns nil after already overwriting the first cell with `0`. The rest are the same family the 2026-08-01 `stats` decision closed — a value that cannot be read is turned into `0` and the answer looks plausible — plus a documented guard that does not exist (`parquet.ReadColumnOptions.MaxValues`), an integer-precision split between two JSON entry points, and three `excelize` handles that are never closed. None of these are design questions; they are fixed here so the design decisions still open in the review (LogFatal, the interfaces, nil returns) can be taken on a correct base.

## What Changes

- `DataList.ReplaceLast` replaces the last occurrence of `oldValue` only; a trailing NaN is no longer touched when `oldValue` is not NaN.
- `DataList.Normalize`, `Standardize`, `ClearOutliers`, `Difference`, and `FillNaNWithMean` scan the whole list before writing anything. A cell that is neither numeric, nil, nor NaN makes the call fail with `Err()` set and the data untouched; nil and NaN cells pass through where they are. The return shape of each function on failure is unchanged (this change does not decide review item D-3).
- `DataList.Rank`, `ExponentialSmoothing`, `DoubleExponentialSmoothing`, and the six `*Interpolation` methods no longer read through `ToF64Slice`. A non-numeric cell fails the call instead of participating as `0`. `Rank` gives nil and NaN cells a NaN rank (pandas `na_option="keep"`); the smoothing and interpolation methods require a fully numeric series.
- `stats.Skewness` and `stats.Kurtosis` read input through the package's `numericValues` and return an error naming the row of a non-numeric or non-finite cell, matching every other `stats` entry point.
- `ReadJSON_File` decodes through the same number-preserving path as `ReadJSON`, so an integer literal loads as `int64` from a file exactly as it does from bytes.
- `csvxl.AppendCsvToExcel` replaces an existing sheet of the same name in full, as its documentation already promises; `AppendCsvToExcel`, `ExcelToCsv`, and `EachExcelToCsv` close every `excelize.File` they open.
- `parquet.ReadColumn` honours `ReadColumnOptions.MaxValues`: the row count of the selected row groups is checked from metadata before any value is read, and an error is returned when it exceeds the limit.
- **BREAKING (behavioural)**: inputs that were silently coerced to `0`, silently skipped, or half-processed now fail. Fully numeric inputs produce results bit-identical to before. Marked in both changelogs the way past release notes mark breaking changes.
- Docs, both changelogs, and the `AGENTS.md` `ToF64Slice` follow-up (interpolation callers removed from its list) are updated in the same change. `api-review.md` items D-1, D-2, D-4 (High part), K-2, K-9, C-3, C-4, Q-1 are marked fixed.

## Capabilities

### New Capabilities

- `datalist-value-replacement`: `ReplaceFirst`/`ReplaceLast`/`ReplaceAll` replace only cells equal to `oldValue`, with NaN matched only when `oldValue` is NaN.
- `datalist-numeric-input`: the numeric transforms and reducers named above refuse unreadable cells before writing, never substitute `0`, and leave the list untouched on failure.
- `stats-moment-input`: `Skewness` and `Kurtosis` refuse non-numeric and non-finite input naming the row.
- `json-read-numbers`: `ReadJSON_File` and `ReadJSON` type integer literals identically.
- `csvxl-excel-append`: appending a sheet whose name exists replaces it entirely; opened workbooks are closed.
- `parquet-read-column-limit`: `ReadColumn` enforces `MaxValues` from metadata before reading.

### Modified Capabilities

(none — no existing spec covers these functions)

## Impact

- `datalist.go` (ReplaceLast, Normalize, Standardize, ClearOutliers, Difference, FillNaNWithMean, Rank, ExponentialSmoothing, DoubleExponentialSmoothing), `datalist_interpolation.go`, a new `datalist_numeric.go` helper, `read.go` (ReadJSON_File), `stats/skewness.go`, `stats/kurtosis.go`, `csvxl/convert.go`, `csvxl/convertDir.go`, `parquet/api.go`, and their tests.
- `Docs/DataList.md`, `Docs/DataTable.md`, `Docs/stats.md`, `Docs/csvxl.md`, `Docs/parquet.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `AGENTS.md`, `api-review.md`.
- No new dependencies. `ToF64Slice` and `SliceToF64` themselves are not changed; their remaining callers (`plot`, `gplot`, `cli`) stay as the follow-up records. The CLI `rank`, `normalize`, `skewness`, `kurtosis` commands inherit the new refusals through the library; their command surface is unchanged.
