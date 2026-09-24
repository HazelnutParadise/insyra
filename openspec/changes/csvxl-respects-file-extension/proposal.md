## Why

`csvxl` appended `.csv` to every path that did not end in lowercase `.csv`, on the way in and on the way out (#269, C-8). Measured on `0.4` on 2026-09-24, `CsvToExcel` cannot read a CSV named `export.txt` (it opens `export.txt.csv`), `DATA.CSV` (it opens `DATA.CSV.csv`, because the check is case-sensitive) or a file with no extension at all, and `ExcelToCsv` cannot write `report.txt`. None of this is in `Docs/csvxl.md`.

The owner ruled on 2026-09-24: both ends keep adding `.csv` as a convenience, but never over a name the caller wrote.

## What Changes

- **Input** (`CsvToExcel`, `AppendCsvToExcel`): the path is opened as given. Only when it does not exist, or names a directory, is `.csv` appended and tried. Everything readable today stays readable, and `export.txt`, `DATA.CSV` and an extensionless file become readable. When both `x` and `x.csv` exist, `x` is read, where today `x.csv` was. When neither exists, the error names both paths it tried.
- **BREAKING** — **output** (`ExcelToCsv`'s `csvNames`, and so the CLI's `convert`): a name that already has an extension is used as written. `report.txt` stays `report.txt` and `REPORT.CSV` stays `REPORT.CSV`, where they used to become `report.txt.csv` and `REPORT.CSV.csv`. A name with no extension still gets `.csv`.
- `Docs/csvxl.md` states both rules.

## Capabilities

### New Capabilities
- `csvxl-file-paths`: how `csvxl` turns the paths it is given into the files it reads and writes.

### Modified Capabilities
None.

## Impact

- `csvxl/convert.go`; `Docs/csvxl.md`; both changelogs.
- `api-review.md` C-8, `delivery-status.md`, issue #269.
