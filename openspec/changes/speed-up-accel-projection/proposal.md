# Change: Project columns without reflection

## Why
`projectValues` dispatches on type through `reflect` once per element. `toFloat64` calls `reflect.ValueOf(v)` and then `rv.Convert(reflect.TypeOf(float64(0)))`, and `Convert` heap-allocates a fresh value for every single element — including for a `float64` being "converted" to `float64`. `inferDataType` adds two more `reflect.TypeOf(v).Kind()` calls per element through `isInt` and `isFloat`.

Profiled on an Apple M3 over a 4 Mi `float64` column: `BenchmarkProjectValues` is 94 ms with **4,194,308 allocations and 71.8 MB allocated** for 32 MB of data. `reflect.Value.Convert` accounts for 21% of samples, its allocation path (`makeFloat` → `unsafe_New` → `mallocgc`) for another 18%, and `inferDataType` for 10%.

This is the dominant remaining cost on the accel path. `ProjectDataList` costs 145 ms against 8 ms of device work, so every end-to-end acceleration figure the runtime reports is really a measurement of reflection.

## What Changes
- Replace per-element `reflect` dispatch with concrete type switches in `toFloat64`, `toInt64`, `isInt`, and `isFloat`
- Keep a `reflect` fallback for named types such as `type Celsius float64`, so nothing that projects today stops projecting
- Fold type inference into a single type switch per element instead of a chain of separate predicate calls
- Add benchmarks that record allocations per element, so a regression back into reflection is visible

## Impact
- Affected specs: `accel-memory-cache`
- Affected code: `accel/dataset.go`
