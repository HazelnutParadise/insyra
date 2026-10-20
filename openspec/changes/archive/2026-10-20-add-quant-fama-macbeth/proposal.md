# Proposal: add-quant-fama-macbeth

## Why

[#478](https://github.com/HazelnutParadise/insyra/issues/478), part 2: `QuantileSort` tests one signal at a time, so it cannot say whether a characteristic still explains next-period returns once the others are controlled for. The standard tool is the Fama-MacBeth regression: one cross-sectional regression per date, then a t-test on the time series of each coefficient. Research code writes it by hand in Python today.

## What Changes

- `quant.FamaMacBeth(table, dateCol, returnCol, charCols, opts)` takes the same long table as `QuantileSort` (one row per stock and date), regresses the return on an intercept and the characteristics each date, and tests the time average of every coefficient with `stats.NeweyWestMeanTest`.
- `FamaMacBethOptions`: `NeweyWestLag` (0 = automatic, positive = as given, negative = error), `NoNeweyWest` (switches the correction off, lag 0; an error together with a non-zero `NeweyWestLag`), `MinObservations` (default and floor k+2), `Standardize` (per-day z-scores of the characteristics).
- `FamaMacBethResult`: the intercept and each characteristic's mean coefficient, standard error, t and p value, the per-day table, the mean R², the lag used, and the skipped days. A day with too few stocks or a collinear cross-section is skipped and listed, never an error for the whole call.
- The column lookup `QuantileSort` uses (name first, then Excel letter) moves into a helper both functions share; `QuantileSort`'s behaviour does not change.

## Capabilities

### New Capabilities

- `quant-fama-macbeth`: Fama-MacBeth cross-sectional regressions on a long panel.

## Impact

`quant/fama_macbeth.go`, `quant/quantile_sort.go` (helper only), tests and a `linearmodels` baseline under `quant/testdata`, `Docs/quant.md`, both CHANGELOGs.
