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

- `func FromDataTable(dt insyra.IDataTable) (*DataFrame, error)`
  - Converts an object implementing `insyra.IDataTable` into a `pd.DataFrame`.
  - Column types are inferred per-column (int/float/bool/string) and fall back to `any` when mixed.
  - Preserves row names as the DataFrame index when present. Uses `DataTable.AtomicDo` to read a consistent snapshot.

- `func (t *DataFrame) ToDataTable() (*insyra.DataTable, error)`
  - Converts a wrapped `gpandas.DataFrame` back to an `insyra.DataTable`.
  - Column order and values are preserved. Index becomes row names when present.

- `func FromGPandasDataFrame(df *gpdf.DataFrame) (*DataFrame, error)`
  - Wraps an existing `gpandas.DataFrame` into `pd.DataFrame`.

- `type Series struct { gpdc.Series }` — wrapper around `gpandas` collection `Series`.

- `func FromDataList(dl insyra.IDataList) (*Series, error)`
  - Creates a `pd.Series` from an `insyra.DataList`.
  - Infers element type across the list: `int` (normalized to `int64`), `float` (`float64`), or `string`. If types are mixed or unknown, falls back to `any`.
  - Returns an error for `nil` or empty `DataList`.

- `func FromGPandasSeries(gpds gpdc.Series) (*Series, error)`
  - Wraps an existing `gpandas` series into `pd.Series`.

- `func (s *Series) ToDataList() (*insyra.DataList, error)`
  - Converts a `pd.Series` back to an `insyra.DataList`, copying values and preserving `nil`s.

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

// Convert DataList -> pd.Series -> back to DataList
s, err := pd.FromDataList(insyra.NewDataList(1, 2, 3))
if err != nil {
    log.Fatal(err)
}
dl, err := s.ToDataList()
if err != nil {
    log.Fatal(err)
}
```
## Type Inference & Notes

- `pd.FromDataTable` inspects each column and returns one of: `int`, `float`, `bool`, `string`, or `any`.
- Integer values are normalized to `int64`, floats to `float64`.
- `nil` values are preserved where present.
- For empty `DataTable` (no columns) an empty `gpandas.DataFrame` is created and returned successfully.

**Series notes:**

- `pd.FromDataList` inspects elements in the `DataList` and returns a `Series` of one of: `int` (normalized to `int64`), `float` (`float64`), `string`, or `any` (when mixed/unrecognized).
- Mixed element types produce an `any`-typed series.
- `pd.FromDataList` returns an error for `nil` or empty `DataList`.
- `nil` values inside a `DataList` are preserved when converting to a `Series` and back via `ToDataList()`.
