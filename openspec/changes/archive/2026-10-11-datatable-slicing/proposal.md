# Proposal: datatable-slicing

## Why

Ten `DataTable` methods named `FilterColsByColIndex…` and `FilterRowsByRowIndex…` (`GreaterThan`, `GreaterThanOrEqualTo`, `EqualTo`, `LessThan`, `LessThanOrEqualTo`) take a contiguous range of columns or rows by position, which is slicing, not filtering (#231, T-21 in `api-review.md`). pandas writes it `iloc[a:b]`. Two of them panic: `FilterColsByColIndexLessThan("Z")` on a four-column table fails with `slice bounds out of range [:25] with capacity 4`, and `…LessThanOrEqualTo("Z")` the same way. `Headers` and `SetHeaders` are second names for `ColNames` and `SetColNames`.

## What Changes

- Add `SliceRows(from, to int) *DataTable` and `SliceCols(from, to any) *DataTable`.
- `SliceRows` bounds follow a Go slice expression: 0-based, `from` included, `to` excluded, `0 <= from <= to <= NumRows()`. A bound outside that range records an error on the table and returns an empty DataTable.
- `SliceCols` takes an Excel-style letter, an int position (negative counts from the end), or `nil` for an open end (first column for `from`, one past the last for `to`). Column selector `Name()` is not ported to this line.
- `SliceCols` keeps every row of a table whose columns differ in length, padding short columns with `nil`.
- The ten index methods are Deprecated, each naming the slice it stands for, and removed in 0.4. They keep their old results, except that the two less-than column methods clamp to the last column instead of panicking.
- `Headers` and `SetHeaders` are Deprecated in favour of `ColNames` and `SetColNames`, removed in 0.4.
- `IDataTable` lists `SliceRows` and `SliceCols`.
- Left on 0.4: `Name()` column selectors in `SliceCols`.

## Capabilities

### New Capabilities

- `datatable-slicing`: `SliceRows` and `SliceCols`, and the deprecated names they replace.

### Modified Capabilities

(none)

## Impact

- `datatable_filters.go`, `datatable_colname.go`, `interfaces.go`; tests in `datatable_slice_test.go` and `datatable_slice_review_test.go`.
- `Docs/DataTable.md` (a Slicing section, and Deprecated notes on twelve methods), both changelogs, `api-review.md`, `delivery-status.md`.
