# Proposal: datatable-names-one-rule

## Why

Api-review T-14 ([#227](https://github.com/HazelnutParadise/insyra/issues/227)) and the AGENTS.md follow-up of 2026-10-01. Measured on `0.4` at f8912873, on a table with columns `a` and `b`. Every call returned a nil `Err()`:

- `SetColNames({"x","y","z"})` appended an empty column `z`. On a 2-row table, `SetRowNames({"r1","r2","r3"})` dropped `r3`. The two name setters disagreed, and neither said anything.
- `SetColNames({"a","b"})` gave `[a_1 b_1]` and `SetColNames({"b","a"})` gave `[b_1 a]`, because `safeColName` counted the column being renamed among the taken names. The row setters had the same defect.
- `AppendCols` of a second `a` gave `[a b a_1]` with no notice.
- The CLI's `setrownames t r1 … r6` on a four-row table reported success and dropped two names. `setcolnames` already refused that.

The owner ruled on 2026-10-15:
- `SetColNames` follows `SetRowNames`.
- A renamed duplicate warns.
- The self-collision is a defect.

## What Changes

- **BREAKING**: `SetColNames` names only the existing columns. Extra names are ignored with a warning listing them, and no column is added. Columns past the last name lose their names, as before.
- `SetRowNames` keeps ignoring extra names and now lists them in a warning.
- A column or row being renamed no longer counts its own name as taken. `SetColNames` and `SetRowNames` check each name only against names given earlier in the same call. The single-target setters check only against the other columns or rows.
- Every place that adds a numeric suffix to a clashing column or row name logs a warning naming the original and the new name: the setters, `AppendCols`, `AppendRowsFromDataList`, `UpdateRow`, `Merge` and the CSV reader. `Err()` is untouched. Sampling with replacement duplicates row names on purpose, so it stays silent.
- `SimpleRandomSample` creates its columns named instead of relying on `SetColNames` to add them.
- **BREAKING (CLI)**: `setrownames` refuses a name count that differs from the row count.
- Updates `Docs/DataTable.md`, `Docs/cli-dsl.md`, both changelogs, `api-review.md`, `delivery-status.md` and AGENTS.md (the 2026-10-01 follow-up is resolved, and the `setrownames` part of the 2026-09-26 one is removed).

## Capabilities

### New Capabilities

- `datatable-names`: how the column and row name setters treat a count mismatch, a clashing name and a column's or row's own name.

## Impact

`datatable.go`, `datatable_colname.go`, `datatable_rowname.go`, `datatable_merge.go`, `datatable_sampling.go`, `read.go`, `py/pyresult_decode.go` (comment), `internal/dsl/commands/setrownames.go`.
