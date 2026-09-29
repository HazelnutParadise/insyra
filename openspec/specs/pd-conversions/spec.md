# pd-conversions Specification

## Purpose
What `pd`'s converters return for an empty or nil input, matching pandas where pandas has an answer, and how the package states that its types are gpandas types, whose methods follow the gpandas version insyra requires.
## Requirements
### Requirement: An empty list converts to an empty Series

`pd.FromDataList` SHALL return a `*Series` of length 0 and a nil error for a list with no elements. The series SHALL be gpandas's any-typed series, whose `DType` is `interface {}`, the counterpart of pandas' `object` dtype for an empty Series. `FromDataList` SHALL still return an error for a nil list. It SHALL read the list's elements once.

#### Scenario: Empty list
- **WHEN** `pd.FromDataList(insyra.NewDataList())` is called
- **THEN** it returns a non-nil `*Series` whose `Len()` is 0 and whose `DType()` is the type of `interface {}`, and a nil error

#### Scenario: Empty Series back to a list
- **WHEN** the empty series from `FromDataList` is converted with `ToDataList`
- **THEN** it returns a `*insyra.DataList` whose `Len()` is 0 and a nil error

#### Scenario: Nil list
- **WHEN** `pd.FromDataList(nil)` is called
- **THEN** it returns a nil `*Series` and an error

### Requirement: pd names the gpandas types it exposes

The doc comments of `pd.DataFrame`, `pd.Series`, `pd.FromGPandasDataFrame` and `pd.FromGPandasSeries` SHALL name the `github.com/apoplexi24/gpandas` type each one embeds or takes, and SHALL say that the methods promoted through the embedding belong to gpandas at the version insyra's `go.mod` requires. `Docs/pd.md` SHALL state the same.

#### Scenario: Reading the Series documentation
- **WHEN** a reader runs `go doc github.com/HazelnutParadise/insyra/pd.Series`
- **THEN** the comment names gpandas's `collection.Series` as the embedded type and says its methods come from gpandas

