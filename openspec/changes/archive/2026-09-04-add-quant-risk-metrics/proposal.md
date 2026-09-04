# Proposal: add-quant-risk-metrics

## Why

`quant` measures return (`SharpeRatio`, `AnnualizedReturn`) and one drawdown number (`MaxDrawdown`) but has no tail-risk measure and no downside-aware ratio. A risk desk's daily numbers are Value at Risk and its conditional form; a research desk reports Sortino, Calmar, and information ratio next to Sharpe. All of these are a few lines over primitives the package already has, and their absence is the first thing a finance user notices.

## What Changes

- New `quant.VaRMethod` (`VaRHistorical`, `VaRParametric`) and `quant.ValueAtRisk(returns insyra.IDataList, confidence float64, method VaRMethod) (float64, error)`. VaR is reported as a **positive loss fraction** at the given confidence (0.95 means the 5th percentile of returns, negated). Historical uses the R type-7 quantile `PercentileBands` already carries; parametric uses `mean + z·sd` with `z = stats.NormPPF(1−confidence)` and the sample standard deviation.
- New `quant.ConditionalValueAtRisk(returns, confidence, method) (float64, error)`: historical is the negated mean of returns at or below the VaR quantile; parametric is `−(mean − sd·φ(z)/(1−confidence))`.
- New `quant.SortinoRatio(returns insyra.IDataList, minimumAcceptableReturn, periodsPerYear float64) (float64, error)`: annualized `mean(r − MAR) / downsideDeviation`, where downside deviation is the root mean square of `min(r − MAR, 0)` over **all** periods (the Sortino & van der Meer definition), not only the losing ones.
- New `quant.CalmarRatio(equity insyra.IDataList, days int) (float64, error)`: `AnnualizedReturn / MaxDrawdown`, composed from the existing functions; a zero drawdown is an error (ratio undefined).
- New `quant.InformationRatio(returns, benchmark insyra.IDataList, periodsPerYear float64) (float64, error)`: annualized mean of active return divided by tracking error (sample standard deviation of active return); zero tracking error is an error.
- New `quant.DrawdownSeries(equity insyra.IDataList) (*insyra.DataList, error)`: per-period drawdown from the running peak as non-negative fractions, the series `MaxDrawdown` takes the maximum of.
- All new functions read input through `numericSeries` and refuse unreadable, NaN, or Inf cells with the row named. Confidence must lie in `(0, 1)`; `periodsPerYear` must be positive.
- Docs (`Docs/quant.md`, README rows), `skills/insyra`, and both changelogs are updated in the same change.

## Capabilities

### New Capabilities

- `quant-risk-metrics`: value at risk, conditional value at risk, Sortino, Calmar, information ratio, and the drawdown series of a return or equity series.

### Modified Capabilities

(none)

## Impact

- New files `quant/risk.go`, `quant/risk_test.go`. Reuses `numericSeries`, `quantileType7`, `maxDrawdownF64`, `annualizedReturnF64`, and `stats.NormPPF`/`NormCDF` (already imported by `quant/overfitting.go`).
- `Docs/quant.md` (overview bullet, new "Risk Metrics" section, usage example, error rows), `Docs/README.md`, `README.md`, `README_TW.md`, `skills/insyra/SKILL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `quant/init.go` package comment.
- No new dependencies, no breaking changes. Existing `quant` functions and their `ToF64Slice` use are untouched (standing follow-up).
