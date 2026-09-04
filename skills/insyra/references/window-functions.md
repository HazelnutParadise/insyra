# Time-series windows and resampling

## DataList

```go
rolling := dl.Rolling(insyra.RollingOptions{Window: 5, MinObs: 3})
rolling.Cov(other)
rolling.Beta(other)

ewm := dl.EWM(insyra.EWMOptions{Alpha: 0.5, Adjust: false})
ewm.Mean()
ewm.Var()
ewm.Std()
```

Rolling paired reducers align by index, skip nil or non-numeric pairs, use
sample covariance and variance, and return nil for fewer than two valid pairs.
`Beta` treats the receiver as the asset and `other` as the benchmark. A flat
benchmark has undefined beta and returns nil. `other == nil` returns an empty
result and records a warning.

`EWMOptions` requires exactly one decay parameter:

- `Alpha`: `(0, 1]`
- `Span`: `>= 1`, converted to `2/(span+1)`
- `HalfLife`: `> 0`, converted to `1-exp(ln(0.5)/halflife)`

`Adjust` selects pandas' adjusted or recursive weighting, `Bias` selects the
population or finite-sample weighted variance form, and `MinObs <= 0` means one
valid observation. Gaps decay the state without resetting it. Invalid options
warn and return an empty result. `DataTable.EWMCol` resolves the column by name
first, then Excel-style index.

## DataTable

```go
monthly, err := dt.Resample("Date", insyra.ResampleMonthly,
    insyra.ResampleAgg{Col: "Open", Op: insyra.OpFirst},
    insyra.ResampleAgg{Col: "High", Op: insyra.OpMax},
    insyra.ResampleAgg{Col: "Low", Op: insyra.OpMin},
    insyra.ResampleAgg{Col: "Close", Op: insyra.OpLast},
    insyra.ResampleAgg{Col: "Volume", Op: insyra.OpSum},
)
```

`ResampleWeekly` uses Monday-Sunday buckets. Monthly, quarterly, and yearly
groups use calendar boundaries. Each non-empty group is labelled with its
period's final calendar day at midnight, retains the input time location, and
is sorted by the time column. Empty periods are not fabricated. `ResampleAgg`
uses `AggregateOp`; `As` defaults to `Col`. Missing columns, no aggregations,
unknown frequencies, and non-`time.Time` values return errors.
