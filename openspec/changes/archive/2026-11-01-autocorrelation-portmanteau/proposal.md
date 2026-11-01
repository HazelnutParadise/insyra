# Proposal: autocorrelation-portmanteau

## Why

[#499](https://github.com/HazelnutParadise/insyra/issues/499), Ljung-Box: `Autocorrelation` tests one lag at a time, while a weak-form efficiency study usually asks whether any of the first `m` autocorrelations is nonzero. The owner approved the design on 2026-11-01 after a Fable review: the cumulative statistics are fields of the per-lag result rather than a new function.

## What Changes

- `AutocorrelationResult` gains `Q` and `PQ`: the Ljung-Box (1978) statistic `n(n+2) * sum_{j<=Lag} Rho_j^2/(n-j)` and its chi-squared p-value on `Lag` degrees of freedom (no fitted-parameter adjustment), equal to R's `Box.test(type = "Ljung-Box", fitdf = 0)` and statsmodels' `acorr_ljungbox`.
- It gains `QRobust` and `PQRobust`: the sum of `TRobust_j^2` over lags 1 to `Lag`, the robust Box-Pierce statistic of Lobato, Nankervis and Savin (2001, IER 42(1)) as Escanciano and Lobato (2009, J. Econometrics 151(2)) write it, with its chi-squared p-value on `Lag` degrees of freedom.
- The documentation says that `QRobust` is Box-Pierce weighted, not Ljung-Box with a robust variance, and that the p-values assume an observed series, not model residuals.
- `Show()` prints the four new columns.

## Capabilities

### Modified Capabilities

- `stats-autocorrelation`: adds the cumulative Ljung-Box and robust portmanteau statistics.

## Impact

`stats/autocorrelation.go`, `stats/autocorrelation_test.go`, reference scripts and their output in `stats/testdata/`, `Docs/stats.md`, both CHANGELOGs. Additive: `Autocorrelation` is unreleased and every existing field keeps its value; only a struct literal of `AutocorrelationResult` written without field names would stop compiling.
