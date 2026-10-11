## Why

#213 applies the owner's 2026-09-25 rulings on how functions take settings.
`ShowRange` and its siblings take positional parameters, but callers often want plainer spellings for the common head and tail views without using negative numbers. `ShowHead` and `ShowTail` provide these explicit names.

## What Changes

- Add `ShowHead`, `ShowTail`, `ShowHeadTo`, `ShowTailTo` on both `DataTable` and `DataList`.
- Left on 0.4: strict ShowRange arguments checking, table fill named column reporting, imputer options struct (`SimpleImputerOptions`), and CLI fillna extrapolate changes.

## Capabilities

### New Capabilities
- `show-range`: display head and tail views.

### Modified Capabilities
None.

## Impact

- `show_head_tail.go`, `interfaces.go`.
- `Docs/DataTable.md`, `Docs/DataList.md`, both changelogs.
