# Proposal: stats-robust-regression-se

## Why

[#478](https://github.com/HazelnutParadise/insyra/issues/478), part 1: a factor regression of daily strategy returns has heteroskedastic, autocorrelated residuals, so the ordinary OLS standard error overstates the t value of alpha. `stats.NeweyWestMeanTest` covers only the mean; `LinearRegression` and `quant.FactorModel` have no robust or HAC standard errors, and `Docs/quant.md` says so.

## What Changes

- `stats.CovType` (`CovOLS`, the zero value and today's behaviour; `CovHC0`, `CovHC1`, `CovHC2`, `CovHC3`; `CovHAC`) with `String()`.
- `stats.LinearRegressionOptions{CovType, Lag}` and `stats.LinearRegressionWithOptions(opts, y, xs...)`, mirroring `LogisticRegressionWithOptions`. `LinearRegression` becomes the zero-options call and returns the same numbers bit for bit. `LinearRegressionResult` records `CovType` and the `Lag` used.
- HC0–HC3 follow R `sandwich::vcovHC`. HAC is `(X'X)^-1 S (X'X)^-1` with Bartlett weights `1 - j/(lag+1)` and no small-sample factor, matching `sandwich::NeweyWest(lag = L, prewhite = FALSE, adjust = FALSE)`. A negative lag picks `NeweyWestLag(n)`, as `NeweyWestMeanTest` does; HAC with lag 0 equals HC0 exactly. t, p and confidence intervals use the t distribution with n-k degrees of freedom, as `NeweyWestMeanTest` and `lmtest::coeftest` do.
- `quant.FactorModel` gains an optional `FactorModelOptions{CovType, Lag}`. Without it the result is unchanged. `Lag` follows the quant convention: 0 picks `stats.NeweyWestLag(n)`, a positive value is used as it is, a negative value is refused. `FactorModelResult` records `CovType` and `Lag`.
- Not changed: the CLI `factor-model` command, `WeightedLinearRegression`, and the other regression families.

## Capabilities

### Modified Capabilities

- `stats-regression`: robust and HAC standard errors for ordinary least squares.
- `quant-factor-model`: the factor model can request them.

## Impact

`stats/regression.go`, `stats/olsutil.go`, `stats/enum_names.go`, `quant/factor.go`, tests (including a statsmodels reference comparison in `stats/testdata/crosslang_baseline.py`), `Docs/stats.md`, `Docs/quant.md`, both CHANGELOGs. `LinearRegressionResult.String()` gains the `CovType` and `Lag` lines.
