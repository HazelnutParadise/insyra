## Why

The fifth batch of #213 (C-2). `csvxl` already refused a second encoding and an unknown one, but it read the encoding argument differently from the core CSV readers: an empty string meant UTF-8 taken as-is where the core detects, and only lowercase `"auto"` detected, so `"AUTO"` failed as an unsupported encoding. The same argument now means the same thing in both. Detection checks for valid UTF-8 before anything else, so a UTF-8 file reads the same as before.

## What Changes

- An empty encoding and `"auto"` in any case mean detection in `CsvToExcel`, `AppendCsvToExcel`, `EachCsvToOneExcel` and `ReadCsvToString`.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `optional-values`: an encoding argument means the same in `csvxl` as in the core readers.

## Impact

- `csvxl/convert.go`, `csvxl/read_csv.go`, `csvxl/encoding_names_test.go` (new); `Docs/csvxl.md`, both changelogs.
