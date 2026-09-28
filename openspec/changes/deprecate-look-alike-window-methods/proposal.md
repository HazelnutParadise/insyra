# Proposal: deprecate-look-alike-window-methods

## Why

D-14 of the API review ([#222](https://github.com/HazelnutParadise/insyra/issues/222)) found six `DataList` methods overlapping six others. `legacy-transforms-pinned` showed that no pair is two names for one function: each differs in output length, `nil`/`NaN` handling, edge windows or what a bad input does, and a test pins every difference. The owner ruled on 2026-09-28 that four of them go through the one-name rule of #211 anyway, keeping their behaviour for one release rather than being changed into aliases, which would change their results under the same name without a compile error: `MovingAverage(3)` on five values would return five cells instead of three.

## What Changes

- `Difference`, `MovingAverage`, `MovingStdev` and `WeightedMovingAverage` are **Deprecated**, each pointing at `Diff(1)`, `Rolling(...).Mean()`, `Rolling(...).Std()` and `Rolling(RollingOptions{Weights: ...}).Mean()` and saying how the result differs. Their behaviour does not change until they are removed, in the release after the one that deprecates them.
- `ExponentialSmoothing` is not deprecated: `EWM` refuses `alpha = 0`, which it accepts. `FillNaNWithMean` keeps the Deprecated status it has had since v0.2.19.
- The CLI's `movavg` and `diff` keep calling the deprecated methods, so their output does not change, and are marked for the linter the way `fillnan` already is. What they become when the methods go is part of the removal follow-up.
- `Docs/DataList.md` marks the four and presents the difference table as their migration note.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datalist-numeric-input`: adds "Four look-alike window methods are deprecated and keep their meaning".

## Impact

- `datalist.go` (four doc comments), `interfaces.go` (the same four marked on `IDataList`, as `ToJSON_Bytes` is on `IDataTable`, so a call through the interface is flagged too), `cli/commands/timeseries.go` (two lint markers), the two characterization test files (their header comments name the renamed docs section), a new test that the four doc comments carry the deprecation.
- `Docs/DataList.md`, both changelogs, `AGENTS.md` (removal follow-up), `api-review.md` (D-14), `delivery-status.md`.
