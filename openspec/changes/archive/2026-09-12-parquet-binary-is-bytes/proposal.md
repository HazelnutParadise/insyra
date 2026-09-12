# Proposal: parquet-binary-is-bytes

## Why

`parquet-foreign-column-types` made a `Binary` column read as a `string` holding the raw bytes, because a `DataList` cell could not be a slice and `[]byte` in a cell did not work: it could not be counted, it was never found by a search, and `Counter` panicked on it.

Both reasons are gone. `identify-uncomparable-cells` made a slice cell work — counted, matched, grouped and ordered by content — and `one-value-one-cell` gave the constructor a way to hold one whole.

What is left is the defect the string representation causes: **a `Binary` column and a `String` column are indistinguishable.** Measured on 2026-09-12, writing both with the same bytes and reading them back:

```
String column → string "A-01"
Binary column → string "A-01"     same Go type, equal values
```

So a reader cannot tell which column held text and which held bytes, and a read-then-write round trip turns the binary column into a string one. The display is worse: since a string that is not valid UTF-8 renders as hex, a `Binary` column shows `'A-01'` on one row and `00ff41` on the next — **the rendering follows the data rather than the column**.

## What Changes

- `Binary`, `LargeBinary` and `FixedSizeBinary` read as `[]byte`. `LargeString` stays a `string`.
- The two places that build a column from a `[]any` append the values instead of passing the slice to `NewDataList`, so a `[]byte` cell is not flattened into its bytes. For every other value this is identical: flattening a `[]any` of scalars already produces one cell per element.
- `inferArrowType` and `appendValue` write a `[]byte` column back as Arrow `Binary`, so a read-then-write round trip keeps the column's type.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `parquet-column-types`: a binary column reads as bytes, and writes back as binary.

## Impact

- A `Binary` column's cells change from `string` to `[]byte`. `[]byte(cell)` gave the bytes before and `cell` gives them now; a caller asserting `.(string)` has to change, which is the point — the two column types were indistinguishable and now are not.
- `Show` renders the whole column as hex, from `FormatValue`'s existing `[]byte` case, so the rendering follows the column rather than whether a particular row happened to be valid UTF-8.
- Writing a table that holds `[]byte` cells now produces a `Binary` column rather than a string one. A table that never holds one is unaffected.
- JSON export of a binary column changes from a UTF-8-replaced string to base64, which is what `encoding/json` does with `[]byte` and is lossless where the string form was not.
