# Proposal: weights-take-float64

## Why

`DataList.WeightedMean(weights any)` and `WeightedMovingAverage(windowSize int, weights any)` take their weights as `any` and work out what they were given at run time through `ProcessData`: a slice of any element type, an array, a pointer to either, or a `DataList`. Anything else is logged as a warning and treated as an empty list of weights, which then fails the length check with a message about the length rather than the type. `RollingOptions.Weights`, which does the same job for `Rolling`, is a `[]float64`. Two signatures for one idea is finding D-10 of the API review ([#219](https://github.com/HazelnutParadise/insyra/issues/219)); the compiler can check a `[]float64`, and nothing is lost, because a weight has to be a number.

## What Changes

- **BREAKING**: `WeightedMean(weights []float64) float64` and `WeightedMovingAverage(windowSize int, weights []float64) *DataList`. A call that passes a `[]float64` compiles unchanged. A call that passes a `DataList`, a `[]int` or a `[]any` no longer compiles and has to convert first.
- The paths that only existed because a weight could be something other than a number go away: `WeightedMean` no longer skips an element because its weight could not be read, and `WeightedMovingAverage` no longer fails for a weight that is not numeric.
- Everything else is kept as it is: the length checks and their failure shape, skipping a non-numeric element in `WeightedMean`, `NaN` for an empty list or a zero total weight.
- `IDataList` declares the new signatures.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datalist-numeric-input`: adds "Weights are a float64 slice".
- `exported-functions`: "ProcessData reports failure as an error" loses the `WeightedMean(42)` half of its second scenario. `core-utils-cleanup` taught both methods to report weights `ProcessData` could not read; with a typed parameter that call no longer compiles, so the scenario keeps only `stats.Skewness`.

## Impact

- `datalist.go` (`WeightedMean`, `WeightedMovingAverage`), `interfaces.go`, `datalist_test.go` (the one test that passed a `DataList` of weights), a new test file for the signature and behaviour. `utils_test.go` loses `TestWeightedMethodsReportUnreadableWeights`, which `core-utils-cleanup` added and which passes `42`, `"ab"` and a `DataList` as weights.
- `Docs/DataList.md` (both sections and their examples), both changelogs, `api-review.md` (D-10), `delivery-status.md`.
- `skills/insyra/` is not changed: it teaches no signatures.
- Not in this change: whether `WeightedMovingAverage` should stay beside `Rolling(RollingOptions{Weights: ...}).Mean()`, which is #222 and `legacy-transforms-pinned`.
