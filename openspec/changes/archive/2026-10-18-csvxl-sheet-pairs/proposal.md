# Proposal: csvxl-sheet-pairs (backport)

## Why

`0.4` settled [#270](https://github.com/HazelnutParadise/insyra/issues/270) (api-review C-11) in `csvxl-sheet-pairs` (19a2b854): a CSV and its sheet are named together, never paired by position. `dev` carries `CSVToExcel`, `AppendCSVToExcel` and `ExcelToCSV` unreleased, in the old positional shape. Shipping them that way in v0.3.5 would make a caller moving off the deprecated `Csv` names change their calls twice, once now and again for 0.4. The new names take 0.4's final signatures now; the old names keep their released behaviour.

## What Changes

- `CSVToExcel(sheets []SheetFromCSV, output, csvEncoding...)` and `AppendCSVToExcel(sheets []SheetFromCSV, existingFile, csvEncoding...)`, with `SheetFromCSV{Path, Sheet}`.
- `ExcelToCSV(excelFile, outputDir, opts...)` with `ExcelToCSVOptions.FileNames` (sheet name to file name); a key that is not a converted sheet is an error, and `ExcelDirToCSV` refuses `FileNames`.
- The new names refuse two CSVs for one sheet and two sheets for one CSV file, compared without regard to case and across workbooks in `ExcelDirToCSV`, before anything is written. `ExcelToCSV` checks every file name before writing the first.
- New `SheetNames(excelFile)`.
- CLI: the additive `convert <file.xlsx> sheet <name> <file.csv> ...` form and a `sheets <file.xlsx>` command.
- Unchanged on this line, unlike `0.4`: `CsvToExcel`, `AppendCsvToExcel`, `ExcelToCsv` and `EachExcelToCsv` keep their signatures, positional pairing and output, without the duplicate-target refusal; the CLI's plain `convert <file.xlsx> <file.csv>` keeps writing every sheet.

## Capabilities

### New Capabilities

- `csvxl-sheet-pairing`: how the new conversions pair a CSV file with a sheet and refuse a pairing that would lose data.

### Modified Capabilities

- `csvxl-names`: the new names no longer take the old names' arguments.
- `csvxl-sheet-name-safety`: `ExcelToCSV` and `ExcelDirToCSV` check every name before writing.
- `dsl-commands`: `convert` gains the sheet form; `sheets` lists a workbook's sheets.

## Impact

`csvxl`, `cli/commands` (`convert`, new `sheets`), `cli/repl` completion, `Docs/csvxl.md`, `Docs/cli-dsl.md`, both CHANGELOGs.
