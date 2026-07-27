# Proposal: csv-raw-read-option

## Why

`ReadCSV_File` / `ReadCSV_String` always run column-level type inference, which is destructive for "looks numeric but isn't" data: Taiwan stock IDs lose leading zeros irreversibly (`0050` → int64 `50`), exact monetary amounts are coerced to float64, and empty cells become NaN ([issue #188](https://github.com/HazelnutParadise/insyra/issues/188)). Callers who need exact values (stock IDs, tax IDs, phone numbers, decimal amounts) currently cannot use Insyra's CSV loading at all.

## What Changes

- Add `CSVReadOptions` struct (row/col name flags, encoding, `RawStrings bool`) to the root package.
- Add `ReadCSV_FileWithOptions` / `ReadCSV_StringWithOptions`; when `RawStrings` is true every cell stays its original string (empty cell stays `""`), and type inference is skipped entirely.
- Existing `ReadCSV_File` / `ReadCSV_String` keep their exact signatures and behavior, becoming thin wrappers over the new functions.
- `isr`: `CSV_inOpts` gains a `RawStrings` field, passed through by `DT.From`.
- CLI: `load <file.csv> infer true|false` option (default `true`); rejected for JSON/Excel.
- Docs (`Docs/DataTable.md`, `Docs/cli-dsl.md`) and skills (`skills/insyra`, `skills/use-insyra-cli`) updated in the same change.

## Capabilities

### New Capabilities
- `csv-read-options`: Options-based CSV loading API in the root package, including the raw-strings (no type inference) mode.

### Modified Capabilities
- `dsl-commands`: `load` command gains an `infer true|false` option for CSV files; non-CSV formats reject it.

## Impact

- `read.go` (new options struct + functions; existing functions delegate), `read_test.go`
- `isr/csv.go`, `isr/dt.go`
- `cli/commands/load.go` and its tests
- `Docs/DataTable.md`, `Docs/cli-dsl.md`, `skills/insyra/SKILL.md`, `skills/use-insyra-cli/SKILL.md`
- No breaking changes; no new dependencies.
