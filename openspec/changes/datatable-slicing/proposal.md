# Proposal: datatable-slicing

## Why

Ten `DataTable` methods named `FilterColsByColIndex…` and `FilterRowsByRowIndex…` (`GreaterThan`, `GreaterThanOrEqualTo`, `EqualTo`, `LessThan`, `LessThanOrEqualTo`) take a contiguous range of columns or rows by position, which is slicing, not filtering (#231, T-21 in `api-review.md`). pandas writes it `iloc[a:b]`. Two of them panic: on `0.4` at 7bfbbdfe, `FilterColsByColIndexLessThan("Z")` on a four-column table fails with `slice bounds out of range [:25] with capacity 4`, and `…LessThanOrEqualTo("Z")` the same way. `Headers` and `SetHeaders` are second names for `ColNames` and `SetColNames`.

The finding counts thirteen names. Ten are these slicing methods; the other three are `Headers`, `SetHeaders` and `DataTable.Counter`, which the ledger tags with T-21 as well.

## What Changes

- Add `SliceRows(from, to int)` and `SliceCols(from, to int)`. The bounds follow a Go slice expression: 0-based, `from` included, `to` excluded, `0 <= from <= to <= NumRows()` or `NumCols()`. A bound outside that range records an error on the table and returns an empty DataTable; Go would panic, and the library never does. The result keeps the table's name, the column names and the row names of what it holds, and owns its data. `IDataTable` lists both.
- The ten index methods are Deprecated, each naming the slice it stands for, and removed in the next release. They keep their old results, including returning an empty table for a bound they do not like, except that the two less-than column methods no longer slice past the end: a letter past the last column keeps every column, which is what "less than" meant.
- `Headers` and `SetHeaders` are Deprecated in favour of `ColNames` and `SetColNames`.
- Not changed: `DataTable.Counter`. It counts values across the whole table, which `DataList.Counter` does not do, so it is not a second name for anything; removing it would remove a capability, which is the owner's decision (asked on #231).

## Capabilities

### New Capabilities

- `datatable-slicing`: `SliceRows` and `SliceCols`, and the deprecated names they replace.

### Modified Capabilities

(none)

## Impact

- `datatable_filters.go`, `datatable_colname.go`, `interfaces.go`; new `datatable_slice_test.go`.
- `Docs/DataTable.md` (a Slicing section, and Deprecated notes on twelve methods), both changelogs, `api-review.md`, `AGENTS.md` (removal follow-up), `delivery-status.md`.
- The agent skills name none of these methods and are unchanged.
