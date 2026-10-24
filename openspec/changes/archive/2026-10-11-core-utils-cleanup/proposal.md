# Proposal: core-utils-cleanup

## Why

The core package exports four helpers that have nothing to do with tables, and two of them misbehave (#214, K-15 and IN-17 in `api-review.md`). Measured before the fix: `SqrtRat(big.NewRat(-1, 1))` panicked with `square root of negative operand`, and `PowRat(big.NewRat(2, 3), -2)` returned `1` because the loop over the exponent ran zero times; `PowRat(nil, 2)` panicked too.

## What Changes

- `SqrtRat` returns nil for a nil or negative input instead of panicking. `PowRat` computes a negative power as the reciprocal, returns nil for a nil base and for zero to a negative power, and computes through `big.Int.Exp`.
- `SqrtRat`, `PowRat`, `SortTimes` and `F64orRat` are Deprecated, each naming its replacement, and are removed in 0.4.
- Left on 0.4: `ProcessData` signature change (`([]any, error)`). On this line `ProcessData` keeps its original `([]any, int)` signature.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `exported-functions`: adds "The big.Rat helpers never panic".

## Impact

- `utils.go`, `mkt/cai.go`; tests in `utils_test.go`.
- `Docs/utils.md`, both changelogs, `delivery-status.md`.
