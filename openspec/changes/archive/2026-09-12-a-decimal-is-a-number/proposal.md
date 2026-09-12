# Proposal: a-decimal-is-a-number

## Why

Two places in the library put a fixed-point decimal in a cell — `finance.ScheduleTable` and a Parquet `Decimal128` column — and the numeric path cannot read either. Measured:

```
IsNumeric(decimal)  = false
ToFloat64Safe       = 0, false
ScheduleTable column Mean() = NaN, Err() = nil
```

So `Mean`, `Sum`, `Describe` and every `stats` entry point silently answer `NaN` on a column of money. That is FI-2 of [#247](https://github.com/HazelnutParadise/insyra/issues/247), and the owner has ruled that a decimal is a number.

Sorting is inconsistent with the same ruling. A decimal sits at sorting rank 5 while numbers are rank 2 and strings are 3, so a mixed column comes out `1, 3, 2.50, 10.00` — the decimals grouped at the end rather than interleaved by value.

## What Changes

- `ToFloat64` and `ToFloat64Safe` read a decimal, and `IsNumeric` agrees. The two have to move together: `fix-clear-defects-core` fixed exactly this split for named numeric types, and reopening it would mean a value that is a number to one half of the library and unreadable to the other.
- A decimal sorts among numbers. Two decimals still compare through `decimal.Cmp`, which is exact; a decimal against a float compares as floats, like any two numbers of different width.
- Conversion goes through the value's own text. A `float64` holds about 16 significant digits, so a decimal wider than that is rounded — which is what asking for float arithmetic means, and the cell keeps its exact value either way.

**Matched by shape, not by type.** `internal/utils` is on the path of every value in the library and must not depend on one decimal package; that dependency is FI-1's complaint about `finance` and spreading it into core would make it worse. A value is treated as a decimal when it can report both its own text and its scale, and when that text parses as a number — two conditions together, so a type that merely happens to have those methods is not swept in. Any decimal library of that shape works, and a `Float64()` method upstream would be cleaner still.

## Capabilities

### New Capabilities

- `decimal-cells`: what the library does with a fixed-point decimal in a cell.

### Modified Capabilities

(none)

## Impact

- `Mean`, `Sum`, `Describe`, `stats` and every other numeric path start reading decimal columns instead of answering `NaN`. `finance.ScheduleTable`'s output becomes usable without converting it by hand.
- A mixed column of decimals and floats sorts by value rather than in two blocks. A column of only decimals sorts as it did, exactly, through `decimal.Cmp`.
- `IsNumeric` returns true for a decimal, which a caller may be branching on.
- Nothing about identity, counting or grouping changes: a decimal is still not usable as a map key, because `big.Int` holds a slice.
