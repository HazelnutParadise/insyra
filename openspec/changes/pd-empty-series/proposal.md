# Proposal: pd-empty-series

## Why

PD-1 of the API review ([#256](https://github.com/HazelnutParadise/insyra/issues/256)):

- `pd.FromDataList` returns the error `empty DataList` for a list with no elements, while pandas builds an empty Series (`pd.Series([])`, dtype `object`). A caller converting whatever a filter left behind has to handle zero rows separately for this one call. `FromDataTable` already turns a table with no columns into an empty DataFrame, so the two converters also disagree with each other.
- `pd` is a thin wrapper around `github.com/apoplexi24/gpandas`, and neither the code nor `Docs/pd.md` says so plainly. `DataFrame` embeds gpandas's `*dataframe.DataFrame`, `Series` embeds gpandas's `collection.Series`, and `FromGPandasDataFrame` and `FromGPandasSeries` take those types. Every method a caller uses on the result is gpandas's, and it changes when insyra moves to another gpandas version.

## What Changes

- **BREAKING (behaviour)**: `FromDataList` on a list with no elements returns an empty `*Series` and a nil error. The series holds gpandas's any-typed series (`collection.AnySeries`, `DType` `interface {}`), which matches pandas' `object` dtype for an empty Series. Code that relied on the error to detect an empty list gets a series of length 0 instead. A nil list is still an error.
- `FromDataList` reads the list once. It used to read `Len()` and then `Data()`, two separate reads that a writer could land between.
- The doc comments of `DataFrame`, `Series`, `FromGPandasDataFrame` and `FromGPandasSeries`, and `Docs/pd.md`, say which gpandas type each one embeds or takes, that the promoted methods are gpandas's, and that gpandas's version is the one insyra's `go.mod` requires.

## Capabilities

### New Capabilities

- `pd-conversions`: what `pd`'s converters return for an empty or nil input, and how the package describes its gpandas types.

### Modified Capabilities

(none)

## Impact

- Code: `pd/series.go`, `pd/dataframe.go` (doc comments only).
- Tests: `pd/pd_test.go`, run first against the old code.
- Docs: `Docs/pd.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md` (PD-1), `delivery-status.md`. `skills/insyra/SKILL.md` points readers to `pd.md` and teaches no `pd` behaviour, so it does not change.
