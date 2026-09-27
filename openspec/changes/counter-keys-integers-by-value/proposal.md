# Proposal: counter-keys-integers-by-value

## Why

`Counter()` keys its map by the stored value, so an integer's Go width decides whether a caller finds it. A CSV or JSON load stores integers as `int64`, while an integer a Go caller writes is an `int`, so `counter[5]` returns 0 on a loaded column holding two fives. A column mixing `int64(5)` with an appended `5` reports two keys while `Count(5)` reports their total. `match-integers-by-value` made every search, count, replace and drop match integers by value, and GroupBy, Pivot and Merge already did; `Counter` is the last place that does not. The AGENTS.md follow-up of 2026-09-11 left the key type undecided.

pandas, Python, R and SQL all count integers by value. Python and pandas also look a count up by value whatever the integer type. A Go map compares keys by type as well as value, so no key can answer both `counter[5]` and `counter[int64(5)]`. The owner ruled on 2026-09-27 that integers inside `int`'s range key as `int`, because that is what a Go literal is, and that a lookup by any width goes through `Count` or `ToMapKey`.

## What Changes

- **BREAKING**: `Counter()` on `DataList` and `DataTable` keys every integer by its value, as an `int` when `int` can hold it, as `int64` or `uint64` otherwise. Code that indexed a counter with `int64(5)`, or type-asserted a key as `int64`, must use `int`.
- `ToMapKey` returns the same key for an integer of any width, so `counter[insyra.ToMapKey(v)]` finds it.
- Floats and strings are unchanged and stay separate from integers, as in `Count`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `cell-identity`: "A stand-in key can be looked up and printed" now leaves integers to a new requirement, "Integers are counted by value".

## Impact

- `cell_identity.go` (`ToMapKey`, new `integerKey`); a new test file `counter_integers_test.go`.
- `Docs/DataList.md`, `Docs/DataTable.md` (the Counter sections, and the `ToToMapKey` typo in both), `skills/insyra/SKILL.md`, both changelogs (the new entry, and the earlier `Counter` entry's sentence this contradicts, plus its typo), `AGENTS.md` (the follow-up is resolved), `delivery-status.md`.
- Not in this change: whether a float `5.0` should count with the integer 5, which pandas does. That is the matching rule `Count` has used since `match-integers-by-value`, and changing it would change every search, not only `Counter`.
