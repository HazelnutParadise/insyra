# Proposal: match-integers-by-value

## Why

A CSV load stores every integer as `int64`. Go code and the CLI write an integer as `int`. The library's value lookups compared the two with `==`, and inside an `any`, `int64(2) == int(2)` is false. Measured on 2026-09-11 against a table read from CSV:

- `dt.Count(2)` returns 0, `FindAll(2)` and `FindRowsIfContains(2)` return nothing, and `Replace(2, 0)` changes nothing.
- The CLI inherits it: `count t 2` prints 0, `find t 2` prints `[]`, and `replace t 2 0` prints `replaced` and changes nothing. In one-shot mode every restored variable is `int64`, so this happens to any integer data, not only CSV.
- `encode t ordinal v order 1,2,3` on a CSV column prints success and encodes every cell as nil, because the encoders key a category by its Go type as well as its value.

The library produces both types itself: `groupby`'s count column, `LabelEncode` codes and `kmeans` labels are `int`, while CSV loads and restored variables are `int64`. No choice of literal type in the CLI can match both, so the comparison has to change.

## What Changes

- **Integers match by value, whatever their Go type.** The value lookups on DataList (`Count`, `FindFirst`, `FindLast`, `FindAll`, `ReplaceFirst`, `ReplaceLast`, `ReplaceAll`, `DropAll`) and DataTable (`Count`, `FindRowsIfContains`, `FindRowsIfContainsAll`, `FindColsIfContains`, `FindColsIfContainsAll`, `Replace`, `ReplaceInRow`, `ReplaceInCol`, `DropRowsContain`, `DropColsContain`) share one rule. Two integers are equal when their values are, across `int`, `int8` to `int64` and `uint` to `uint64`, and a negative value never equals an unsigned one.
- **Floats stay separate**, as the owner chose: `2` does not match `2.0`, and a float is found by searching with a float. NaN still matches NaN.
- **Encoder categories follow the same rule.** The one-hot, label and ordinal encoders key an integer category by its value. `Order: []any{1, 2, 3}` then matches a CSV column, an encoder fitted on `int` data transforms `int64` data, and `int(1)` and `int64(1)` in one column become one category instead of two that collide.
- `IsEqualTo` and `IsTheSameAs` keep comparing type by type. They answer whether two lists hold identical data, not whether a value occurs, the way pandas' `equals` also checks dtype.
- The CLI needs no code of its own: `count`, `find`, `replace` and `encode` call these methods.

## Capabilities

### New Capabilities

- `value-matching`: how the library decides that a cell holds the value being looked for.

### Modified Capabilities

(none)

## Impact

- `datalist.go`, `datalist_notatomic.go`, `datatable.go`, `datatable_replace.go`, `datatable_encode.go`; tests in the root package and `cli/commands`; a benchmark, `value_lookup_bench_test.go`.
- Results change only where an integer was compared with an integer of another Go type, a comparison that never matched before.
- The comparison is picked once per call from the value being searched for. A first version that decided it again for every cell made a million-cell `Count` of floats or strings three times slower. The version kept leaves integers unchanged and makes floats (2.3 to 1.5 ms) and strings (2.7 to 2.0 ms) faster, because they are compared as their own type instead of through Go's general interface equality.
- Not changed: `Counter()` still keys by the stored value, so a column that mixes `int(1)` and `int64(1)` reports two keys. A column only mixes the two when rows are added by hand to loaded data. Recorded as a follow-up. GroupBy, Pivot and Merge already key an integer by its value (`encodeGroupKey` writes `i:<value>`), so they needed no change.
- `Docs/DataList.md`, `Docs/DataTable.md`, the CLI usage reference's literal table, `CHANGELOG.md`, `CHANGELOG_TW.md`; the `AGENTS.md` follow-up this resolves is deleted.
