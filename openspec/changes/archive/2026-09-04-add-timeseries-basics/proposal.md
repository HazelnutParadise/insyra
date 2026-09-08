# Proposal: add-timeseries-basics

## Why

Three operations that every return-series workflow reaches for are missing from the core: exponentially weighted statistics (the standard volatility estimator), rolling covariance and beta (time-varying exposure), and time-based resampling (daily bars to weekly or monthly). Today `Rolling` stops at `Corr`, nothing in the repository applies exponential decay, and turning daily OHLC into monthly bars requires a hand-written `GroupBy` over a truncated date column. The risk-metric, factor-model, and Taiwan-data changes that follow all consume these primitives, so they go first.

## What Changes

- New `DataList.EWM(opts EWMOptions) *EWMDataList` with reducers `Mean()`, `Var()`, `Std()`. `EWMOptions` takes exactly one of `Alpha`, `Span`, `HalfLife` (no default), `Adjust bool` (pandas `adjust`), `Bias bool` (pandas `bias`, false means the unbiased variance), and `MinObs int`. Semantics follow pandas `Series.ewm` exactly and are pinned by pandas-generated fixtures in the existing cross-language corpus. Invalid options warn and return an empty result, the same contract as `Rolling`.
- New `RollingDataList.Cov(other *DataList) *DataList` (sample covariance, n−1) and `RollingDataList.Beta(other *DataList) *DataList` (`Cov(src, other) / Var(other)`; the receiver is the asset, `other` the benchmark). Alignment, nil handling, `MinObs`, and truncation to the shorter series follow `Corr`. Windows whose benchmark variance is zero emit nil.
- Matching `DataTable.EWMCol(col string, opts EWMOptions) *EWMDataList` alongside the existing `RollingCol`/`ExpandingCol`.
- New `DataTable.Resample(timeCol string, freq ResampleFreq, aggs ...ResampleAgg) (*DataTable, error)`. `ResampleFreq` is `ResampleWeekly`, `ResampleMonthly`, `ResampleQuarterly`, `ResampleYearly`. Each `ResampleAgg{Col, Op AggregateOp, As string}` reuses the existing `GroupBy` operators, so OHLC is `OpFirst`/`OpMax`/`OpMin`/`OpLast` and volume is `OpSum`. The output has one row per non-empty period, labelled by the period's last calendar day in the time column, in ascending order; empty periods are not fabricated. `timeCol` must hold `time.Time` values.
- Docs (`Docs/DataList.md`, `Docs/DataTable.md`), the `skills/insyra` skill, and both changelogs are updated in the same change.

## Capabilities

### New Capabilities

- `datalist-ewm`: exponentially weighted mean, variance, and standard deviation on a `DataList`, pandas-compatible.
- `datalist-rolling-cov-beta`: rolling sample covariance and rolling beta between two `DataList`s.
- `datatable-resample`: time-based resampling of a `DataTable` to weekly, monthly, quarterly, or yearly periods with per-column aggregation.

### Modified Capabilities

(none — existing rolling reducers are unchanged)

## Impact

- New files `datalist_ewm.go`, `datalist_ewm_test.go`, `datatable_resample.go`, `datatable_resample_test.go`; `datalist_window.go` gains `Cov` and `Beta`; `datatable_window.go` gains `EWMCol`; `interfaces.go` gains the new methods on `IDataList`/`IDataTable`.
- Cross-language fixtures: new pandas-generated cases for `ewm` in the corpus `datalist_window_crosslang_test.go` reads, produced by the same generator.
- `Docs/DataList.md`, `Docs/DataTable.md`, `skills/insyra/SKILL.md`, `skills/insyra/references/` where window functions are listed, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- No new dependencies, no breaking changes. CLI/DSL exposure is not part of this change.
