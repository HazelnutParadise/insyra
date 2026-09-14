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

## Backport to dev (0.3.x)

Dev received:
- The shared `DecodingReader` with the full decoder table and name normalisation, used by the root reader and `csvxl` (`csv-encoding-integrity`, trimmed).
- `DetectEncoding` testing the UTF-32 BOMs before the UTF-16 ones.
- `ToCSVWithOptions` and `CSVWriteOptions{SanitizeFormulas}`, off by default (`csv-formula-safety`).
- The quoted SQLite `PRAGMA table_info` lookup (`sql-identifier-quoting`).

Adapted on dev:
- A name outside the table falls back to the substring rules dev already had (`utf-8`, `big5`, `gb`, `utf-16`), then reads the bytes undecoded, so `big5-hkscs`, `x-gbk` and `utf-8-sig` resolve as before.
- `ToCSVWithOptions` creates the file directly, like dev's `ToCSV`.

Dev received (documented behaviour):
- `csvxl.ReadCsvToString` returning an error naming the supported encodings for a name nothing decodes, given or detected by `Auto` (2026-09-14): v0.3.2's `Docs/csvxl.md` said it "returns UTF-8 content". `SupportedEncodings` comes with it; `CheckDecodable` is dev's check, because `DecodingReader` keeps its fallback for the other readers.

Stayed on 0.4:
- An unknown encoding name returning an error in the other CSV readers (`ReadCSV_File`, `CsvToExcel`, `AppendCsvToExcel`, `EachCsvToOneExcel`): starts returning an error, and their documentation did not promise UTF-8.
- A chardet failure returning `utf-8` with a warning: `DetectEncoding` and the readers built on it stop returning an error.
