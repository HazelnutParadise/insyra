# Proposal: csvxl-go-names

## Why

C-12 of the API review, filed under [#212](https://github.com/HazelnutParadise/insyra/issues/212), found that `csvxl` names break Go's initialism convention: Go writes an initialism in one case (`CSV`, `URL`, `ID`), and every other package in insyra already does (`ReadCSVFile`, `ToCSV`, `CSVReadOptions`), while `csvxl` writes `Csv` in all seven of its exported functions and its options type. `EachCsvToOneExcel` also has to be read twice: it puts every CSV file in a directory into one workbook, which "each CSV to one Excel" does not say. And most of the package's doc comments do not start with the name they document (`// Convert multiple CSV files to an Excel file`, `// CsvEncoding Options`, none at all on `ReadCsvToString`), which `go doc` and linters expect.

The owner ruled on #211 that each function has one name, and that an old name stays for one release as a Deprecated wrapper keeping its old meaning. The shape of the functions, including the parallel `csvFiles`/`sheetNames` slices (C-11, #270), was chosen by the owner on 2026-09-26 and is not changed here.

C-6 ([#268](https://github.com/HazelnutParadise/insyra/issues/268)) is already fixed on this line: `fix-api-review-batch-6` replaced the `strings.Contains` matching with the shared decoder table, which refuses a name it has no decoder for, and `csvxl-encoding-like-core` made `""` and `"auto"` mean detection as in the core readers. Two gaps remained, both found by reviewing this change: the name was checked only while a CSV was being read, so with no file to read (an empty list, or `EachCsvToOneExcel` on an empty directory) `"klingon-1"` was accepted and a workbook written, and with several files the same error was repeated once per file.

## What Changes

On this line the CSV-spelled names and the `ExcelToCSV` functions are added, and the old names become Deprecated. The old names keep working until 0.4 removes them.

| Old name (Deprecated) | New name |
| --- | --- |
| `CsvToExcel` | `CSVToExcel` |
| `AppendCsvToExcel` | `AppendCSVToExcel` |
| `EachCsvToOneExcel` | `CSVDirToExcel` |
| `ReadCsvToString` | `ReadCSVToString` |

- The new names take exactly the parameters the old ones took and do exactly what they did. Each old name keeps its behaviour, and its `Deprecated:` paragraph names its replacement and says it is removed in 0.4.
- `ExcelToCSV` and `ExcelDirToCSV` are new. They take `ExcelToCSVOptions{Sheets, AllowFormulas}` and guard formula-like text by default. The guard decides by spelling, so a plain decimal number is left alone even when a float64 cannot hold it.
- `ExcelToCsv` and `EachExcelToCsv` are Deprecated but are not wrappers of the new functions. They keep their signatures and output, with no formula guard, and their `Deprecated:` paragraphs name `ExcelToCSV` and `ExcelDirToCSV`.
- The doc comments of the seven new names start with their name, and the deprecation notices of the six old names are checked by a test.
- The CLI's `convert` calls `CSVToExcel` for CSV to Excel, and keeps calling `ExcelToCsv` for Excel to CSV, so its output names do not change.

Left on 0.4: the `ExcelToCsvOptions` alias; the package comment starting with `Package csvxl`, and the doc comments of the other exported identifiers; and the check that refuses an unknown encoding name before any file is read. On this line `CSVToExcel` and `AppendCSVToExcel` still read an unknown encoding undecoded, and `ReadCSVToString` refuses it. The `csvxl-batch-failures` behaviour is not carried over either.

## Capabilities

### New Capabilities

- `csvxl-names`: the names of the `csvxl` functions, their doc comments, the deprecated spellings and the formula guard of `ExcelToCSV`.

### Modified Capabilities

None. The existing `csvxl-*` specifications name the old spellings, which keep working as Deprecated wrappers; they are brought up to date when the old names are removed.

## Impact

- `csvxl/convert.go`, `csvxl/convertDir.go`, `csvxl/read_csv.go`; `csvxl/names_test.go` and `csvxl/new_names_behaviour_test.go`; `cli/commands/convert.go`.
- `Docs/csvxl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- Not on this line: `api-review.md` (this repo does not carry it), a `delivery-status.md` entry for this port, and the AGENTS.md follow-up for removing the old names, which 0.4 owns.
