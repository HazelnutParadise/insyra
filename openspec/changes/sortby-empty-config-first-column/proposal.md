# Proposal: sortby-empty-config-first-column

## Why

[#233](https://github.com/HazelnutParadise/insyra/issues/233). `sortby-column-selection` refused a sort config that names no column, and because `{ColumnNumber: 0}` is the same Go value as `{}`, it refused sorting by the first column by number too. The owner reversed that part on 2026-09-13: `SortBy` only runs when a caller asked for a sort, so sorting a config that names no column by the first column is not a sort nobody asked for, and making `ColumnNumber: 0` unusable is not worth it. The zero value is documented instead.

## What Changes

- A config with no `ColumnIndex`, no `ColumnName` and a zero `ColumnNumber` sorts by the first column, with no error and no warning.
- Precedence is unchanged: `ColumnIndex`, then `ColumnName`, then `ColumnNumber`, and `ColumnNumber` counts as given only when it is not zero. A config with a name or an index therefore never falls back to the first column.
- `Docs/DataTable.md` says a config that names no column sorts by the first column.
- The CLI `sort` passes a position as `ColumnNumber` again instead of converting it to `ColumnIndex`.
- The rest of `sortby-column-selection` stands: a missing column is an error on `SortBy`, a multi-level sort is all or nothing, and several selectors sort by precedence with a warning.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datatable-sort-config`: the requirement that a config must name a column is replaced by one saying a config that names none sorts by the first column.

## Impact

- `{}`, `{Descending: true}` and `{ColumnNumber: 0}` sort by the first column, as they did before `sortby-column-selection`. That change was never released, so its BREAKING changelog entry is rewritten to the net change rather than reversed by a second entry.
