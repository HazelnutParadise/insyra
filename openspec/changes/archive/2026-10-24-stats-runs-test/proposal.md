# Proposal: stats-runs-test

## Why

[#499](https://github.com/HazelnutParadise/insyra/issues/499), item 2: a weak-form efficiency study asks whether the order of up and down moves is random, and the Wald-Wolfowitz runs test is the standard check (Fama 1965 applies it to the sign of price changes). The owner approved the shape on 2026-10-24 after a Fable review.

## What Changes

- New `stats.RunsTest(series, opts ...RunsTestOptions)` returning a `RunsTestResult`: `Runs`, `Above`, `Below`, `Dropped`, `Expected`, `Variance`, `Z`, `PValue`.
- `RunsTestOptions` has `Threshold` (default 0) and `Correction` (a continuity correction of 0.5, off by default).
- Values equal to the threshold are dropped before runs are counted and reported in `Dropped`.
- Errors: a nil or empty list, a non-finite value or threshold, more than one options value, and no value on one side of the threshold.
- `RunsTestResult` has `String()` and `Show()`.
- Not in this change: the other items of #499.

## Capabilities

### New Capabilities

- `stats-runs-test`: the Wald-Wolfowitz runs test on a threshold.

## Impact

`stats/runs.go`, `stats/result_strings.go`, `stats/runs_test.go`, `Docs/stats.md`, both CHANGELOGs. Additive; no existing call changes.
