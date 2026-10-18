# Proposal: stats-adjust-pvalues

## Why

[#444](https://github.com/HazelnutParadise/insyra/issues/444): testing many rules at once (44 main tests in a moving-average study) needs the p-values corrected together, and `stats` had nothing for it, so researchers wrote it by hand.

## What Changes

- `stats.AdjustPValues(pvals, method)` with `PAdjustHolm`, `PAdjustBonferroni`, `PAdjustHochberg`, `PAdjustBH` and `PAdjustBY`, matching R's `p.adjust`.
- A `NaN` is a test not run: it stays `NaN` and is not counted, as in R. Values outside [0, 1], infinities, non-numbers, nil and empty lists are errors. The method is an enum rather than the issue's string, following the library's enum convention (`String()` included).
- Hommel is not included; nothing asked for it.

## Capabilities

### New Capabilities

- `stats-multiple-comparisons`: correcting a batch of p-values.

## Impact

`stats/padjust.go`, tests with R reference values, `Docs/stats.md`, both CHANGELOGs.
