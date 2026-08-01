## Context
`projectValues` turns a `[]any` column from `insyra.DataList` into a typed buffer. It runs three passes: `inferDataType` decides the column type, a conversion loop fills the typed slice and the nulls slice, and `buildValidityBitmap` walks the nulls again.

Every one of those passes reaches for `reflect` per element:

| Call | What it does per element |
| --- | --- |
| `isInt` / `isFloat` | `reflect.TypeOf(v).Kind()`, twice per element during inference |
| `toFloat64` | `reflect.ValueOf(v)`, then `rv.Convert(reflect.TypeOf(float64(0))).Float()` |
| `toInt64` | `reflect.ValueOf(v)`, then `rv.Int()` or `rv.Uint()` |

`reflect.Value.Convert` allocates. Converting a `float64` to `float64` goes through `reflect.cvtFloat` → `reflect.makeFloat` → `reflect.unsafe_New` → `mallocgc`, so a 4 Mi column performs 4 Mi heap allocations to produce values it already had.

Profile, Apple M3, 4 Mi `float64` column:

```
94 ms/op   354 MB/s   71,827,492 B/op   4,194,308 allocs/op

23.4%  accel.toFloat64
 21.0%   reflect.Value.Convert
 18.6%    reflect.cvtFloat
 17.7%     reflect.makeFloat
 16.1%      reflect.unsafe_New
10.5%  accel.inferDataType
 6.5%  accel.buildValidityBitmap
```

## Goals / Non-Goals
- Goals:
  - remove per-element allocation from projection
  - keep every value that projects today projecting to the same result
- Non-Goals:
  - changing `Buffer`, `DataType`, or what `projectValues` returns
  - changing how `insyra.DataList` stores values; that is a core change with a far wider blast radius
  - parallelising projection, or caching projections across calls

## Decisions

- Decision: dispatch on concrete types with a type switch, and keep `reflect` only as a fallback.
  - Rationale: a type switch over concrete types is an interface-type comparison with no allocation. It cannot match a named type such as `type Celsius float64`, though, and `reflect.TypeOf(v).Kind()` can — so today those columns project and a naive type switch would silently start rejecting them. Falling through to the existing reflect path preserves that exactly, at the cost of one failed type switch for the rare case.
  - This is the whole fix. The reflect calls were never needed for the common path; they were a general mechanism applied uniformly to a loop that runs millions of times.

- Decision: collapse `inferDataType`'s predicate chain into one type switch per element.
  - Rationale: it currently calls `isBool`, then `isInt`, then `isFloat`, then `isString`, and the middle two each build a `reflect.Type`. One switch answers the same question once.

- Decision: leave `buildValidityBitmap` alone until the numbers say otherwise.
  - Rationale: it is 6.5% of a profile that reflection dominates. Folding it into the conversion loop means either duplicating bitmap arithmetic across five branches or introducing a helper called per element. Neither is worth doing on a guess. Task 1.6 re-measures first and only then decides — if it does become material after reflection is gone, the fill-with-`0xFF`-and-clear-on-null approach keeps the work proportional to the number of nulls rather than to the column length, with the trailing padding bits masked so the output stays byte-identical.

## Measured

Apple M3, 4 Mi `float64` column (32 MB).

| | before | after reflect removal | after bitmap fold |
| --- | --- | --- | --- |
| `BenchmarkProjectValues` | 94.0 ms, 354 MB/s | 20.6 ms, 1628 MB/s | **13.0 ms, 2587 MB/s** |
| allocations | 4,194,308/op | 4/op | 4/op |
| bytes allocated | 71.8 MB/op | 38.3 MB/op | 38.3 MB/op |

Removing `reflect` took the per-element allocations to zero. That left `buildValidityBitmap` as the largest single node at 20.7% of the profile, which met task 1.6's condition, so it was folded into the projection loop as designed: the bitmap starts all-ones and only nulls clear a bit, with the trailing padding masked so the bytes stay identical.

Downstream, over the same column:

| | original | after `fix-accel-fingerprint-cost` | after this change |
| --- | --- | --- | --- |
| `BenchmarkProjectionOnly` (`ProjectDataList`) | 357 ms | 145 ms | **43 ms** |
| `BenchmarkColumnSum/4Mi/gpu` end to end | 354 ms | 117 ms | **47.6 ms** |
| device total for the same column | 4.6 ms | 8.3 ms | 5.6 ms |

`BenchmarkProjectValues` improved 7.2x, not the full order of magnitude the delivery plan asked for, but `ProjectDataList` is 8.3x off its original figure and host cost is now within roughly eight times the device cost rather than seventy. What remains splits three ways at about 13 ms of projection, 14 ms of fingerprinting, and the rest in allocation and cache bookkeeping — no single dominant cost is left to remove.

## Risks / Trade-offs
- Risk: a type switch that misses a case would silently change a column's inferred type rather than erroring.
  - Mitigation: the reflect fallback is the default arm, so a missed case falls through to the code that runs today instead of to a wrong answer. Tests cover named types, mixed int/float, and unconvertible values.
- Risk: `uint64` values above `MaxInt64` wrap when converted to `int64`.
  - Not a change: `reflect`'s `int64(rv.Uint())` wraps identically. Covered by a test so it stays that way deliberately rather than accidentally.
