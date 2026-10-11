# Design: legacy-transforms-pinned

## What "equivalent" was tested against

The owner's list: `NaN` handling, `nil` handling, edge windows (`w = 1`, `w = n`, `w > n`, `w = 0`), output length, and what a bad input does. A pair counts as equivalent only if the old call's output, for every input, equals the replacement's output after a fixed, documented adjustment (dropping the leading `w-1` positions), including which calls fail.

## Why pin the differences in a test rather than only document them

The table in the proposal was derived by reading the code; a test that asserts each row turns it into evidence, and keeps the documentation honest if either side changes. Where the two agree, the test compares numbers exactly for the rolling mean, the weighted mean and the difference, which run the same arithmetic in the same order, and within `1e-12` for the standard deviation and exponential smoothing, whose formulas differ in operation order.

## Recommendation left for the owner

The replacements are the better-designed set (pandas-compatible, same-length output, `nil` rather than failure for a gap) and the old set is where D-3 and D-4 concentrated. Retiring the old ones is still a decision to drop behaviour: a caller relying on `MovingAverage` failing on a gap, or on its shorter output, would silently get something else from a mechanical rewrite. The recommendation on #222 is to deprecate `Difference`, `MovingAverage`, `MovingStdev` and `WeightedMovingAverage` with the migration note this change writes, keep `ExponentialSmoothing` until `EWM` accepts or deliberately refuses `alpha = 0` with a documented reason, and remove `FillNaNWithMean` only if no `NaN`-only fill is wanted in the library.
