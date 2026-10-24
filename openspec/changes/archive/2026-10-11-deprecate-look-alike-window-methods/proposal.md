# Proposal: deprecate-look-alike-window-methods

## Why

D-14 of the API review ([#222](https://github.com/HazelnutParadise/insyra/issues/222)) found six `DataList` methods overlapping six others. `legacy-transforms-pinned` showed that no pair is two names for one function: each differs in output length, `nil`/`NaN` handling, edge windows or what a bad input does, and a test pins every difference. The owner ruled on 2026-09-28 that four of them go through the one-name rule of #211 anyway, keeping their behaviour rather than being changed into aliases.

## What Changes

- `Difference`, `MovingAverage`, `MovingStdev` and `WeightedMovingAverage` are **Deprecated**, each pointing at `Diff(1)`, `Rolling(...).Mean()`, `Rolling(...).Std()` and `Rolling(RollingOptions{Weights: ...}).Mean()` and saying how the result differs, and noting removal in 0.4. Their behaviour does not change.
- `ExponentialSmoothing` is not deprecated: `EWM` refuses `alpha = 0`, which it accepts. `FillNaNWithMean` keeps the Deprecated status it has had since v0.2.19.
- The CLI's `movavg` and `diff` keep calling the deprecated methods, so their output does not change.
- `Docs/DataList.md` marks the four and presents the difference table as their migration note.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datalist-numeric-input`: adds "Four look-alike window methods are deprecated and keep their meaning".

## Impact

- `datalist.go`, `interfaces.go`, `cli/commands/timeseries.go`, test files.
- `Docs/DataList.md`, both changelogs, `delivery-status.md`.
