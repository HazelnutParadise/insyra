# Proposal: fix-api-review-batch-6

## Why

Three I/O defects let data be silently corrupted, silently unusable, or turned into something a spreadsheet will execute. A CSV in an encoding insyra cannot decode is copied byte-for-byte into the table, producing cells that are not valid UTF-8 with nothing to say so. A UTF-32 file is reported as UTF-16 because their byte-order marks share a prefix, and a file too short for the detector fails the whole read instead of falling back. A table name containing a space cannot be appended to on SQLite, because one statement interpolates the identifier instead of quoting it. And a cell beginning with `=`, `+`, `-` or `@` is written to CSV verbatim, so whoever opens the file in a spreadsheet executes it. Closes #285, #286, #287.

## What Changes

- One shared `internal/csv.DecodingReader` resolves an encoding name to a decoder for the root reader and `csvxl`. It covers UTF-8/16, Big5, GB18030, Shift-JIS, EUC-JP, EUC-KR, Windows-1250/1251/1252 and ISO-8859-1/2/15, and **returns an error** for anything else instead of passing raw bytes through.
- `DetectEncoding` tests the UTF-32 byte-order marks before the UTF-16 ones, and falls back to UTF-8 with a warning when the detector cannot name a charset, rather than failing the read.
- New `DataTable.ToCSVWithOptions(path, CSVWriteOptions)` with a `SanitizeFormulas` switch that prefixes a formula-leading cell with a single quote. It is **off by default**: turning it on changes the value written, and a data library's CSV is as often a round trip as an export. `ToCSV` is unchanged.
- The SQLite column lookup quotes its identifier like every other statement in the file.

## Capabilities

### New Capabilities

- `csv-encoding-integrity`: text is decoded or refused, never copied through undecoded.
- `csv-formula-safety`: an export can be made safe to open in a spreadsheet.
- `sql-identifier-quoting`: every statement quotes its identifiers.

### Modified Capabilities

(none)

## Impact

- `internal/csv/decoder.go` (new), `internal/csv/read_csv.go`, `csvxl/convert.go`, `utils.go`, `datatable_csv.go`, `datatable_to_sql.go`, plus tests and docs.
- No signature changes. A CSV in an encoding insyra cannot decode now fails instead of loading garbage — that is the fix, but it turns a silent wrong answer into a visible error.
