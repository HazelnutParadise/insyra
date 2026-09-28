# Proposal: append-csv-reads-before-replacing

## Why

`AppendCsvToExcel` replaces the target sheet before it reads the CSV. When the CSV cannot be read, because the file is missing, its encoding cannot be detected or it is not valid CSV, the call counts the file as failed and moves on, and then saves the workbook with that sheet empty. Measured on 2026-09-28: the call returned `1 files failed to append` and the sheet read back with no rows. The data the sheet held is gone from the file. v0.3.3 introduced this: it cleared the sheet in place before reading, and the rebuild that replaced the clearing kept the order.

## What Changes

- `AppendCsvToExcel` reads each CSV in full before it touches that CSV's sheet. A CSV that cannot be read leaves its sheet, and the rest of the workbook, as they were, and no longer adds an empty sheet when the workbook had none of that name; the other CSVs in the call are still appended, and the call still returns `<n> files failed to append`.
- Reading a CSV is split from writing its records, so `CsvToExcel`, which writes into a new workbook, keeps its current behaviour.
- The error text, the sheet replacement itself and every other behaviour stay as they are. `0.4` made the same fix as part of a breaking change to the error (71c168a8); the error change stays there.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `csvxl-excel-append`: a CSV that cannot be read leaves its sheet untouched.

## Impact

- `csvxl/convert.go`, a test in `csvxl/`, `Docs/csvxl.md`, both changelogs, and the `AGENTS.md` follow-up this resolves.
- Lands on `dev` for Huashan v0.3.4.
