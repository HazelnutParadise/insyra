## Why

The last three items of #414: Excel's long names only say the function does not exist, `CEIL` and `FLOOR` cannot round to a multiple, and `REGEX_MATCH` differs from the name Excel 365 uses.

## What Changes

- `AVERAGE`, `AVERAGEIF`, `POWER`, `CEILING`, `REPT`, `ISNUMBER`, `ISLOGICAL`, `CONCATENATE` and `SUBSTITUTE` stay errors, and name what to write.
- `CEIL` and `FLOOR` take an optional positive multiple.
- `REGEXTEST` is added, and `REGEX_MATCH` is deprecated with one warning. It is removed in 0.4.

## Impact

- `internal/ccl`, `Docs/CCL.md`, both changelogs.
