# Proposal: stats-discrete-time-hazard

## Why

[#443](https://github.com/HazelnutParadise/insyra/issues/443)'s last item: a discrete-time hazard model, to ask whether a spell (a bull market, say) is more likely to end the longer it has lasted. The design of PR #479 (2026-10-19) was brought up to date with the current `quant/ta` and `stats` code, re-reviewed by Fable, and approved by the owner on 2026-10-31.

## What Changes

- `stats.DiscreteTimeHazard(durations, observed, opts ...DiscreteTimeHazardOptions)` fits Allison's person-period binomial GLM with a `log t` (default), linear or one-level-per-period baseline and optional constant covariates, returning named terms, the `Duration` term, the whole GLM and `String`/`Show`.
- `stats.PersonPeriod` returns the person-period table, for models the function does not offer.
- `GLM` accepts the `Cloglog` link for the binomial family, as R's `binomial("cloglog")`.
- `ta.RegimeDurations` turns `Regimes` into durations and ended flags, adding each segment's still-going last stretch as a censored spell, without changing `Regimes` or `Regime`.
