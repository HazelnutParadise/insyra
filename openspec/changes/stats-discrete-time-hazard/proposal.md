# Proposal: stats-discrete-time-hazard

## Why

[#443](https://github.com/HazelnutParadise/insyra/issues/443) asked, for the Dow-theory study, whether a bull or bear market is more likely to end the longer it has lasted. `VarianceRatio`, `ChowDenning` and `KaplanMeier` are done. The discrete-time hazard model, the third item, is not.

## What Changes

- `stats.DiscreteTimeHazard(durations, observed, opts)` fits the probability that a spell ends in each period given that it lasted until then, as a function of how long it has lasted and of covariates.
- It expands each spell into one row per period it was at risk, marks the last row 1 when the spell ended and 0 when it was censored, and fits a binomial GLM with a logit or complementary log-log link.
- The complementary log-log link, a `TODO` stub in `stats/glm_links.go`, is implemented as part of it.

## Capabilities

### New Capabilities

- `stats-discrete-time-hazard`: per-period ending probability of a spell.

## Impact

`stats/survival_hazard.go`, `stats/glm_links.go`, tests, `Docs/stats.md`, both CHANGELOGs.

## Also in this change

- `quant.Regimes` reports the last, unfinished stretch when `RegimeOptions.SeriesRows` is set, `Regime` gains `Ended`, and `quant.RegimeDurations` turns regimes into the two lists the model takes (see design.md).
- `stats.PersonPeriod` is public.
