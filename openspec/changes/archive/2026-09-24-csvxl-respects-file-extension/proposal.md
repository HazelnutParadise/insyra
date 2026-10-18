# Proposal: csvxl-respects-file-extension

## Why

`CsvToExcel` and `AppendCsvToExcel` appended `.csv` to every input path that did not end in lowercase `.csv`. A CSV named `export.txt` was opened as `export.txt.csv`, `DATA.CSV` as `DATA.CSV.csv`, and a file with no extension could not be read under its own name. The owner ruled on 2026-09-24 on `0.4` that `.csv` stays a convenience but is never added over a name the caller wrote. Only the input side is non-breaking and is ported here; the output side (`ExcelToCsv`'s `csvNames`) changes file names that existing callers get and stays on `0.4`.

## What Changes

- **Input** (`CsvToExcel`, `CSVToExcel`, `AppendCsvToExcel`, `AppendCSVToExcel`): the path is opened as given. Only when it does not exist, or names a directory, is `.csv` appended and tried. Everything readable before stays readable, and `export.txt`, `DATA.CSV` and an extensionless file become readable.
- When both `x` and `x.csv` exist, `x` is read, where `x.csv` was before.
- A file that is neither there nor there with `.csv` fails the way an unreadable CSV already did and is counted in the `N files failed` error.

## Capabilities

### New Capabilities

- `csvxl-file-paths`: how `csvxl` turns the paths it is given into the files it reads.

### Modified Capabilities

None.

## Impact

- `csvxl/convert.go`, `csvxl/file_extension_test.go`, `Docs/csvxl.md`, both changelogs.
