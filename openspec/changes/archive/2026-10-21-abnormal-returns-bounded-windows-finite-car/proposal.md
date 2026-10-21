# Proposal: abnormal-returns-bounded-windows-finite-car

## Why

Review of `quant.AbnormalReturns` found that an ordered `EventWindow` such as `[minInt, 0]` made the window length overflow and the allocation panic, and that a running CAR of finite abnormal returns could sum to an infinity with a nil error.

## What Changes

- Each end of `EventWindow` and `EstimationWindow` must be within 2^29 rows of the event, the table must have at most 2^29 rows, and `EventWindow` must not span more rows than the table has. Otherwise an error is returned before anything is allocated. The fit loop is bounded by the table's rows.
- An event whose abnormal return or CAR is not finite, or whose fit is not finite, is dropped whole and counted in `Missing`, like a non-finite return.
- A cross-sectional mean or spread whose sum overflows gives NaN.

## Capabilities

### Modified Capabilities

- `quant-abnormal-returns`: two requirements added.

## Impact

`quant/abnormal_returns.go`, its test, `Docs/quant.md`. No changelog change: the feature is unreleased and its entry does not describe these cases.
