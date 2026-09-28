# Proposal: legacy-transforms-pinned

## Why

Finding D-14 of the API review ([#222](https://github.com/HazelnutParadise/insyra/issues/222)) lists six older `DataList` methods as duplicates of newer ones: `Difference` of `Diff(1)`, `FillNaNWithMean` of `FillWithMean`, `MovingAverage(w)` of `Rolling(RollingOptions{Window: w}).Mean()`, `MovingStdev(w)` of `Rolling(...).Std()`, `ExponentialSmoothing(a)` of `EWM(EWMOptions{Alpha: a}).Mean()`, and `WeightedMovingAverage` of `Rolling(RollingOptions{Weights: ...}).Mean()`. Under the owner's ruling on #211 a function has one name, and an old name stays one release as a Deprecated wrapper that keeps its meaning. That ruling covers two names for one function. The owner asked for each pair to be proved equivalent first, covering `NaN` and `nil`, edge windows, output length and the failure on bad input, and for only the equivalent pairs to be deprecated.

Read against the code, no pair is equivalent. On a fully numeric series inside the window each old method gives the same numbers as its replacement, but every pair differs in at least two of the respects the owner named:

| Old | Replacement | Differences |
| --- | --- | --- |
| `Difference()` | `Diff(1)` | length `n-1` against `n` with a leading `nil`; a `nil` operand gives `NaN` against `nil`; a non-numeric cell fails the call against giving `nil`; fewer than two values give an empty list against `[nil]` |
| `FillNaNWithMean()` | `FillWithMean()` | fills `NaN` only and leaves `nil`, against both; rewrites every other number as `float64`, against leaving it as it was |
| `MovingAverage(w)` | `Rolling{Window: w}.Mean()` | length `n-w+1` against `n`; a `nil` or non-numeric cell fails the call against `nil` for the windows holding it; `NaN` propagates against `nil`; `w > n` fails against a list of `nil` |
| `MovingStdev(w)` | `Rolling{Window: w}.Std()` | length `n-w+1` against `n`; a `nil` or non-numeric cell is skipped against `nil` for the window; `NaN` propagates against `nil`; `w = 1` gives `NaN` against `nil`; `w > n` fails against a list of `nil` |
| `ExponentialSmoothing(a)` | `EWM{Alpha: a}.Mean()` | `a = 0` is accepted against refused; a `nil`, `NaN` or non-numeric cell fails the call against carrying the last mean forward with decaying weight |
| `WeightedMovingAverage(w, ws)` | `Rolling{Window: w, Weights: ws}.Mean()` | length `n-w+1` against `n`; a `nil` or non-numeric cell fails against `nil`; `NaN` propagates against `nil`; weights summing to zero give `±Inf` or `NaN` against `nil`; `w > n` fails against a list of `nil` |

So these are different functions, not two names for one, and deprecating any of them would retire a behaviour its replacement does not offer. `FillNaNWithMean` has been Deprecated since v0.2.19 and keeps that status and its meaning; no other method in `insyra` fills `NaN` alone (the CLI's `fillna … missing nan` does it by restoring `nil` cells after a fill).

## What Changes

- A test pins each pair: the numbers agree on a fully numeric series, and each difference in the table holds. A later change that makes a pair equivalent, or breaks one of the old methods, fails it.
- `Docs/DataList.md` gets a table of the six pairs and their differences, and each old method's section links to it and states its own contract where it was missing (`MovingAverage`, `MovingStdev`, `WeightedMovingAverage` say nothing about missing values today, and `Difference` still says it returns `nil` on failure, which it stopped doing in `make-errors-non-terminating`).
- Nothing is deprecated or removed. Whether to retire the old methods anyway, with the differences as a migration note, is left to the owner on #222.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datalist-numeric-input`: adds "The older window and fill methods keep contracts of their own".

## Impact

- A new test file. No library code changes.
- `Docs/DataList.md`, `api-review.md` (D-14 gets a note, not a fix mark), `delivery-status.md`. No changelog entry: nothing a user sees changes.
