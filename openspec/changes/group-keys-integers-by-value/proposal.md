# Proposal: group-keys-integers-by-value

## Why

#230 (T-17 in `api-review.md`) asked for GroupBy to put `int 1` and `float64 1.0` in one group, as pandas does. The owner has ruled the other way: integers match by value across widths while floats stay separate (`match-integers-by-value`, and `counter-keys-integers-by-value` on 2026-09-27), so `1` and `1.0` stay two groups.

What the finding also implies, that a CSV load's `int64(1)` and a Go literal `1` must fall in one group, already holds: the group key encoder writes every builtin integer width as `i:<value>`. Checked on `0.4` at fc01a546, one width was missing. `uintptr`, which `Count`, `Counter` and `ToMapKey` match by value (`value-matching`, "Integers are counted by value"), fell through to the generic encoder and formed its own group: a column holding the value 1 in each of the eleven integer widths grouped into sizes `[10 1 …]`, and `OpNUnique` over `1`, `int64(1)`, `uintptr(1)`, `1.0`, `"1"` returned 4. Nothing documented the rule.

## What Changes

- The group key (`encodeGroupKey`), the unique-count key (`uniqueKey`) and the scalar arm of `encodeCell` treat `uintptr` as an integer, so `GroupBy`, `Pivot`, `Merge` keys, `OpNUnique` and `Describe`'s unique count put it with the other widths.
- `Docs/DataTable.md` states the rule at `GroupBy`: integers by value across widths, floats by value across `float32` and `float64`, text apart, and an integer never with a float or a string of the same value.
- Not changed: `1` and `1.0` stay two groups, by the owner's ruling.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `value-matching`: adds "Group keys compare integers by value".

## Impact

- `datatable_groupby.go`, `cell_identity.go`; new `groupby_integer_keys_test.go`.
- `Docs/DataTable.md`, both changelogs, `api-review.md`, `delivery-status.md`.
- The agent skills do not describe grouping keys and are unchanged.
