## Why

Nothing lets a program ask what kind of values a column holds. `ShowTypes` prints the Go type of every cell for a person to read, and the library's own code answers the question privately several times over (the table summary, imputation, `Describe`, Parquet and CSV type inference), each by its own rule. The owner asked for one public answer on 2026-09-25 while deciding how the table-level fills should report a column they cannot fill (#213, E-7), and chose the names `DataType()` / `ColDataTypes()` over `ValueKind` / `ValueType` because "data type" is the word Excel, SQL and pandas users already know.

## What Changes

- New `DataType` with `DataTypeEmpty`, `DataTypeNumber`, `DataTypeString`, `DataTypeBool`, `DataTypeTime`, `DataTypeOther` and `DataTypeMixed`, and a `String()` giving `"number"`, `"mixed"`, ...
- `(*DataList).DataType()` judges the list over its values with missing ones (`nil`, `NaN`) left out. Every int, uint and float width, a named type over one, and a decimal are all numbers; text that looks numeric stays text.
- `(*DataTable).ColDataTypes()` returns each column's data type in column order, beside `ColNames()`.
- `ShowTypes` opens with a `DataType` row (a `DataType:` line for a DataList) showing the same verdict above the per-cell Go types, which the owner asked for on seeing how the two differ.
- The private classifiers are not rewritten in this change.

## Capabilities

### New Capabilities
- `data-type`: what kind of values a column holds.

### Modified Capabilities
None.

## Impact

- `datalist_datatype.go` (new), `interfaces.go`.
- `Docs/DataList.md`, `Docs/DataTable.md`, `skills/insyra/SKILL.md`, both changelogs.
