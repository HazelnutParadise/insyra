# Proposal: gplot-refuses-bad-settings

## Why

After `gplot-refuses-partial-charts`, two `gplot` settings still gave the caller something other than what was asked for, with a nil error and a logged warning:

- `CreateStepChart` replaced an unknown `StepStyle` with `"post"`, so a misspelled `"pr"` drew post steps. The CLI stopped doing this in v0.4's unreleased work ("a misspelled option value is rejected instead of falling back to the default"), and the library should behave the same.
- `CreateBarChart` dropped `ErrorBars` whose length differed from the data, or which held a `NaN` or an infinity, and drew the bars without them. The chart looked complete while missing what was requested.

The owner agreed to make both errors on 2026-09-28.

## What Changes

- **BREAKING (behaviour)**: `gplot.CreateStepChart` returns a nil chart and an error naming the value when `StepStyle` is not `"pre"`, `"mid"`, `"post"` or empty.
- **BREAKING (behaviour)**: `gplot.CreateBarChart` returns a nil chart and an error when `ErrorBars` is given and its length differs from the data's, or it holds a `NaN` or an infinity.
- After this, the only outcome `gplot` still reports as a warning is a nil list among real ones in a line or step chart, which #413 covers.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `chart-constructors`: "Every chart constructor returns the chart and an error" no longer lists error bars of the wrong length among the outcomes that stay a warning; a new requirement, "gplot refuses a setting it cannot honour", covers `StepStyle` and `ErrorBars`.

## Impact

- `gplot/bar.go`, `gplot/step.go`; tests in `gplot/chart_errors_test.go`, `gplot/charts_test.go`, `gplot/step_test.go`.
- `Docs/gplot.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`.
