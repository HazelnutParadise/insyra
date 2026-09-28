# Design: weights-take-float64

## Decision

Take `[]float64`, the type `RollingOptions.Weights` already uses. Weights are numbers by definition, so the narrower type costs a caller nothing it could meaningfully pass, and it moves a whole class of mistakes (a `[]string`, a map, a struct) from a run-time length failure to a compile error.

## Alternatives considered

- **Keep `any` and document it.** Leaves two spellings of one idea and keeps `ProcessData`'s reflection on a hot numeric path.
- **Take `*DataList` or `IDataList`.** Weights are a parameter of the calculation, not a column of data; `Rolling` already settled on a slice.
- **Generic `[T Number]`.** Methods cannot have type parameters in Go, and it would still differ from `RollingOptions`.

## Behaviour kept

- Length mismatch: `WeightedMean` records the error and returns `NaN`; `WeightedMovingAverage` records it and returns an empty list carrying it.
- `WeightedMean` skips a non-numeric element together with its weight, and returns `NaN` for an empty list, for no numeric element, or for a zero total weight.
- A `NaN` weight is a number to both functions and propagates into the result, as it did through `ToFloat64Safe`.

## Migration

A caller holding weights in a `DataList` converts them before the call. The changelog says so; it does not recommend `ToF64Slice`, which turns an unreadable cell into `0` without saying so.
