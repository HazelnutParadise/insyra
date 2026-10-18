# Proposal: chi-squared-p-value-never-panics

## Why

`chiSquaredPValue` hands its statistic straight to gonum, which panics with `cephes: parameter out of bounds` for a negative statistic or a non-positive df. A statistic built as a difference of logs or sums lands just below zero by rounding when the groups agree exactly, so `BartlettTest` on two identical groups, `KruskalWallis` on four all-zero groups and `FriedmanTest` on an all-zero table end the caller's goroutine with a panic. Measured on 2026-10-18; recorded in `AGENTS.md` since 2026-09-19.

## What Changes

- `chiSquaredPValue` returns 1 for a statistic at or below zero, and NaN for a NaN statistic, a NaN df or a df that is not positive.
- Every caller (`BartlettTest`, `KruskalWallis`, `FriedmanTest`, `ChiSquareTest`, `PartialCorrelation`, `FactorAnalysis`) gets a value instead of a panic. Results of calls that did not panic are unchanged.
- The `AGENTS.md` follow-up for this defect is removed.

## Capabilities

### Modified Capabilities

- `stats-test-input`: tests on degenerate-but-finite input return a result.

## Impact

`stats/distutil.go`, `stats/chisq_pvalue_guard_test.go`, `AGENTS.md`, both CHANGELOGs.
