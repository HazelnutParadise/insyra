## Why

The library has four ways of saying which column an operation means, and they disagree:

| Where | Rule |
| --- | --- |
| `GetCol`, `colSilently` (`SetColToRowNames`, `SortBy`) | Excel index first, then the string uppercased and looked up as a name |
| `GroupBy`, `Pivot`, `Unpivot`, `Resample`, the ten window `*Col` methods | **name first**, then the Excel index |
| `DataTableSortConfig` | three fields, `ColumnIndex` / `ColumnName` / `ColumnNumber`, with a documented precedence and a warning when more than one is set |
| `isr.Col` | a typed selector: a string is an index, `isr.Name("x")` is a name, an int is a position |

So `dt.GetCol("price")` and `dt.CumSumCol("price")` read the same string on the same table and answer differently, which is #225 (T-11). The owner ruled on 2026-09-22 that one rule is worth the churn, and that the fourth line above is the one to keep, because it is the only one where the reader says what they mean.

## What Changes

- **One selector everywhere.** Every column parameter and every config field takes `any`: a `string` is an Excel-style index (`"A"`, `"B"`, `"AA"`), `Name("price")` is a column name, an `int` is a 0-based position. `Name` moves into the root package and `isr.Name` becomes an alias of it, so a selector built in either package works in both.
- **The explicit methods stay and are completed.** `GetColByIndex`, `UpdateColByName`, `SetColToRowNamesByName`, `ReplaceInColByName`, `ReplaceNaNsInColByName`, `ReplaceNilsInColByName` and `ReplaceNaNsAndNilsInColByName` are added, so every core accessor has all three spellings beside the generic one. The analysis family gets no new methods: the selector already says which.
- **BREAKING**: a bare string is never a column name. `GroupBy("revenue")`, `CumSumCol("revenue")`, `PivotConfig{Values: "sales"}` and `Resample("ts", ...)` must say `Name("revenue")`. Most such calls fail loudly, because a name rarely decodes to an in-range index.
- **BREAKING**: `GetCol` and `colSilently` lose the fallback that uppercased a string and looked it up as a name, which is the 2026-09-13 ruling on #225.
- **BREAKING**: `DataTableSortConfig`'s three fields collapse into one `Col`. The precedence rule and its warning go with them; an empty config still sorts by the first column.
- A bare string that resolves to an in-range index on a table that also has a column of exactly that name records a warning naming both readings. Names shape diagnostics, never resolution, the same rule `ccl-identifier-is-only-an-index` settled.

## Capabilities

### New Capabilities
- `column-selector`: what a column selector is, how each form resolves, and what a failure says.

### Modified Capabilities
- `datatable-sort-config`: the config carries one selector, so the precedence requirement is replaced by the shared rule.

## Impact

- 43 `DataTable` methods, `PivotConfig`, `UnpivotConfig`, `ResampleAgg`, `AggregateConfig`, `DataTableSortConfig`.
- `isr` (its `name` type becomes the root's), `cli` (its own selector tokens), `engine` if it re-exports any of them.
- `Docs/DataTable.md`, `Docs/CCL.md` cross-references, both READMEs, both changelogs, `skills/insyra/`, `skills/use-insyra-cli/`.
- `api-review.md` T-11, `delivery-status.md`, issue #225.
