# Proposal: legacy-transforms-pinned

## Why

D-14 of the API review asked whether six `DataList` methods overlap six others.
The investigation showed that no pair is two names for one function: each differs in output length, `nil`/`NaN` handling, edge windows or what a bad input does.

## What Changes

- A test pins each pair: the numbers agree on a fully numeric series, and each difference in the table holds on this line.
- `Docs/DataList.md` gets a table of the six pairs and their differences based on actual results of this line.
- Nothing is deprecated or removed in this change.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datalist-numeric-input`: adds "The older window and fill methods keep contracts of their own".

## Impact

- Test files `datalist_legacy_transforms_test.go` and `datalist_legacy_smoothing_test.go`.
- `Docs/DataList.md`, `delivery-status.md`.
