# Proposal: shuffle-in-place

## Why

D-7 of the API review ([#218](https://github.com/HazelnutParadise/insyra/issues/218)) found that a `DataList` method's name does not say whether it changes the list or returns a new one. The documentation half is done (49c559fa): `Docs/DataList.md` "In Place or a New List" lists the 38 methods that change the list and the 21 that return a new one, and says that `GetCol` and its explicit spellings return a copy (measured: `dt.GetColByName("x").Sort()` leaves the table as it was, because `getCol` returns `dt.columns[num].Clone()`).

Measured on `0.4` at 2e223e18 for the ruling:

- Judged by the Swift API Design Guidelines' rule, where a mutating method reads as an imperative verb (`sort()`) and a non-mutating one as a noun or an `-ed` form (`sorted()`), the names already say which group they are in, except `Shuffle`. It returns a new list, while Swift's `shuffle()`, Kotlin's `shuffle()`, Python's `random.shuffle` and NumPy's `Generator.shuffle` all shuffle in place, and `shuffled()` is the copying form. The other new-list names are nouns (`Diff`, `CumSum`, `MovingAverage`) or return new values in every reference library (`Filter`, `Map`, `Sample`, `Rank`, `Shift`).
- Making every transform return a new list, pandas' default, would turn every statement-form call into a silent no-op: a type-aware count found 108 such calls in library code (86 of them `Append`) and 148 in tests.

The owner ruled on 2026-10-14: option 乙, with `Shuffle` becoming the in-place method and **not** deprecated. #222 deprecated look-alike names so their results would not change under the same name. For this one name the owner chose the meaning every reference library gives it instead, on the breaking line.

## What Changes

- **BREAKING**: `DataList.Shuffle` and `DataTable.Shuffle` reorder the receiver in place and return it, the way `Sort` and `Reverse` do. The name is kept. A failure (two `SamplingOptions`) records the error and leaves the receiver as it was.
- New `DataList.Shuffled` and `DataTable.Shuffled` return a shuffled copy named `<name>_Shuffled` and leave the receiver as it is, which is what `Shuffle` did. For one seed, `Shuffle` and `Shuffled` give the same order. A failed `DataList.Shuffled` returns an empty list carrying the error. A failed `DataTable.Shuffled` returns an empty table, as `DataTable`'s other sampling methods do.
- `IDataList` and `IDataTable` gain `Shuffled`.
- The CLI's `shuffle … as` calls `Shuffled`, so it keeps storing a copy and leaving the variable as it was. `ml`'s fold split keeps calling `Shuffle` on the table it builds for itself, so the folds are the same and one copy is saved.
- `Docs/DataList.md`, `Docs/DataTable.md`, both changelogs, the `insyra` skill (the principle that a method changes its value or returns a new one, with the verb/`-ed` pair), `api-review.md` (D-7) and `delivery-status.md`.

## Capabilities

### New Capabilities

- `in-place-or-copy`: which transforms change their receiver and which return a copy, as the `Shuffle`/`Shuffled` pair shows.

### Modified Capabilities

None.

## Impact

- `datalist_sampling.go`, `datatable_sampling.go`, `interfaces.go`, `internal/dsl/commands/sample.go`, and the sampling tests.
- A caller of `Shuffle` that kept using the receiver in its original order now sees it shuffled. `s := dl.Shuffle()` still gives a shuffled `s`.
