# [ pd ] Package

Pandas-like helpers for Go using gpandas. The `pd` package provides a thin wrapper around `gpandas.DataFrame` with convenience functions to convert between Insyra's `DataTable` and a gpandas `DataFrame`, and to expose a pandas-like API.

## Table of Contents

- [Overview](#overview)
- [Import](#import)
- [Primary Types & Functions](#primary-types--functions)
- [Examples](#examples)
- [Type Inference & Notes](#type-inference--notes)

## Overview

The `pd` package is a small interoperability layer between Insyra's `DataTable` and `gpandas` data frames. It allows conversion back and forth while preserving column ordering and optional row names (index).

## Import

```go
import "github.com/HazelnutParadise/insyra/pd"
```

You will also need `gpandas` as a dependency; the package imports `github.com/apoplexi24/gpandas` internally.

> [!NOTE]
> For the full `gpandas` API reference and usage examples, please see: https://gpandas.apoplexi.com/docs/

## Primary Types & Functions

- `type DataFrame struct { *gpdf.DataFrame }` — wrapper around `gpandas.DataFrame`.

- `func FromDataTable(dt *insyra.DataTable) (*DataFrame, error)`
  - Converts an `insyra.DataTable` into a `pd.DataFrame`.
  - Column types are inferred per-column (int/float/bool/string) and fall back to `any` when mixed.
  - Preserves row names as the DataFrame index when present.

- `func (t *DataFrame) ToDataTable() (*insyra.DataTable, error)`
  - Converts a wrapped `gpandas.DataFrame` back to an `insyra.DataTable`.
  - Column order and values are preserved. Index becomes row names when present.

- `func FromGPandasDataFrame(df *gpdf.DataFrame) (*DataFrame, error)`
  - Wraps an existing `gpandas.DataFrame` into `pd.DataFrame`.

## Examples

```go
// Convert DataTable -> gpandas DataFrame
dt := insyra.NewDataTable(
    insyra.NewDataList("Alice", "Bob").SetName("name"),
    insyra.NewDataList(30, 25).SetName("age"),
)
df, err := pd.FromDataTable(dt)
if err != nil {
    log.Fatal(err)
}

// Work with df (gpandas API) then convert back
newDt, err := df.ToDataTable()
if err != nil {
    log.Fatal(err)
}
```

## Type Inference & Notes

- `pd.FromDataTable` inspects each column and returns one of: `int`, `float`, `bool`, `string`, or `any`.
- Integer values are normalized to `int64`, floats to `float64`.
- `nil` values are preserved where present.
- For empty `DataTable` (no columns) an empty `gpandas.DataFrame` is created and returned successfully.
