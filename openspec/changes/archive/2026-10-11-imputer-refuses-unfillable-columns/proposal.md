## Why

`SimpleImputer` with the mean or median left a selected column that is not a number column unfilled and reported it only through `Params()[name].PassThrough`, so `Fit` returned nil and the fit looked complete when a column the caller had chosen was never going to be filled. The table fills now report a named column they cannot fill (`core-settings-batch`), and the owner ruled on 2026-09-26 (#213) that the imputer follows the same rule, since every column it fits is one the caller named.

## What Changes

- `ScalerParams.PassThrough` is marked Deprecated. On this line it keeps its meaning, and it is removed in 0.4.
- Left on 0.4: **BREAKING (behaviour, not signature)**. `SimpleImputer.Fit` with `ImputeMean` or `ImputeMedian` fails when a selected column is not a number column, naming the column and its data type, and learns nothing. On this line the imputer still leaves such a column unfilled and reports it only through `PassThrough`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. The `table-fill` requirement that describes the refusal stays with the 0.4 change.

## Impact

- `datatable_scale.go` (the `ScalerParams.PassThrough` doc comment).
- `Docs/DataTable.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- Not on this line: `datatable_simple_imputer.go` and its tests, `skills/insyra/SKILL.md` (the refusal they would carry is left on 0.4), and the AGENTS.md follow-up for removing `PassThrough`.
