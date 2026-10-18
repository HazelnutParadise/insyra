# Proposal: nan-is-not-a-map-key

## Why

`identify-uncomparable-cells` set the rule that counting and searching give the same answer. NaN breaks it, and NaN in a numeric column is ordinary — it is what a missing value reads as.

Measured on a list holding three NaNs and one `1.0`:

```
Counter has 4 entries      (should be 2)
counter[NaN]  = 0          (not retrievable)
Count(NaN)    = 3          (correct)
```

`float64` is comparable in Go's type system, so `ToMapKey` passes a NaN straight through as its own map key. But `NaN != NaN`, so every `counter[NaN]++` creates a **new entry nobody can ever look up**: three NaNs become three entries of one. `equalCell` meanwhile treats two NaNs as equal, the pandas rule, so `Count` is right and the two disagree.

The guard is wrong, not the special case. `reflect.Value.Comparable()` asks whether `==` is legal, and the question that matters for a map key is whether `==` is *useful*: **a value can be a key only if it equals itself.** That single test also catches the nested shapes, measured:

| Value | `Comparable()` | `v == v` |
| --- | --- | --- |
| `[2]float64{NaN, 1}` | true | **false** |
| `struct{Name string; V float64}{"x", NaN}` | true | **false** |
| the same struct with `V: 1` | true | true |
| `[]byte{1}` | false | (panics) |

## What Changes

- The guard becomes "comparable **and** equal to itself". `Comparable()` still runs first, because `==` panics otherwise.
- A NaN, and any array or struct containing one, is keyed by an `UncomparableKey` like any other value that cannot be a key. All the NaNs in a column collapse to one entry with the right count, and `counter[ToMapKey(math.NaN())]` reads it.
- `UncomparableKey`'s documentation says what it now covers: not only a value Go refuses to compare, but one whose comparison is useless.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `cell-identity`: a value that does not equal itself is not a map key either.

## Impact

- `Counter` on a column holding NaN changes from one unreachable entry per NaN to a single entry with the count. Nothing could read the old entries, so nothing can depend on them.
- `Count`, `FindAll` and the rest are unchanged: `equalCell`'s NaN rule already made two NaNs equal.
- `ToMapKey(math.NaN())` returns an `UncomparableKey` rather than the NaN. `counter[math.NaN()]` returns 0 as it always did, and always must, because a Go map cannot match a NaN key.
