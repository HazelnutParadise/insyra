## Why

`isr.DT.From` and `isr.DL.From` return unexported types, so a caller cannot name an isr table or list in its own function signature. `CSV.OutputOpts` is read by nothing, because isr can read every format through `DT.From` but has no way to write one. The error for a `Row`/`Col` with an unusable key reports `map[interface {}]interface {}` instead of naming the type or the key. This is `0.4`'s `isr-exported-types-and-output`, ported with the option names this line already uses.

## What Changes

- Export the types as `isr.DataTable` and `isr.DataList`. The `DT` and `DL` variables and the embedded field names are unchanged.
- Add `(*isr.DataTable).To(target any) error`, which writes to `CSV`, `JSON` or `Excel`, by value or by pointer. A pointer target without a `FilePath` receives the output in `CSV.String` or `JSON.Bytes`.
- `CSV_outOpts` gains `IncludeBOM`. New `JSON_outOpts{UseColIndex}` and `Excel_outOpts{RowNames2FirstCol, ColNames2FirstRow, IfSheetExists}`, attached as `OutputOpts` fields. They follow the names `CSV_outOpts` already has on this line; `0.4` renames them together.
- `Row`/`Col` key errors name the isr type and the first unusable key.

## Impact

`isr/`, `Docs/isr.md`, both CHANGELOGs. Additive: code outside the package could not name the old types, so no caller breaks.
