# Proposal: stats-autocorrelation

## Why

[#499](https://github.com/HazelnutParadise/insyra/issues/499), item 1: weak-form efficiency tests need the autocorrelation of a return series by lag, with a test that stays valid when volatility clusters. `VarianceRatio` and `ChowDenning` exist; the autocorrelation they are built from is not offered on its own. The owner approved the shape on 2026-10-24 after a Fable review.

## What Changes

- New `stats.Autocorrelation(series, maxLag)` returning one `AutocorrelationResult` per lag from 1 to `maxLag`: `Rho`, a white-noise test (`T`, `P`) and a heteroskedasticity-robust test (`TRobust`, `PRobust`, Lo and MacKinlay 1988).
- `Rho` follows R's `acf` (deviations from the mean, divisor the sum of squares).
- Errors: a nil or empty list, a non-finite value, a constant series, `maxLag` outside 1 to n-1, and a lag whose robust variance is zero.
- `AutocorrelationResult` has `String()` and `Show()` like every `stats` result.
- Not in this change: Ljung-Box, the runs test, the rank variance ratio, the filter rule and the momentum portfolios, which are the other items of #499 and each get their own change.

## Capabilities

### New Capabilities

- `stats-autocorrelation`: the autocorrelation by lag with a white-noise and a robust test.

## Impact

`stats/autocorrelation.go`, `stats/result_strings.go`, `stats/autocorrelation_test.go`, `Docs/stats.md`, both CHANGELOGs. Additive; no existing call changes.
