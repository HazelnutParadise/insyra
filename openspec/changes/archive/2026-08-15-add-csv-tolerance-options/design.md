# Design: add-csv-tolerance-options

## Decisions

### Ragged rows: pad short, widen for long — never truncate

`AllowRaggedRows: true` sets `FieldsPerRecord = -1` on the `csv.Reader`. That alone is not enough: `csvRowsToDataTable` appends each row's cells positionally, so a short row would shift every later value up one row in the trailing columns. Normalization is therefore part of the table build, not the reader:

- **Short row** — missing trailing cells become `""`. This matches the existing meaning of an empty CSV cell (inference maps `""` to NaN in float columns; `RawStrings` keeps `""`).
- **Long row** — extra cells are kept by appending new columns to the table. New columns get auto-generated safe names (`safeColName`), and all previously-read rows hold `""` in them. Truncation was considered and rejected: it silently drops data, and the common trailing-comma case only produces one all-empty extra column, which is harmless.

The issue proposed "keep or offer a truncation option"; the truncation option is deliberately not built — it adds a switch whose only effect is silent data loss.

### Interaction with existing options

- `FirstRowToColNames` — the header row defines the initial column count; extra columns created later are auto-named, not header-named.
- `FirstColToRowNames` — a one-field row contributes only its row name; every data column gets `""` for that row.
- `RawStrings` — padding cells are `""` strings, indistinguishable from genuinely empty cells, which is the intended semantic.
- Type inference — padded `""` cells behave exactly like real empty cells today (an otherwise-integer column becomes float64 with NaN). Documented, not special-cased.

### The file path parses once

The file reader used to decode the encoding, parse the CSV, re-serialize it to a string, and parse it again. That round trip destroys any record holding a single empty field: `csv.Writer` emits `[""]` as a blank line, which `csv.Reader` skips. Under strict parsing such records were only reachable in single-column files, but `AllowRaggedRows` makes them reachable in any file, silently dropping rows and shifting row names. The internal reader therefore returns parsed records directly (`ReadCSVRecordsWithEncodingOptions`), and the file path builds the table from them in a single parse — file and string loading now agree on every input. The string-returning `ReadCSVWithEncoding` remains as a shim for `csvxl`.

### TrimLeadingSpace is a plain pass-through

`csv.Reader.TrimLeadingSpace = true` already trims spaces before deciding whether a field is quoted, so it fixes both `2330, 1000` and `2330, "1,000"` in one option. No custom code.

### Naming

Field names follow the sources users already know: `TrimLeadingSpace` is the exact `encoding/csv` field name; `AllowRaggedRows` uses the established "ragged" term for uneven row lengths. CLI options follow the `infer` precedent: single lowercase words `ragged`, `trimspace`.
