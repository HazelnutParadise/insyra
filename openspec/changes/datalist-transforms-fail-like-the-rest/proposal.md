# Proposal: datalist-transforms-fail-like-the-rest

## Why

Classifying every `DataList` method for [#218](https://github.com/HazelnutParadise/insyra/issues/218) on 2026-09-28 found five that fail in a shape the `chainable-never-nil` rule does not allow, recorded as an `AGENTS.md` follow-up. Each was reproduced on `0.4` at 2e223e18 before this change:

- `Normalize` changes the list in place, but on failure returned an empty list, so `dl = dl.Normalize()` threw the data away. `[1, 2, "x"].Normalize()` returned `[]`.
- `Sample` and `SampleFrac` returned a bare `NewDataList()` on every failure, so the result's `Err()` was nil. `Sample(-1, false)` and `SampleFrac(2, false)` gave a nil result error.
- `Rank` and `Shift` given two optional values returned the receiver itself, so `r := dl.Shift(1, 0, 0)` handed back the unshifted list as if it were the shifted one.

The follow-up's `Shuffle` case is covered by `shuffle-in-place`. `Shuffle` now changes the list in place and returns it unchanged on failure, and `Shuffled` returns `failedResult()`.

## What Changes

- **BREAKING**: `Normalize` returns its receiver, unchanged, on failure, like `Standardize` and the other in-place methods.
- **BREAKING**: `Sample`, `SampleFrac`, and `Rank` and `Shift` with two optional values return `failedResult()`, an empty list carrying the error, which is also recorded on the receiver. An empty input to `Sample` and `SampleFrac` is still an empty result with no error.
- The `chainable-never-nil` requirement states both halves of the rule: an in-place method returns its unchanged receiver on failure, and a transform that builds a new list returns an empty list carrying the error.
- `TestChainableTransformsNeverReturnNil` drops `Normalize`, which is not a new-list transform, and `TestDataListNormalize` expects the receiver back.
- `Docs/DataList.md` (Error Handling, `Normalize`, `Rank`, the sampling errors), both changelogs, `AGENTS.md` (the follow-up is removed), `api-review.md` (D-7) and `delivery-status.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `chainable-never-nil`: the failure shape of an in-place method is stated beside that of a new-list transform.

## Impact

- `datalist.go` (`Normalize`, `Rank`), `datalist_window.go` (`Shift`), `datalist_sampling.go` (`Sample`, `SampleFrac`).
- A caller that checked `dl.Normalize().Len() == 0` to detect a failure now gets the list back; `Err()` is the check.
