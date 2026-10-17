# Proposal: datalist-quantile

## Why

D-13 and QU-3 of the API review (#221), ported from `0.4`: `Percentile` takes 0..100, `Quartile` takes 1..3, and nothing takes the 0..1 scale that pandas, R and gonum use. The owner ruled on 2026-10-15 to add `Quantile(p)` and keep the other two.

## What Changes

- New `DataList.Quantile(p float64) float64`, `p` from 0 to 1, R's type-7 quantile, the definition `Percentile` and `Quartile` already use. `nil` and `NaN` cells are skipped. A `p` that is NaN or outside `[0, 1]`, a cell that is neither a number nor missing, or a list with no numeric values records a warning and returns `NaN`.
- `IDataList` gains `Quantile`.
- New CLI command `quantile <var> <p>`.
- **Left on `0.4`:** `Percentile`, `Quartile`, `IQR` and `Median` skipping NaN, and `Percentile(NaN)` being refused, which change results those methods give today. On this line `Quantile` and `Percentile` agree except when the list holds a NaN or a cell that is not a number.

## Capabilities

### New Capabilities

- `datalist-quantile`: the 0..1 quantile.

### Modified Capabilities

None.

## Impact

- `datalist.go`, `interfaces.go`, `cli/commands/stats_dl_extra.go`, `Docs/DataList.md`, `Docs/cli-dsl.md`, both changelogs, and tests.
