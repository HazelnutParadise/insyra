## Why

`one-column-selector` (#225) made a bare string an Excel-style index everywhere and promised that a failure teaches the fix: a table with a column named `Age` asked for `"Age"` says to write `Name("Age")`. Two parts of that change were not finished.

- The scalers, the simple imputer and the three encoders resolve columns through `resolveEncodingColumn`, which throws the lookup's explanation away. `NewStandardScaler().FitTransform(dt, "Age")` fails with `column Age not found`, which reads as though the column is missing and says nothing about `Name`.
- The documentation and the `insyra` skill were not fully migrated. About thirty examples still pass a column name as a bare string, so they fail when copied; five do not compile at all (`[]string` where `[]any` is required, and a sort config field that no longer exists). Struct listings, prose and Go doc comments still describe name-first lookup, and `isr.Row` examples use string keys, which the append path reads as Excel indices and widens the table to hundreds of thousands of columns.

## What Changes

- `resolveEncodingColumn` returns the lookup's explanation, and every scaler, imputer and encoder failure on a column carries it, so the message says to write `Name(...)` when the table has that name.
- Every documentation and skill example that passes a name as a bare string uses `insyra.Name(...)` / `isr.Name(...)`; code that did not compile is corrected; struct and signature listings match the code; prose and Go doc comments that describe name-first lookup are rewritten; the obsolete `mkt` example built on the removed `ColIndex`/`ColName` pair is deleted.
- Nothing changes in how any selector resolves.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `column-selector`: the fitters (scalers, imputer, encoders) are held to the same failure message as every other selector.

## Impact

- `datatable_preprocess.go`, `datatable_scale.go`, `datatable_simple_imputer.go`, `datatable_encode.go`; Go doc comments in `isr/groupby.go`, `utils.go`, `datatable_pivot.go`, `datatable_groupby.go`.
- `Docs/DataTable.md`, `Docs/isr.md`, `Docs/mkt.md`, `Docs/ml.md`, `Docs/datafetch.md`, `Docs/quant.md`, `Docs/tutorials/sales-analysis-end-to-end.md`; `skills/insyra/SKILL.md`, `skills/insyra/references/window-functions.md`; both changelogs.
