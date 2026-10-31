# Proposal: core-stringer

## Why

Every type in the module that has `Show()` also has `String()` except the two core types, `DataList` and `DataTable`, so their text could only be had through `ShowTo` and a buffer, and `fmt.Println(dt)` printed internal fields. The owner asked on 2026-10-31 that every `Show` have a `String`, and approved this design after a Fable review.

## What Changes

- `DataList.String()` and `DataTable.String()` return the view `Show` prints, without colour and laid out for 80 columns, whatever the terminal and `Config` say; both are added to `IDataList` and `IDataTable`.
- The views are drawn by one renderer that takes the width and whether to colour, so `Show` and `String` cannot drift apart.
- A cell holding a table or a list is shown as `DataTable(rows x cols)` or `DataList(n)` in every view, so a table holding itself no longer recurses without end.
