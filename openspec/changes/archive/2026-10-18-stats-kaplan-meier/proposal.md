# Proposal: stats-kaplan-meier

## Why

[#443](https://github.com/HazelnutParadise/insyra/issues/443): asking whether a bull or bear market is more likely to end the longer it has lasted needs a survival curve that handles the market still running, and `stats` had none.

## What Changes

- `stats.KaplanMeier(durations, observed)` returns the survival curve at each event time with the risk set, events, censored counts, Greenwood standard errors and the median.
- The values equal R `survival::survfit` on the same data, including the median at an exact 0.5.
- Not done from the issue: the discrete-time hazard model (logit or cloglog on person-period data), which stays open.

## Capabilities

### New Capabilities

- `stats-kaplan-meier`: the Kaplan-Meier survival curve.

## Impact

`stats/survival.go`, tests, `Docs/stats.md`, both CHANGELOGs.
