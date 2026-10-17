# Proposal: read-write-names

## Why

Part of #213, ported from `0.4` without its breaking parts. The JSON reader and writers were spelled with an underscore (`ReadJSON_File`, `ToJSON_Bytes`, `ToJSON_String`), which Go names do not use and which sat beside underscore-free names such as `ReadJSON`.

## What Changes

- `ReadJSONFile`, `DataTable.ToJSONBytes` and `DataTable.ToJSONString` replace the underscore names. Each old name stays as a Deprecated wrapper with its old meaning and is removed in 0.4. `IDataTable` gains the two methods. Callers in the module moved to the new names.
- **Left on `0.4`:** the CSV names (`ReadCSVFile`, `ReadCSVString`, `ToCSV` taking an options struct) and the rename of `FirstRow2ColNames` to `NoHeaderRow`, which reverses the meaning of an options struct's zero value.

## Capabilities

### New Capabilities

- `read-write-names`: the JSON reader and writer names.

### Modified Capabilities

None.

## Impact

- `read.go`, `datatable_json.go`, `interfaces.go`, `isr/dt.go`, `cli/commands/load.go`, `Docs/DataTable.md`, both changelogs and tests.
