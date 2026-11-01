## Why

The last three items of #414: Excel's long names only say the function does not exist, `CEIL` and `FLOOR` cannot round to a multiple, and `REGEX_MATCH` differs from the name Excel 365 uses.

## What Changes

- `AVERAGE`, `AVERAGEIF`, `POWER`, `CEILING`, `REPT`, `ISNUMBER`, `ISLOGICAL`, `CONCATENATE` and `SUBSTITUTE` stay errors, and name what to write.
- `CEIL` and `FLOOR` take an optional positive multiple.
- `REGEX_MATCH` becomes `REGEXTEST`: deprecated with a warning on 0.3.6, and **BREAKING** on 0.4, where the old name is an error naming the new one.

## Impact

- `internal/ccl`, `Docs/CCL.md`, both changelogs.
