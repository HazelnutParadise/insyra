# Proposal: exported-functions-are-functions

## Why

Part of #211, ported from `0.4`. `ReadSlice2D` was a package variable holding `Slice2DToDataTable`, so a caller could reassign it and the documentation listed two names for one function.

## What Changes

- `ReadSlice2D` is a function declaration and holds the implementation. `Slice2DToDataTable` is a Deprecated wrapper that returns what `ReadSlice2D` returns, and is removed in 0.4. Callers in the module moved to `ReadSlice2D`.
- **Left on `0.4`:** `ToFloat64`, `ToFloat64Safe` and `mkt.CAI` also become function declarations there, and the module-wide test that no exported variable holds a function. Changing them can break code that assigns to or takes the address of the variable.

## Capabilities

### New Capabilities

- `read-slice2d-name`: the name of the 2D slice reader.

### Modified Capabilities

None.

## Impact

- `read.go`, `isr/dt.go`, `py/pyresult_decode.go`, `Docs/DataTable.md`, both changelogs and tests.
