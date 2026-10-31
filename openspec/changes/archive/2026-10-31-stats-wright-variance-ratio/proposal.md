# Proposal: stats-wright-variance-ratio

## Why

[#499](https://github.com/HazelnutParadise/insyra/issues/499) asks for the weak-form efficiency toolkit. Robust autocorrelation, the runs test and the Lo-MacKinlay variance ratio are merged; Wright's (2000) rank and sign variance ratios are the last item. They ask the same question as `VarianceRatio` without assuming normal, homoscedastic returns, and their p-values are exact for any sample size. The owner approved the design on 2026-10-31 after a Fable review, and asked for the multi-period results to print as one table rather than by hand.

## What Changes

- `stats.WrightVarianceRatio(returns, qs []int, opts ...WrightVarianceRatioOptions) (WrightVarianceRatioResults, error)` returns `R1`, `R2`, `S1`, their simulated two-sided p-values and `Zeros` for each holding period. `WrightVarianceRatioOptions{Replications, Seed}`: 10,000 draws by default, and the seed always applies, zero included, as `quant.BootstrapConfig.Seed`.
- `VarianceRatio` and `Autocorrelation` return the named slice types `VarianceRatioResults` and `AutocorrelationResults`, whose `Show()` prints one table. A named slice is assignable to the plain slice, so existing callers keep compiling.
