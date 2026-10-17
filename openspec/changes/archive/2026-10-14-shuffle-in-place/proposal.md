# Proposal: shuffle-in-place

## Why

D-7 of the API review (#218), ported from `0.4` without its breaking part. By the verb and `-ed` rule of the Swift API Design Guidelines every name already says whether it changes its value except `Shuffle`, which returns a copy where Swift, Kotlin and Python's `random.shuffle` shuffle in place. On `0.4`, `Shuffle` shuffles in place and `Shuffled` returns the copy.

## What Changes

- New `DataList.Shuffled` and `DataTable.Shuffled` return a shuffled copy named after the receiver with `_Shuffled` appended and leave the receiver as it is. For one seed they give the order `Shuffle` gives.
- On this line `Shuffle` stays a copy and is not deprecated. Its documentation says that `0.4` shuffles in place, so new code that wants a copy calls `Shuffled`.
- The CLI's `shuffle ... as` calls `Shuffled`, so it keeps leaving the variable alone on both lines.
- `IDataList` and `IDataTable` gain `Shuffled`.
- **Left on `0.4`:** `Shuffle` reordering the receiver, which changes what `dl.Shuffle()` does to `dl`.

## Capabilities

### New Capabilities

- `in-place-or-copy`: which methods change their receiver and which return a copy.

### Modified Capabilities

None.

## Impact

- `datalist_sampling.go`, `datatable_sampling.go`, `interfaces.go`, `cli/commands/sample.go`, `Docs/DataList.md`, `Docs/DataTable.md`, both changelogs, and a new test file.
