# Proposal: slicecols-takes-column-selectors

## Why

`datatable-slicing` gave `SliceCols(from, to int)` positions only (#231). The owner asked on 2026-09-28 that it take column letters and names too. The repository's one-column-selector rule says the same for every parameter that picks a column, and the ten deprecated methods it replaces took Excel letters, so their callers had to turn a letter into a position with `ParseColIndex` before they could move over.

## What Changes

- **BREAKING (unreleased API)**: `SliceCols(from, to any)`. Each bound is a column selector: a letter, a `Name` or an int. The change is to a method added on `0.4` and not yet released.
- The range stays half-open, as in a Go slice, whatever spelling a bound uses. Under the one-selector rule `"D"`, `Name("d")` and `3` are three spellings of one column, so they have to mean one bound. Either end open or either end closed for all spellings keeps that; switching by spelling would not, and pandas keeps its two conventions apart by giving them two accessors, `iloc` (half-open, positions) and `loc` (closed, labels), which insyra does not have. Half-open matches `SliceRows`, Go and `iloc`, and lets two ranges that meet at a column split a table without overlap.
- `nil` is the first column for `from` and one past the last for `to`, since a name or a letter cannot point past the last column. An int bound may be negative, counting from the end, as an int selector does, and may be `NumCols()`.
- A `from` that comes after `to` is reported as such, naming both columns. A bound that picks no column is reported with the selector rule's usual message.
- The deprecation notes of the column index methods give the letter forms: `FilterColsByColIndexGreaterThanOrEqualTo("B")` is `SliceCols("B", nil)`, `FilterColsByColIndexLessThan("C")` is `SliceCols(nil, "C")`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datatable-slicing`: "SliceRows and SliceCols take a range by position" and "A bound outside the table is an error" are rewritten for selector bounds.

## Impact

- `datatable_filters.go` (`SliceCols`, the new `sliceColBound`, five deprecation notes), `interfaces.go`; tests in `datatable_slice_test.go`.
- `Docs/DataTable.md`, the `SliceCols` entry in both changelogs (the method is unreleased, so its entry is corrected in place), `delivery-status.md`.
- The agent skills name no slicing method and are unchanged.
