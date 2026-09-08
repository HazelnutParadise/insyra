# Proposal: add-quant-factor-model

## Why

`CAPM` explains an asset with one factor. Research desks attribute returns to several — market, size, value, momentum, industry — and need the exposure to each with its standard error, the alpha that remains, and how much variance the factors explain. `stats.LinearRegression` already fits the multiple regression and is R-verified; what is missing is the finance-facing entry point that names factors, applies the excess-return convention, and returns exposures keyed by factor name instead of by coefficient position.

## What Changes

- New `quant.FactorModel(asset insyra.IDataList, factors insyra.IDataTable, riskFreeRate float64) (*FactorModelResult, error)`: OLS of `asset − riskFreeRate` on every column of `factors`, all series aligned per period. Factor columns are taken as given (Fama–French style factors are already excess or long–short; the caller subtracts `rf` from a raw market factor before passing it, and the doc says so).
- `FactorModelResult` carries `Alpha`, `AlphaStdErr`, `AlphaTValue`, `AlphaPValue`, `FactorNames []string` (column names in table order), `Exposures`, `StdErrs`, `TValues`, `PValues` (all indexed like `FactorNames`), `RSquared`, `AdjustedRSquared`, `N`, and `Residuals []float64`. `Exposure(name string) (float64, bool)` looks one up by factor name.
- Inputs are validated: nil, zero factor columns, length mismatch between the asset and any factor column, fewer than `k + 2` observations (no residual degree of freedom), and unreadable/NaN/Inf cells (named by series and row) are errors. A collinear factor set is reported as the error `stats` raises rather than a silent near-singular fit.
- A one-factor call agrees with `CAPM` on beta, alpha, and standard errors; a test asserts it.
- Docs (`Docs/quant.md`, README rows), `skills/insyra`, and both changelogs are updated in the same change.

## Capabilities

### New Capabilities

- `quant-factor-model`: multi-factor OLS attribution of an asset's excess returns with named exposures, alpha, inference, and fit statistics.

### Modified Capabilities

(none)

## Impact

- New files `quant/factor.go`, `quant/factor_test.go`. `quant` already imports `stats` (`overfitting.go`), so wrapping `stats.LinearRegression` adds no dependency.
- `Docs/quant.md` (overview bullet, "Factor Models" section, usage example, error rows), `Docs/README.md`, `README.md`, `README_TW.md`, `skills/insyra/SKILL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `quant/init.go`.
- No breaking changes.
