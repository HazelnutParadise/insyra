# 0027 — A cell is identified by its type and content

**Status:** accepted
**Date:** 2026-09-12

## Context
Cells are stored as `[]any`. A cell Go cannot compare, such as a `[]byte` or a `[]int`, panicked in `Counter` and could not be counted, searched, matched or grouped. NaN compares unequal to itself, so it broke every map keyed by cells.

## Decision
A cell Go cannot compare is identified by its type and content, and that one rule covers counting, searching, matching and grouping (`identify-uncomparable-cells`). The test for whether a value can be a map key is self-equality, not comparability: comparability is necessary, so `==` does not panic, but not sufficient, so `==` means something, and the two differ exactly on NaN and anything containing one (`nan-is-not-a-map-key`). `Counter` keeps `map[any]int`, with a stand-in key only for values that cannot key a map, so `counter[1]` is untouched. `NewDataList` keeps flattening slices, `[]byte` included, and `Cell(v)` keeps one argument whole (`one-value-one-cell`). The helper is `ToMapKey`.

## Options rejected
- A comparable cell type for bytes (`type Bytes string`). `[]byte` and `Bytes` would be two unequal ways to say the same thing, the type assertion people write by habit would fail, and `[]int` cells would still reach the `Counter` panic through `Append`.
- Change `Counter`'s signature to carry the original value. Comparable values would no longer key by themselves.
- Stop flattening `[]byte` in `NewDataList`. The flattening is what makes the constructor read like a pandas Series; the escape hatch is per argument instead.
- The names `KeyOf` (it is the key for the Go value, which differs on integer width) and `CounterKey` (it binds a general helper to its first caller).

## Consequences
Whether `Counter` should merge `int(1)` and `int64(1)` was left open at the time. `counter-keys-integers-by-value` (2026-09-27) later keyed integers by value, because a CSV load stores `int64` and a Go literal is `int`, so `counter[5]` returned 0 on a column holding two fives.
