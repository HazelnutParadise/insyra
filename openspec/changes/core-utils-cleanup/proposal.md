# Proposal: core-utils-cleanup

## Why

The core package exports four helpers that have nothing to do with tables, and two of them misbehave (#214, K-15 and IN-17 in `api-review.md`). Measured on `0.4` at 43cb6519: `SqrtRat(big.NewRat(-1, 1))` panics with `square root of negative operand`, and `PowRat(big.NewRat(2, 3), -2)` returns `1` because the loop over the exponent runs zero times; `PowRat(nil, 2)` panics too. Nothing in the module calls `SqrtRat`, `PowRat` or `F64orRat`, and `SortTimes` has one caller, `mkt`, which `slices.SortFunc` replaces.

`ProcessData` returns `([]any, int)`, where the int is `len` of the slice and a failure is `nil, 0` plus a log line. A caller cannot tell an unreadable input from an empty slice, which breaks the rule that an ordinary function returns `(T, error)`. A typed nil `*DataList` crashes it, and through it `stats.Skewness` and `stats.Kurtosis`.

## What Changes

- **BREAKING**: `ProcessData(input any) ([]any, error)`. A type it cannot read, a nil input and a nil pointer are an error with a nil result; an empty slice is an empty, non-nil result. `WeightedMean`, `WeightedMovingAverage`, `stats.Skewness` and `stats.Kurtosis` report that error instead of a length mismatch, a crash or `empty data`. `WeightedMean` reads its weights before taking the list's lock, so a DataList of weights is no longer read unlocked inside another list's callback.
- `SqrtRat` returns nil for a nil or negative input instead of panicking. `PowRat` computes a negative power as the reciprocal, returns nil for a nil base and for zero to a negative power, and computes through `big.Int.Exp`.
- `SqrtRat`, `PowRat`, `SortTimes` and `F64orRat` are Deprecated, each naming its replacement, and are removed in the next release; an `AGENTS.md` follow-up records the removal. `mkt` sorts with `slices.SortFunc`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `exported-functions`: adds "ProcessData reports failure as an error" and "The big.Rat helpers never panic".

## Impact

- `utils.go`, `datalist.go`, `stats/skewness.go`, `stats/kurtosis.go`, `mkt/cai.go`; tests in `utils_test.go` and `stats/moment_input_test.go`.
- `Docs/utils.md`, `Docs/DataList.md` (the weights of `WeightedMean` and `WeightedMovingAverage`), `Docs/stats.md` (the input of `Skewness` and `Kurtosis`), both changelogs, `api-review.md`, `AGENTS.md`, `delivery-status.md`.
- The agent skills teach no helper named here and are unchanged.
