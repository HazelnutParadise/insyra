# Proposal: abnormal-returns-finite-and-date-cell

## Why

Review of `quant.AbnormalReturns` found two faults. A ratio of two finite positive prices or market levels can overflow to +Inf, and the abnormal returns and CAR then carried an infinity with a nil error. A date cell that is a slice was flattened into several rows of the `CARs` table, so the table no longer had one row per event.

## What Changes

- A simple return that is not finite is missing data, so an event whose window needs it is dropped whole and counted in `Missing`.
- The event-date column of `CARs` keeps each date as one cell, whatever its type.

## Capabilities

### Modified Capabilities

- `quant-abnormal-returns`: two requirements added.

## Impact

`quant/abnormal_returns.go`, its test, `Docs/quant.md`. No changelog change: `AbnormalReturns` is unreleased and its entry does not describe either case.
