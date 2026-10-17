# Tasks: review-fixes-core-py-datafetch

## 1. Core tables

- [x] 1.1 `datatable_filters.go`: `Filter`, `FilterRows` and `FilterRowsWhere` take a copy of the column list before their callback runs
- [x] 1.2 `datatable_filters.go`: `SliceCols` pads each selected column shorter than the table's longest column with `nil` (`padColumnsToLength`), without changing the source
- [x] 1.3 `datatable_filters.go`: `FilterColsByColIndexLessThanOrEqualTo` compares the index before adding one, so the largest parsed letter does not overflow
- [x] 1.4 `datatable_filter_rows_where_test.go`: `TestFilterRowsWhereSurvivesDropColsInKeep`, `TestFilterSurvivesDropColsInPredicate`, `TestFilterRowsSurvivesDropColsInPredicate`
- [x] 1.5 `datatable_slice_test.go`: `TestSliceColsPadsShortColumns`, `TestFilterColsLessThanOrEqualToMaxParsedIndex`
- [x] 1.6 `Docs/DataTable.md`: Slicing (`SliceCols` pads with `nil`) and `FilterRowsWhere` (the row's cell slice is a copy, reference values are shared)

## 2. DataType and ShowTypes

- [x] 2.1 `datalist_datatype.go`: `isDataTypeMissing` treats a `NaN` of `float32` or of a named float type as missing
- [x] 2.2 `show.go`: `prepareTableLayoutTypes` widens each column to at least the width of `DataType`, which is at least as wide as every `DataType` word
- [x] 2.3 `datalist_datatype_test.go`: float32 and named-float NaN cases
- [x] 2.4 `showtypes_datatype_test.go`: `TestShowTypesKeepsCompleteDataTypeInNarrowColumns`
- [x] 2.5 `Docs/DataList.md`: `DataType` description names NaN of any floating-point type

## 3. py

- [x] 3.1 `py/pyresult.go`: `acquireIPCServer` opens a server when `serverLn` is nil; `releaseIPCServer` handles a nil listener; `acceptIPC` clears the registered listener when its accept loop stops
- [x] 3.2 `py/pyresult_decode.go`: struct fields match by exact name, then by `strings.EqualFold` in declaration order; the folded map and `foldName` are removed
- [x] 3.4 `py/ipc_server_test.go`: `TestStoppedAcceptLoopInvalidatesSharedListener`
- [x] 3.5 `py/large_integer_test.go`: `TestAssignStructUsesEqualFoldForTurkishI`

## 4. datafetch

- [x] 4.1 `datafetch/geocoding.go`: `ReverseColsContext` returns `ctx.Err()` after its nil and length checks, before any request
- [x] 4.2 `datafetch/geocoding_context_test.go`: `TestReverseColsContextReportsCancellationBeforeEmptyOrInvalidBatch`

## 5. CCL documentation

- [x] 5.1 `engine/ccl/ccl.go`: doc comments of `RegisterFunction`, `RegisterAggregateFunction` and `RegisterSequenceFunction` state the replacement rule per kind and the evaluation order
- [x] 5.2 `Docs/CCL.md`: sequence functions description, scalar registration (row-independent calls), and the registry's replacement and evaluation order

## 6. Changelogs

- [x] 6.1 `CHANGELOG.md`: entries under Core, `py` and `datafetch` under `## Unreleased`
- [x] 6.2 `CHANGELOG_TW.md`: the same entries under `## Unreleased`
