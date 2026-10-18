# Proposal: stats-newey-west-mean-test

## Why

[#457](https://github.com/HazelnutParadise/insyra/issues/457): testing whether the mean of an autocorrelated series (overlapping returns) is zero with an ordinary t-test overstates t several times over. The standard remedy is the Newey-West standard error, and `stats` had none.

## What Changes

- `stats.NeweyWestMeanTest(series, lag)` returns the mean, HAC standard error, t, two-tailed p (t distribution, n-1 degrees of freedom), lag and n. A negative lag uses `NeweyWestLag(n)`, `floor(4(n/100)^(2/9))`. Lag 0 equals the one-sample t-test with an n divisor.
- Not changed: HAC standard errors for `FactorModel` and `LinearRegression`, which the issue names as a later change. The automatic lag is the Newey-West (1994) rule of thumb, not their data-driven plug-in.
- Reference values were computed with the same formula written in R; the R `sandwich` package was not installed, so no comparison against it was made.

## Capabilities

### New Capabilities

- `stats-newey-west`: the Newey-West mean test.

## Impact

`stats/neweywest.go`, tests, `Docs/stats.md`, both CHANGELOGs.
