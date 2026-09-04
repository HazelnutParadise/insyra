# Proposal: add-quant-beta-capm

## Why

The first thing anyone evaluating a single stock asks is "how much does it move with the market?" — its beta, and the alpha left over. Every piece needed to answer that already exists in Insyra (`datafetch` for prices, `DataTable.Merge` for date alignment, `PctChangeCol` for returns, `stats.LinearRegression` for the slope), but a user has to chain five steps, discover that `stats` refuses the leading `nil` that `PctChange` emits, and read the slope back out of a general regression result. Verified on 2026-09-03 against a synthetic price pair: the composition works, the slope equals covariance ÷ variance, and nothing in `quant` names the result. `quant` evaluates strategies (`SharpeRatio`, `MaxDrawdown`, `PBO`, `WalkForward`, `BlockBootstrap`) but has no measure of exposure to a benchmark at all.

## What Changes

- New `quant.Beta(asset, market insyra.IDataList) (float64, error)`: the market beta of a per-period return series against a benchmark return series, `Cov(asset, market) / Var(market)`. One call for the question users actually ask.
- New `quant.CAPM(asset, market insyra.IDataList, riskFreeRate float64) (*CAPMResult, error)`: the single-index market model fitted by OLS on excess returns (`asset − rf` on `market − rf`). `CAPMResult` carries `Beta`, `Alpha` (the per-period intercept — Jensen's alpha), `RSquared`, `BetaStdErr`, `AlphaStdErr`, and `N`. `riskFreeRate` is per period, the same convention `SharpeRatio` uses; pass 0 for raw-return regression.
- Inputs are *aligned per-period returns*, not prices. Both series must have the same length; the package does not align by date, convert prices, or drop cells. Unreadable, `NaN`, or `Inf` values are refused with the row named, through the `numericSeries` helper `BlockBootstrap` introduced — `ToF64Slice` is not used.
- Validation, not defaults: nil input, length mismatch, fewer than 3 observations (no residual degree of freedom), or a benchmark with zero variance (beta undefined) return an error naming the condition.
- `Beta` and `CAPM(...).Beta` agree exactly for any `riskFreeRate`, because subtracting one constant from both series changes neither the covariance nor the variance; a test asserts it.
- Docs (`Docs/quant.md`, `Docs/README.md`, both README package rows), the `skills/insyra` skill, and both changelogs are updated in the same change. The docs carry the alignment recipe (two price tables → `Merge` on date → `PctChangeCol` → `ClearNils`) and the choices that move the number: `Close` versus `Adj Close`, window length, return frequency.

## Capabilities

### New Capabilities

- `quant-capm`: market beta and the single-index CAPM regression (beta, alpha, R², standard errors) of an asset's return series against a benchmark's, computed from aligned per-period returns.

### Modified Capabilities

(none — no existing spec-level requirement changes)

## Impact

- New files `quant/capm.go`, `quant/capm_test.go`. Reuses `numericSeries` from `quant/bootstrap.go`; no new dependencies (`gonum/stat` is already imported by `quant`).
- `quant` does not import `stats`; the test file does, to assert agreement with `stats.LinearRegression`.
- `Docs/quant.md` (overview bullet, new "Market Exposure (CAPM)" section, usage example with the alignment recipe, error-handling rows, related-packages note), `Docs/README.md`, `README.md`, `README_TW.md` (quant row), `skills/insyra/SKILL.md` (quant section), `CHANGELOG.md`, `CHANGELOG_TW.md`, `quant/init.go` package comment.
- No breaking changes, no CLI/DSL surface (`quant` has none today).
- Out of scope, tracked as candidate follow-on changes: rolling covariance / rolling beta on `RollingDataList`, exponentially weighted statistics, and Sortino / Calmar / information ratio.
