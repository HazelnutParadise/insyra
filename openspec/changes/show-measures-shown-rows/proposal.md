# Proposal: show-measures-shown-rows

## Why

`ShowRange(5)` on a 1,000,000 × 3 table took about 475 ms (#236, IN-22 in `api-review.md`), because `prepareTableLayout` formatted every cell of the table to measure column widths and built a label for every row, whatever range was printed. `ShowTypesRange` did the same through `prepareTableLayoutTypes`. Measured on this machine (Apple M3, `go test -bench OfAMillionRows -benchtime 5x -count 3`) before the change: `ShowRange(5)` 279–437 ms and 5,999,710 allocations per call, `ShowTypesRange(5)` 173–181 ms, `Show()` 497–553 ms. A wide cell far below the printed range also widened the printed columns.

#236 also asked, as E-8, for three things. `Show(label, object showable, …)` took an unexported parameter type, so a caller could not write a function that accepts what `Show` accepts. The other two are not part of this change. Whether `Show()` should truncate by default is the owner's decision; it already prints the first 20 and last 5 rows of a table longer than 25, as it has since 323ec68d (2025-05-19), though `Docs/DataTable.md` said "shows all rows". The `ShowRange` arguments were settled by `core-settings-batch`.

## What Changes

- `ShowRange`, `Show` and `ShowTypesRange` on a DataTable measure column widths and build row labels from the rows they print: the range, or the first 20 and last 5 rows when a table of more than 25 rows is shown without a range. After the change: `ShowRange(5)` 38–42 µs and 257 allocations, `ShowTypesRange(5)` 18–20 ms (what remains is the `DataType` row, which summarises the whole column on purpose), `Show()` 296–321 ms (what remains is the stat line, which covers every row on purpose). A view that prints every row is byte-for-byte what it was; a view of part of a table can be narrower.
- `Showable` is exported, and `Show` takes it.
- `Docs/DataTable.md` and `Docs/DataList.md` say what no-argument `ShowRange` and `Show` print on a long table.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `show-range`: adds "A table view is sized by the rows it prints" and "Show takes an exported Showable".

## Impact

- `show.go`; new `show_layout_test.go` with three benchmarks.
- `Docs/DataTable.md`, `Docs/DataList.md`, `Docs/utils.md`, both changelogs, `api-review.md`, `delivery-status.md`.
- The agent skills name no Show function and are unchanged.
