# Proposal: print-a-key-as-the-value-writes-itself

## Why

`UncomparableKey.String` renders the encoded content, which for a struct is its fields. A `decimal.Decimal` — what `finance.ScheduleTable` puts in cells and what a Parquet `Decimal128` column reads as — therefore prints in a counter as

```
decimal.Decimal({{b:1,[i:3400221114815]},i:10}):3
```

That is `big.Int`'s sign and words, and the scale, for a value that knows perfectly well how to write itself as `-340.0221114815`.

The encoded content cannot be replaced by `String()`, because identity has to be exact and a `String()` is allowed to be lossy: two distinct values whose text matched would silently merge into one count, which is the failure this capability exists to prevent.

## What Changes

- `ToMapKey` asks the value whether it implements `fmt.Stringer`, and keeps the answer alongside the encoding. `String` prefers it, truncated by the same limit, and falls back to the encoded content when there is none.
- A value with no `String()` — a `[]byte`, a `[]int`, a map — renders exactly as it does now.

The display is part of the struct, so it takes part in `==`. That is safe because it is a function of the same value the encoding came from: equal values produce equal text, so no new key appears. Two *different* values that shared an encoding would now be told apart rather than merged, which is a better answer than before. The one thing that would break it is a `String()` that returns different text for the same value, which would count that value twice; it is documented next to the field rather than guarded, because nothing can detect it.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `cell-identity`: a stand-in key prints the way the value writes itself.

## Impact

- A printed counter holding a `Stringer` value reads as the value instead of its internals. Nothing else changes: identity, counting, grouping and lookup are untouched, because the encoding still decides them.
- `UncomparableKey` gains an unexported field. `Type` and the behaviour of `ToMapKey` are unchanged.
