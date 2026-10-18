# Proposal: stats-variance-ratio

## Why

[#443](https://github.com/HazelnutParadise/insyra/issues/443): testing whether returns trend or reverse at scales from days to months needs the Lo-MacKinlay variance ratio, and `stats` had none.

## What Changes

- `stats.VarianceRatio(returns, qs...)` returns VR(q) with the homoscedastic z and the heteroskedasticity-robust z* and their p-values; `stats.ChowDenning` is the joint test over several q with the Sidak correction.
- Checked against an independent loop-by-loop Python implementation of the definitions and by simulation (5% size for both statistics, rejection under momentum). R `vrtest` and Python `arch` were not installed, so no comparison against them was made.

## Capabilities

### New Capabilities

- `stats-variance-ratio`: the Lo-MacKinlay variance ratio and Chow-Denning tests.

## Impact

`stats/variance_ratio.go`, tests, `Docs/stats.md`, both CHANGELOGs.
