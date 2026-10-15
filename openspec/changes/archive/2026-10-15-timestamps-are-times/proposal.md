# Proposal: timestamps-are-times

## Why

D-12 and TS-15 of the API review ([#220](https://github.com/HazelnutParadise/insyra/issues/220)). Every `DataList` and `DataTable` records when it was created and last changed, as `time.Now().Unix()`, so two changes within one second look like no change. `DataList.IsTheSameAs` compares those seconds, which makes its answer depend on the clock rather than the data. Measured on `0.4` at 1e9b11b6:

- Two separately built `NewDataList(1, 2, 3)` in one second are "the same"; so are `NewDataList(1, 2)` followed by `Append(3)` and `NewDataList(1, 2, 3)`.
- `a.Clone()` is "the same" as `a` within one second and different once the second boundary passes.
- `Append(4)` then `Pop()` leaves `GetLastModifiedTimestamp()` unchanged.
- `TestDataListGetCreationTimestamp` sleeps for a real second to cross a boundary.

Nothing else in the repository reads either timestamp, and the CLI's `state.json` does not store them.

The owner ruled on 2026-10-15: option 甲 with the names `CreationTime` and `LastModifiedTime`, and `IsTheSameAs` comparing data and name.

## What Changes

- `DataList` and `DataTable` gain `CreationTime() time.Time` and `LastModifiedTime() time.Time`, at the clock's full resolution. The names drop `Get`, as Effective Go recommends and as `ColNames()` and `Data()` already do.
- `GetCreationTimestamp()` and `GetLastModifiedTimestamp()` are **deprecated**. For one release they keep returning Unix seconds, then they are removed (AGENTS.md follow-up).
- `LastModifiedTime` still never moves backwards when the wall clock does.
- **BREAKING (silent result change)**: `DataList.IsTheSameAs` compares data and name only. Two lists that differ only in when they were created or changed are now the same. Before, that depended on whether both fell in the same second. Object identity is `a == b`.
- `IDataList` and `IDataTable` gain `CreationTime` and `LastModifiedTime`.
- A package clock (`var clock = time.Now`, unexported) lets the tests set time instead of sleeping.
- `Docs/DataList.md`, `Docs/DataTable.md`, both changelogs, `api-review.md` (D-12, TS-15) and `delivery-status.md`.

## Capabilities

### New Capabilities

- `object-timestamps`: when a list or table was created and last changed, and how `IsTheSameAs` relates to them.

### Modified Capabilities

None.

## Impact

- `datalist.go`, `datatable.go`, `datalist_sampling.go`, `datatable_sampling.go`, `datatable_filters.go`, `datatable_ccl.go`, `interfaces.go`, and their tests.
- A caller of `IsTheSameAs` that relied on two equal lists being different because they were made in different seconds now gets `true`.
