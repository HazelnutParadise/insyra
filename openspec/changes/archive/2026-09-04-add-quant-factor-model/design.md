# Design: add-quant-factor-model

## Context

`stats.LinearRegression(dlY, dlXs...)` returns `Coefficients` (intercept first), `StandardErrors`, `TValues`, `PValues`, `ConfidenceIntervals`, `Residuals`, `RSquared`, `AdjustedRSquared`, R-verified, and refuses unreadable input with the row named. `quant` already imports `stats` (`overfitting.go`), so the dependency exists. `quant/capm.go` regresses excess asset on excess market with a closed form; `quant/bootstrap.go` has `numericSeries`. `DataTable` exposes column names and `GetColByNumber`.

## Goals / Non-Goals

**Goals:**
- Named-factor attribution with the same inference `stats` provides, in finance vocabulary.
- One-factor consistency with `CAPM`.
- Errors that name the factor column, not "predictor 2".

**Non-Goals:**
- Building factor returns (SMB/HML portfolio construction), rolling factor exposures, Newey–West standard errors, factor risk decomposition (variance attribution). Each is a separate change.
- Re-implementing OLS; `CAPM` did so to stay dependency-free, but multivariate inference belongs in `stats`.

## Decisions

### Wrap `stats.LinearRegression`, keep the numbers identical

The value of this change is the entry point, not new arithmetic. `FactorModel` converts inputs, subtracts `rf` from the asset, calls `stats.LinearRegression(excess, factorCols...)`, and relabels the result. A test asserts field-by-field equality so the wrapper can never disagree with the R-verified core.

### Factor columns are taken as given

Fama–French factors are published as excess (`Mkt−RF`) or long–short (`SMB`, `HML`) series; subtracting `rf` from them would be wrong for all but a raw market index. The function therefore subtracts `rf` from the asset only, and the doc comment tells the caller to pass `market − rf` when they build the market factor themselves. The one-factor CAPM agreement test uses exactly that construction.

### Validation before conversion, errors name the factor

Length checks run before cell conversion (the `gatherRegressionInputs` precedent), and each factor column is converted through `numericSeries(col, name)` so an unreadable cell reports the column name. The minimum `n = k + 2` leaves one residual degree of freedom, matching `CAPM`'s `n >= 3` at `k = 1`.

### Result indexed by `FactorNames` plus a lookup method

Slices indexed like `FactorNames` keep the result cheap and ordered; `Exposure(name)` gives the common single lookup without a map allocation per call.

## Risks / Trade-offs

- [Users pass raw index returns as the market factor] → doc comment and docs example show `market − rf`; the CAPM agreement scenario encodes the convention.
- [`stats` error text changes] → the wrapper passes errors through unchanged; tests assert on the presence of an error, not its wording, for the collinear case.
- [Heteroskedastic or autocorrelated residuals] → OLS standard errors are what `stats` gives; documented as a limitation with Newey–West named as future work.

## Open Questions

None that block implementation.
