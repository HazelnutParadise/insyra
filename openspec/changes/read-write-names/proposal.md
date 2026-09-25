## Why

The last batch of #213 (K-14, and the renames #210 deferred here so callers break once). The CSV and JSON readers and writers had two names each (`ReadCSV_File` and `ReadCSV_FileWithOptions`), underscores Go does not use, positional bools nobody could read at the call site (`ReadCSV_File(path, false, true)`, `ToCSV(path, false, true, false)`), and options structs whose zero value described a file almost nobody has: no header row. Every one of the 38 documented calls turned the header on. The owner ruled on 2026-09-25: one name per function with settings in one optional options struct; header and row names spelled `NoHeaderRow` and `HasRowNames` everywhere, so the zero value is the common file; the old names kept one release as Deprecated wrappers; `isr` keeps its short `Opts` type names.

## What Changes

- **BREAKING**: `ReadCSVFile(path, opts...)`, `ReadCSVString(s, opts...)`, `ReadJSONFile(path)`, `ToJSONBytes`, `ToJSONString`; `ReadCSV(r, opts...)`; `StreamCSV(r, batchSize, opts...)`; `ToCSV(path, opts...)` replaces the positional-bool `ToCSV`; `WriteCSV(w, opts...)`.
- **BREAKING**: `CSVReadOptions`, `CSVWriteOptions`, `ExcelWriteOptions`: `NoHeaderRow` (inverted from the old header field) and `HasRowNames`; `ToSQLOptions.HasRowNames`. `isr`'s `CSV_inOpts`, `CSV_outOpts`, `Excel_inOpts` fields follow.
- `ReadCSV_File`, `ReadCSV_FileWithOptions`, `ReadCSV_String`, `ReadCSV_StringWithOptions`, `ReadJSON_File`, `ToCSVWithOptions`, `ToJSON_Bytes`, `ToJSON_String` remain one release, Deprecated, with their old meaning.
- `ReadExcelSheet`/`ReadExcel` rename their bool parameters to `rowNames`, `headerRow`; order and meaning unchanged, since flipping a positional bool would silently invert every call.
- The CLI's `load`/`save` keep their `headers`/`rownames` tokens and defaults.

## Capabilities

### New Capabilities
- `read-write-names`: one name per reader and writer, and a zero value that is the common file.

### Modified Capabilities
None.

## Impact

- `read.go`, `datatable_csv.go`, `datatable_json.go`, `datatable_excel.go`, `datatable_to_sql.go`, `interfaces.go`, `isr/csv.go`, `isr/excel.go`, `isr/dt.go`, `cli/commands/load.go`, `cli/commands/save.go`, `cli/commands/db_save.go`, `cli/env/state.go`; tests across the root, `isr` and the CLI.
- `Docs/`, `skills/`, both READMEs, both changelogs; `AGENTS.md` follow-up to remove the deprecated names.
