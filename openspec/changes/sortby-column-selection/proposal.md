# Proposal: sortby-column-selection

## Why

[#233](https://github.com/HazelnutParadise/insyra/issues/233) (T-23). `DataTableSortConfig{}` cannot be told from `{ColumnNumber: 0}` — in Go they are the same value — so a config that names no column silently sorts by the first one. Measuring every shape of config turned up more than that:

| Config | Result | `Err()` |
| --- | --- | --- |
| `{}` | sorts by column 0 | nil |
| `{Descending: true}` | sorts by column 0, descending | nil |
| `{ColumnIndex: "Z"}` on a two-column table | nothing happens | **nil** |
| `{ColumnName: "scroe"}` | nothing happens | names `GetColByName`, not `SortBy` |
| `{ColumnNumber: 99}` | nothing happens | names `GetColByNumber`, not `SortBy` |

`Docs/DataTable.md` already promises "At least one of ColumnIndex, ColumnNumber, or ColumnName must be specified"; the code never enforced it. It also never says which field wins when more than one is given, although the code comments do and the rule — index before name — is the one `mkt`'s configs document and follow.

A sort with several levels skips a bad level and applies the rest, so the table ends up in an order nobody asked for.

## What Changes

- **A config that names no column is refused.** `{}`, `{Descending: true}` and `{ColumnNumber: 0}` alone all record an error on `SortBy` and leave the table as it was. Because the third is indistinguishable from the first two, the first column by position is written `ColumnIndex: "A"`; the error says so.
- **A column that is not there is refused, and the error names `SortBy`.** Columns are resolved through the silent helpers, so an index that matches nothing, a name that is not there and a number out of range all report the call the user made. A negative `ColumnNumber` stays refused.
- **Several levels are all or nothing.** Every config is resolved before any row moves; one bad level leaves the table unchanged, as `ExecuteCCL` already does.
- **More than one selector is a warning, not an error.** When a config gives more than one of `ColumnIndex`, `ColumnName` and `ColumnNumber`, the sort runs by precedence — index, then name, then number — and logs a warning naming the fields it ignored. `ColumnNumber` counts as given only when it is not zero.
- `Docs/DataTable.md` states the precedence, the zero-value rule and the errors.
- The CLI's `sort` builds only the field it resolved. It used to set `ColumnNumber: -1` next to every name, which would now warn on every sort by name; a number is passed as the matching `ColumnIndex`.

Not changed here: `ColumnIndex` falls back to a column name, uppercased, when its letters are not a position in the table, the same fallback `GetCol` and `SetColToRowNames` use. The owner has ruled that the fallback should be removed; it is shared by three methods and is tracked on [#225](https://github.com/HazelnutParadise/insyra/issues/225).

## Capabilities

### New Capabilities

- `datatable-sort-config`: how a sort config selects its column, and what happens when it selects none, a missing one, or more than one.

### Modified Capabilities

(none)

## Impact

- **BREAKING**: `{ColumnNumber: 0}` alone records an error instead of sorting by the first column; use `ColumnIndex: "A"`. `ColumnNumber` from 1 up is unaffected.
- A config whose column is not there now records an error on `SortBy`; an out-of-range `ColumnIndex` used to do nothing and say nothing.
- A multi-level sort with a bad level no longer applies the good ones.
- A config giving more than one selector logs a warning; the result is unchanged.
