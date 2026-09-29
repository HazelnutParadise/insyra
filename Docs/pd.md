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

`pd` is built on [gpandas](https://github.com/apoplexi24/gpandas) (`github.com/apoplexi24/gpandas`), a separate Go module that `go get` fetches along with insyra, at the version insyra's `go.mod` requires. `pd`'s two types are gpandas types with insyra's converters added:

- `pd.DataFrame` embeds `*dataframe.DataFrame` from `github.com/apoplexi24/gpandas/dataframe`.
- `pd.Series` embeds `collection.Series` from `github.com/apoplexi24/gpandas/utils/collection`.

Every method you call on them other than `ToDataTable` and `ToDataList` is gpandas's, so what it does and its signature follow gpandas and can change when insyra moves to a newer gpandas version. `FromGPandasDataFrame` and `FromGPandasSeries` take gpandas values, so code that calls them imports gpandas itself.

> [!NOTE]
> For the full `gpandas` API reference and usage examples, please see: https://gpandas.apoplexi.com/docs/

## Primary Types & Functions

- `type DataFrame struct { *dataframe.DataFrame }` — embeds gpandas's `*dataframe.DataFrame` (`github.com/apoplexi24/gpandas/dataframe`).

- `func FromDataTable(dt insyra.IDataTable) (*DataFrame, error)`
  - Converts an object implementing `insyra.IDataTable` into a `pd.DataFrame`.
  - Column types are inferred per-column (int/float/bool/string) and fall back to `any` when mixed.
  - Preserves row names as the DataFrame index when present. Uses `DataTable.AtomicDo` to read a consistent snapshot.

- `func (t *DataFrame) ToDataTable() (*insyra.DataTable, error)`
  - Converts a wrapped `gpandas.DataFrame` back to an `insyra.DataTable`.
  - Column order and values are preserved. Index becomes row names when present.

- `func FromGPandasDataFrame(df *dataframe.DataFrame) (*DataFrame, error)`
  - Wraps a data frame built with gpandas into `pd.DataFrame`. A `nil` data frame is an error.

- `type Series struct { collection.Series }` — embeds gpandas's `collection.Series` interface (`github.com/apoplexi24/gpandas/utils/collection`).

- `func FromDataList(dl insyra.IDataList) (*Series, error)`
  - Creates a `pd.Series` from an `insyra.DataList`.
  - Infers element type across the list: `int` (normalized to `int64`), `float` (`float64`), or `string`. If types are mixed or unknown, falls back to `any`.
  - An empty `DataList` gives an empty `any`-typed `Series` (length 0), as pandas gives an empty Series of dtype `object` for `pd.Series([])`. A `nil` `DataList` is an error.

- `func FromGPandasSeries(gpds collection.Series) (*Series, error)`
  - Wraps a series built with gpandas into `pd.Series`. A `nil` series is an error.

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
- `pd.FromDataList` turns an empty `DataList` into an empty `any`-typed `Series` and returns an error for a `nil` one.
- `nil` values inside a `DataList` are preserved when converting to a `Series` and back via `ToDataList()`.
