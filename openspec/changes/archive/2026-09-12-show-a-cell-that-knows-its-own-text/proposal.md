# Proposal: show-a-cell-that-knows-its-own-text

## Why

`parquet-foreign-column-types` made a Parquet `Decimal128` column read as a `decimal.Decimal`, and said it "displays through its `String()`". Running it shows that it does not. Measured on 2026-09-12 on a three-row table read from a file written by another tool:

| Surface | What a decimal cell produces | Right? |
| --- | --- | --- |
| `ToCSV` | `10.50` | yes |
| `fmt.Printf("%v")` on the cell | `10.50` | yes |
| Sorting | by value | yes |
| **`Show()`** | `<decimal.Decimal>` | **no** |
| **`ToJSON`** | `{}` | **no** |

Both have the same cause, and neither is specific to decimals.

- `utils.FormatValue` ends in a `reflect.Struct` branch that returns `<TypeName>` without ever asking whether the value can print itself. `time.Time` escapes it only because it has an explicit case higher up. A decimal is the first struct cell type to reach that branch, which is how the gap surfaced.
- `buildJSONRows` puts the raw cell into the map and lets the marshaller deal with it. A `decimal.Decimal` holds a `big.Int` and an `int32`, both unexported, so it marshals to `{}` — the value is gone from the export with nothing to say so. `time.Time` escapes this one because it implements `json.Marshaler`.

## What Changes

- `FormatValue` asks a value whether it knows its own text before falling back to `<TypeName>`: a `fmt.Stringer` is displayed as its `String()`. Everything with an explicit case above — floats, integers, `bool`, `string`, `[]byte`, `time.Time` — is unaffected, as are slices and maps.
- `buildJSONRows` converts a cell that implements `fmt.Stringer` but neither `json.Marshaler` nor `encoding.TextMarshaler` to its text. The two marshaller interfaces are the test for "this type already knows how to serialise itself", which is exactly what keeps `time.Time` on its RFC 3339 form rather than Go's `2024-01-01 00:00:00 +0000 UTC`.

## Capabilities

### New Capabilities

- `cell-display-and-export`: what a cell that is neither a primitive nor a known type shows and exports as.

### Modified Capabilities

(none)

## Impact

- A struct cell with a `String()` method changes from `<pkg.Type>` to its text in every display path, and from `{}` to its text in JSON. Nothing in the library produces such a cell except the Parquet decimal, so in practice this is the decimal; a caller who put their own struct in a cell gets a better rendering than they had.
- Not changed: `Mean` on a decimal column returns `NaN` with no error, because the numeric path skips a cell it cannot read and the column is entirely such cells. A `time.Time` column behaves identically, measured side by side. That is the pre-existing shape for a non-numeric column, not something decimals introduced, and whether it should report instead of returning `NaN` is recorded in `AGENTS.md` rather than decided here.
- The archived `parquet-foreign-column-types` proposal said the numeric path would "refuse a decimal column, naming the row". It does not; it returns `NaN` silently. The record is corrected.

## Backport to dev (0.3.x)

Dev received: `FormatValue` asking a `fmt.Stringer` before its struct fallback, so every display path (`Show`, `ShowRange`, the table layout) prints such a struct as its text; the display test, plus the `time.Time` JSON test, which pins behaviour this line already has; the changelog entry, display only; the correction to the archived `parquet-foreign-column-types` proposal, reworded to what this line measures (the numeric path skips a decimal cell with a warning on `Err()`, and an all-decimal list gives `NaN`).

Left on 0.4:
- `jsonCell` in `buildJSONRows`: breaking, it changes `ToJSON` output for values that already export, such as a `time.Duration` (a number today, its text after), a `fmt.Stringer` struct with exported fields, or a named number with `String()`.
- The JSON requirement, its test and the changelog sentence on JSON: they describe the excluded code.
- The `Docs/parquet.md` note on exporting a binary column to JSON: it describes 0.4's binary-as-string cells.
- The AGENTS.md follow-up rewording: this line's entry already says the numeric path treats a decimal as not a number, and its measured behaviour differs from 0.4's (`Err()` is set, not nil).
