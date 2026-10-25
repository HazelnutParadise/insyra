# Proposal: csv-refuse-unknown-encoding

## Why

Reported from Syralit on 2026-10-25: `insyra.CSVReadOptions{Encoding: ...}` with a mistyped or unsupported name did not fail; the bytes were handed back undecoded, so a Big5 file read with "big6" came out as mojibake and no error. `internal/csv.CheckDecodable` already refuses such a name with a message listing the supported ones, but only `csvxl.ReadCsvToString` called it; the root readers and the `csvxl` converters did not.

## What Changes

- `ReadCSV`, `ReadCSV_FileWithOptions`, `StreamCSV` (and so `isr` and the CLI `load`) return the `CheckDecodable` error for an explicit `Encoding` nothing decodes. A charset the auto-detector reports is read as before.
- `csvxl.CsvToExcel` and `AppendCsvToExcel` refuse such a name before opening or writing anything.
- The Windows code page names `cp950` and `cp936` are accepted as Big5 and GBK (GB18030), as `cp932` and `cp949` already were.

## Capabilities

### Modified Capabilities

- `csv-encoding-integrity`: an explicit encoding name nothing decodes is an error in every CSV reader.

## Impact

`read.go`, `csvxl/convert.go`, `internal/csv/decoder.go`, their tests, `Docs/DataTable.md`, both CHANGELOGs. A behavior change: a call that used to return garbled text now returns an error.
