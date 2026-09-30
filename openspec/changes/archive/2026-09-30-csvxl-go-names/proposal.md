# Proposal: csvxl-go-names

## Why

C-12 of the API review, filed under [#212](https://github.com/HazelnutParadise/insyra/issues/212), found that `csvxl` names break Go's initialism convention: Go writes an initialism in one case (`CSV`, `URL`, `ID`), and every other package in insyra already does (`ReadCSVFile`, `ToCSV`, `CSVReadOptions`), while `csvxl` writes `Csv` in all seven of its exported functions and its options type. `EachCsvToOneExcel` also has to be read twice: it puts every CSV file in a directory into one workbook, which "each CSV to one Excel" does not say. And most of the package's doc comments do not start with the name they document (`// Convert multiple CSV files to an Excel file`, `// CsvEncoding Options`, none at all on `ReadCsvToString`), which `go doc` and linters expect.

The owner ruled on #211 that each function has one name, and that an old name stays for one release as a Deprecated wrapper keeping its old meaning. The shape of the functions, including the parallel `csvFiles`/`sheetNames` slices (C-11, #270), was chosen by the owner on 2026-09-26 and is not changed here.

C-6 ([#268](https://github.com/HazelnutParadise/insyra/issues/268)) is already fixed on this line: `fix-api-review-batch-6` replaced the `strings.Contains` matching with the shared decoder table, which refuses a name it has no decoder for, and `csvxl-encoding-like-core` made `""` and `"auto"` mean detection as in the core readers. Two gaps remained, both found by reviewing this change: the name was checked only while a CSV was being read, so with no file to read (an empty list, or `EachCsvToOneExcel` on an empty directory) `"klingon-1"` was accepted and a workbook written, and with several files the same error was repeated once per file.

## What Changes

| Old name (Deprecated) | New name |
| --- | --- |
| `CsvToExcel` | `CSVToExcel` |
| `AppendCsvToExcel` | `AppendCSVToExcel` |
| `ExcelToCsv` | `ExcelToCSV` |
| `ExcelToCsvOptions` | `ExcelToCSVOptions` (the old name is an alias of the same type) |
| `EachCsvToOneExcel` | `CSVDirToExcel` |
| `EachExcelToCsv` | `ExcelDirToCSV` |
| `ReadCsvToString` | `ReadCSVToString` |

- The new functions take exactly the parameters the old ones took and do exactly what they did. Each old name stays one release as a Deprecated wrapper whose doc comment names its replacement.
- `CSVDirToExcel` and `ExcelDirToCSV` name what they take, a directory, and mirror each other; `EachExcelToCsv` gets the directory name too so the pair reads alike.
- Every exported identifier's doc comment starts with its name, the package comment starts with `Package csvxl`, and the encoding constants say what they are for.
- `CSVToExcel` and `AppendCSVToExcel` check the encoding name once, before any file is read or any workbook is opened, so an unknown name is always an error that names it once, and nothing is written for it.
- The CLI's `convert` calls the new names. Its behaviour does not change.
- Removing the old names is recorded as an `AGENTS.md` follow-up.

`UTF8`, `Big5` and `Auto` stay plain string constants. A typed `Encoding` would add nothing now that an unknown name is an error: the names accepted are an open set of about a hundred aliases that no Go type can list, an untyped string literal such as `"klingon"` would still compile into it, and the core readers take the same argument as a plain `string` in `CSVReadOptions.Encoding`, so a `csvxl`-only type would give one argument two types across packages and break every caller passing a string variable.

## Capabilities

### New Capabilities

- `csvxl-names`: the names of the `csvxl` functions, their doc comments and the deprecated spellings.

### Modified Capabilities

None. The existing `csvxl-*` specifications name the old spellings, which keep working as Deprecated wrappers; they are brought up to date when the old names are removed.

## Impact

- `csvxl/*.go` and their tests; `cli/commands/convert.go`.
- `Docs/csvxl.md`, both changelogs, `AGENTS.md` (removal follow-up), `api-review.md` (C-6, C-12), `delivery-status.md`.
