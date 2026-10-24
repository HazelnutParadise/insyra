## Why

The CLI's `load` reads Excel and `convert` turns CSV into a workbook, but `save <var> report.xlsx` fails with `unsupported output file type` (#329, CLI-21). The library has no way to write a DataTable to Excel at all: `ReadExcelSheet` and `ReadExcel` read, nothing writes.

A workbook is not a CSV. It holds many sheets, so saving to one that already exists raises two questions a CSV never does: what the sheet is called, and what happens to the sheets already there. The owner ruled on 2026-09-25 that a save names its sheet (`Sheet1` unless told otherwise) and touches only that sheet: it is added if absent, and if present the save refuses unless told to replace it — the same `if-exists` rule `save … sql` already follows, so a workbook's other sheets can never be lost to a save.

## What Changes

- `(*DataTable).ToExcel(path, ExcelWriteOptions)` writes the table as one sheet of the workbook at `path`, creating the workbook if needed and leaving every other sheet as it was. `ExcelWriteOptions` names the sheet (default `Sheet1`), whether column names and row names are written (`SetColNamesToFirstRow`, `SetRowNamesToFirstCol`), and `IfSheetExists`: `SheetExistsFail` (the zero value) refuses with `ErrSheetExists` and leaves the file byte-for-byte unchanged, and `SheetExistsReplace` replaces that sheet in its original position.
- Replacing a sheet shares `internal/excelsheet.Replace` with `csvxl`: it preserves hidden state, avoids collisions with placeholder names, and does not move defined names belonging to other sheets.
- An existing file keeps its permissions; a new file is created with permissions governed by umask.
- `(*DataTable).WriteExcel(w, ExcelWriteOptions)` writes a one-sheet workbook to any destination, matching the other formats' writer entry points.
- The file is written through a temporary file and renamed into place, like `ToCSV` and `ToJSON`.
- CLI: `save <var> <file.xlsx> [sheet <name>] [if-exists fail|replace] [headers true|false] [rownames true|false]`. `.xls`, the legacy binary format, is refused with a message naming `.xlsx`. `sheet` and `if-exists` are refused for other formats.

Left on 0.4: read-write-names (renames, `NoHeaderRow`, `HasRowNames`), one value per optional parameter, CSV formula guard on by default, and exported function variables becoming functions.

## Capabilities

### New Capabilities
- `excel-write`: how a DataTable is written into a workbook, and what an existing workbook keeps.

### Modified Capabilities
None.

## Impact

- New `datatable_excel.go`; `interfaces.go`; `cli/commands/save.go`; `internal/excelsheet/replace.go`.
- `Docs/DataTable.md`, `Docs/cli-dsl.md`; both changelogs.
