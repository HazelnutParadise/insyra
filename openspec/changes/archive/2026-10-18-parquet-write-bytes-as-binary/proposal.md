# Proposal: parquet-write-bytes-as-binary

## Why

`Write` converted every `[]byte` cell through `conv.ToString`, so a cell holding `A-01` was stored as the text `[65 45 48 49]`. The reader already returns a Binary column as `[]byte`, so a read-then-write round trip destroyed the bytes.

## What Changes

- A column whose cells are all `[]byte` (nil allowed) is written as Parquet `Binary` and reads back as the same bytes.
- A `[]byte` cell in a column that also holds other kinds is written as its text when it is valid UTF-8, and as Go's text for the slice otherwise.
- Not changed: other type downgrades on a round trip (`Date32`, decimal, `Int16`), which would need the source schema carried through a read.

## Capabilities

### Modified Capabilities

- `parquet-write`: see What Changes.

## Impact

`parquet/internal.go`, `parquet/binary_write_test.go`, `parquet/foreign_types_test.go`, `Docs/parquet.md`, both CHANGELOGs, `AGENTS.md` follow-up narrowed.
