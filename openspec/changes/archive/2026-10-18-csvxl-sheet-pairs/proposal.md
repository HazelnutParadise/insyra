# Proposal: csvxl-sheet-pairs

## Why

`CSVToExcel(csvFiles, sheetNames, ...)` and `AppendCSVToExcel` pair a CSV with its sheet name by position in two slices, and `ExcelToCSV(..., csvNames, ...)` pairs a file name with the i-th converted sheet ([#270](https://github.com/HazelnutParadise/insyra/issues/270), api-review C-11). Measured on `0.4` at ffff6a1d, every case below returned a nil error:

- three CSVs with two names: the third sheet silently took its file name; two CSVs with three names: the third name was dropped;
- two CSVs both named `Sales`: one sheet holding `[feb1] [jan2]`, rows of both files mixed;
- `ExcelToCSV` with `csvNames` and no `Sheets`: names paired with the workbook's internal sheet order, which the caller has to know;
- two equal `csvNames`: one sheet's CSV overwritten by another's;
- the CLI's `convert book.xlsx out.csv` on a three-sheet workbook wrote `out.csv`, `Q2.csv` and `Q3.csv`.

The owner ruled on 2026-10-18: a sheet and its file travel together, and nothing is paired by position.

## What Changes

- **BREAKING**: `CSVToExcel(sheets []SheetFromCSV, output, csvEncoding...)` and `AppendCSVToExcel(sheets []SheetFromCSV, existingFile, csvEncoding...)`. `SheetFromCSV{Path, Sheet}` names one CSV and the sheet it becomes; an empty `Sheet` is the file name, as before.
- **BREAKING**: `ExcelToCSV(excelFile, outputDir, opts...)` drops `csvNames`. `ExcelToCSVOptions.FileNames` maps a sheet name to its CSV file name; a key that is not a converted sheet is an error. `ExcelDirToCSV` refuses `FileNames`, which cannot name files for several workbooks at once.
- Two CSVs aimed at one sheet, and two sheets aimed at one CSV file, are refused before anything is written, compared without regard to case. This also covers `ExcelDirToCSV`, where `a.xlsx`'s sheet `b_c` and `a_b.xlsx`'s sheet `c` both became `a_b_c.csv`.
- `ExcelToCSV` checks every file name before writing the first CSV, so a refused name leaves no partial output.
- New `SheetNames(excelFile)` lists a workbook's sheets in order.
- The deprecated `CsvToExcel`, `AppendCsvToExcel` and `ExcelToCsv` keep their signatures and positional pairing; a duplicate target is now an error there too.
- **BREAKING** (CLI): `convert <file.xlsx> sheet <name> <file.csv> [sheet <name> <file.csv> ...]` pairs each sheet with its file; `convert <file.xlsx> <file.csv>` works only on a one-sheet workbook and otherwise fails, listing the sheets. New command `sheets <file.xlsx>` lists them.

## Capabilities

### New Capabilities

- `csvxl-sheet-pairing`: how the conversions pair a CSV file with a sheet and refuse a pairing that would lose data.

### Modified Capabilities

- `csvxl-file-paths`: the output-name rule moves from `csvNames` to `FileNames`.
- `csvxl-sheet-name-safety`: names are checked before any CSV is written.
- `dsl-commands`: `convert` pairs sheets with files; `sheets` lists them.

## Impact

`csvxl`, `internal/dsl/commands` (`convert`, new `sheets`), `Docs/csvxl.md`, `Docs/cli-dsl.md`, both CHANGELOGs, `api-review.md`, `delivery-status.md`, `AGENTS.md` (follow-up for the dev backport).
