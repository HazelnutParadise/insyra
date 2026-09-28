# Proposal: gplot-refuses-partial-charts

## Why

`chart-constructors-return-errors` (#253) gave every `gplot` constructor an error to return, but kept one partial outcome from before: a line, step or scatter chart that could draw some of its series and not others returned the chart with a nil error and named the missing series only in a logged warning. A caller who asked for two lines and got one had no signal in the return value. The rest of `gplot` already treats a series it cannot use as an error: a scatter series whose `X` and `Y` differ in length, and a bar chart or histogram holding a `NaN`, both fail the call. Reviewing the new API after #253, the owner agreed that a length mismatch should fail the whole chart, as matplotlib's `plot(x, y)` raises when `x` and `y` differ in length.

## What Changes

- **BREAKING (behaviour)**: `gplot.CreateLineChart`, `CreateStepChart` and `CreateScatterPlot` return a nil chart and an error when any series cannot be drawn: a line or step series whose length differs from `XAxis`, an empty series, or a series holding a `NaN` or an infinity. The error names every such series and why. Nothing is logged for them. Before, such a series was skipped with a warning and the chart came back with the others.
- A nil list among real ones in a line or step chart is still dropped with a warning, as `plot` drops one; that rule belongs to #413, which covers what the two packages do with missing input.
- `Docs/gplot.md`, the three constructors' doc comments and the unreleased changelog entries describe the new rule.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `chart-constructors`: "Every chart constructor returns the chart and an error" no longer lists a dropped series among the outcomes that stay a warning; "A gplot chart that can draw no series returns an error" is replaced by "A gplot chart that cannot draw every series returns an error".

## Impact

- `gplot/line.go`, `gplot/step.go`, `gplot/scatter.go`; tests in `gplot/chart_errors_test.go` and `gplot/charts_test.go`.
- `Docs/gplot.md`, `CHANGELOG.md`, `CHANGELOG_TW.md` (the unreleased `gplot` entry that described the skip is rewritten, since neither behaviour has shipped), `delivery-status.md`.
- `skills/insyra/` is not changed: it teaches no chart behaviour at this level.
