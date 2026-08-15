# Proposal: add-csv-tolerance-options

## Why

Real-world CSV exports (broker inventory files, Excel re-saves) commonly contain a trailer note row with fewer fields (`以上資料僅供參考`), rows with a trailing comma (one extra empty field), and a space before a quoted field (`2330, "1,000",600.86`). The internal `csv.Reader` runs with strict defaults, so any one of these fails the whole load with `wrong number of fields` or `bare " in non-quoted-field`, and callers must fall back to `encoding/csv` by hand ([issue #198](https://github.com/HazelnutParadise/insyra/issues/198)).

## What Changes

- `CSVReadOptions` gains two opt-in fields, both defaulting to the current strict behavior:
  - `AllowRaggedRows bool` — rows may have a field count different from the first row. Short rows are padded with empty strings; extra cells in long rows are kept by creating new auto-named columns (no silent data loss).
  - `TrimLeadingSpace bool` — leading white space in a field is ignored, including before an opening quote (maps to `csv.Reader.TrimLeadingSpace`).
- Both `ReadCSV_FileWithOptions` and `ReadCSV_StringWithOptions` honor the new fields.
- `isr`: `CSV_inOpts` gains matching `AllowRaggedRows` / `TrimLeadingSpace` fields, passed through by `DT.From`.
- CLI: `load <file.csv>` gains `ragged true|false` and `trimspace true|false` options (default `false`); rejected for JSON/Excel, same as `infer`.
- Docs (`Docs/DataTable.md`, `Docs/cli-dsl.md`), skills (`skills/insyra`, `skills/use-insyra-cli`), and both changelogs updated in the same change.

## Capabilities

### Modified Capabilities

- `csv-read-options`: two new tolerance options on the options-based CSV loading API.
- `dsl-commands`: `load` command gains `ragged` and `trimspace` options for CSV files; non-CSV formats reject them.

## Impact

- `read.go` (two option fields, reader wiring, ragged-row normalization in `csvRowsToDataTable`), `read_test.go`
- `isr/csv.go`, `isr/dt.go`
- `cli/commands/load.go` and its tests
- `Docs/DataTable.md`, `Docs/cli-dsl.md`, `skills/insyra/SKILL.md`, `skills/use-insyra-cli/SKILL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`
- No breaking changes; zero-value options keep today's behavior exactly. No new dependencies.
