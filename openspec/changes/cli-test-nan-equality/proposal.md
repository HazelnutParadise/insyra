# Proposal: cli-test-nan-equality

## Why

`approxEqualAny`, the comparison 22 assertions in `cli/commands` use, reads two numeric cells and fails only when `math.Abs(got-want) > tol`. That comparison is false whenever either side is `NaN`, so a result holding `NaN` where the test expects `1` counts as equal. `nan-only-fill-path` ran into it: the old `fillna … ffill limit 1 missing nan` returned `[1 <nil> NaN]`, and a check against `[1 <nil> 1]` with this helper passed. Every CLI test that uses it could be passing over a wrong `NaN`.

## What Changes

- `approxEqualAny` treats `NaN` as equal only to `NaN`. A test pins that.
- Every CLI test is rerun with the fixed helper. None failed, so none of the 22 assertions was passing over a wrong `NaN`.
- `fillCellsEqual`, which `nan-only-fill-path` added only to get around this, is removed; its five assertions use `approxEqualAny` again, and still fail against the pre-fix `fillna` behaviour.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `test-suite-integrity`: adds "A test comparison never lets NaN stand for a number".

## Impact

- `cli/commands/timeseries_test.go`, and whatever the rerun turns up.
- `AGENTS.md` (the follow-up is resolved), `delivery-status.md`. No changelog entry unless the rerun finds a user-visible defect.
