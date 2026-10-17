## Why

api-review I-2, I-3 and I-5 (#237, #238, #239). `isr.DT.From` and `isr.DL.From` return unexported types, so a caller cannot name an isr table or list in its own function signature. `CSV.OutputOpts` is read by nothing, because isr can read every format through `DT.From` but has no way to write one. The error for a `Row`/`Col` with an unusable key reports `map[interface {}]interface {}` instead of naming the type or the key.

## What Changes

- Export the types as `isr.DataTable` and `isr.DataList`. The `DT` and `DL` variables and the embedded field names are unchanged.
- Add `(*isr.DataTable).To(target any) error`, which writes to `CSV`, `JSON` or `Excel`, by value or by pointer. A pointer target without a `FilePath` receives the output in `CSV.String` or `JSON.Bytes`.
- `CSV_outOpts` gains `IncludeBOM`. New `JSON_outOpts{UseColIndex}` and `Excel_outOpts{NoHeaderRow, HasRowNames, IfSheetExists}`, attached as `OutputOpts` fields.
- `Row`/`Col` key errors name the isr type and the first unusable key.
- Kept by ruling: `From(any)`, `map[any]any` for `Row`/`Col`, and `Pivot`/`Unpivot` returning only the isr table.

## Impact

- `isr/`, `Docs/isr.md`, both CHANGELOGs, `api-review.md`, `delivery-status.md`.
- Additive. Code outside the package could not name the old types, so no caller breaks.
