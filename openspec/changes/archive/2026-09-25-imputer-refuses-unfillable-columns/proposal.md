## Why

`SimpleImputer` with the mean or median left a selected column that is not a number column unfilled and reported it only through `Params()[name].PassThrough`, so `Fit` returned nil and the fit looked complete when a column the caller had chosen was never going to be filled. The table fills now report a named column they cannot fill (`core-settings-batch`), and the owner ruled on 2026-09-26 (#213) that the imputer follows the same rule, since every column it fits is one the caller named.

## What Changes

- **BREAKING (behaviour, not signature)**: `SimpleImputer.Fit` with `ImputeMean` or `ImputeMedian` fails when a selected column is not a number column, naming the column and its data type, and learns nothing. Mode and constant are unchanged.
- `ScalerParams.PassThrough` is always false and marked Deprecated; an AGENTS.md follow-up removes it next release.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `table-fill`: the fitted imputer reports a selected column it cannot fill, as the table fills do.

## Impact

- `datatable_simple_imputer.go`, `datatable_scale.go`, `datatable_simple_imputer_test.go`.
- `Docs/DataTable.md`, `skills/insyra/SKILL.md`, both changelogs, `AGENTS.md` follow-up.
