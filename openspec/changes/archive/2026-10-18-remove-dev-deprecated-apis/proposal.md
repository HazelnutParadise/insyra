## Why

The Go APIs that 0.3.x deprecated "for one release" are removed in 0.4 (124 symbols: the `Csv` spellings in `csvxl`, the index filters and `Headers` on `DataTable`, the window and fill methods on `DataList`, the `New…` layer constructors and `Float32`-style aliases in `nn`, `WriteONNX`, `RunCodeWithTimeout`, `Diag`, the error-popping helpers and more). The main specs still require those names to "remain for one release" or use them as scenario examples, so they contradict the code.

## What Changes

- **BREAKING** The deprecated names are gone from the code; the requirements that only kept them alive are removed from the specs.
- Requirements that also state the behaviour of a surviving name are modified to drop only the removed name.
- Scenarios that used a removed name as their example now use the surviving one (`CSVToExcel`, `ExcelToCSV`, `ReadJSONFile`, `ReadCSVToString`, and so on).
- Still deprecated and still present, so untouched: `ReadCSV_File`, `ReadCSV_FileWithOptions`, `ReadCSV_String`, `ReadCSV_StringWithOptions`, `ToCSVWithOptions`, `Config.SetDontPanic`, the `cli/repl`, `cli/commands` and `cli/env` names, the accel names, and the `Tape` loss methods.

## Impact

- Specs: 32 capabilities (`csvxl-*`, `datalist-numeric-input`, `datafetch-*`, `nn-*`, `ml-*`, `core-names`, `dsl-commands` and others).
- Code, docs and both changelogs were updated in the same change.
