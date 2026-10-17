# Proposal: review-fixes-core-py-datafetch

## Why

A review of the 0.4 port against 0.3.5 found defects that had been copied along with the code, and doc claims that did not match the code. Commit `76a9b4c1` fixes them, each with a test:

- `Filter`, `FilterRows` and `FilterRowsWhere` shared the table's column slice with the predicate, so a predicate that dropped a column shifted the columns being read (`TestFilterSurvivesDropColsInPredicate`, `TestFilterRowsSurvivesDropColsInPredicate`, `TestFilterRowsWhereSurvivesDropColsInKeep`).
- `SliceCols` returned a selected column shorter than the table's longest column as it was, so the slice lost the rows past its length (`TestSliceColsPadsShortColumns`).
- `FilterColsByColIndexLessThanOrEqualTo` overflowed on the largest column letter `ParseColIndex` reads and panicked (`TestFilterColsLessThanOrEqualToMaxParsedIndex`).
- `DataType` counted only a `float64` NaN as missing (`TestDataListDataType`), and `ShowTypes` cut the `DataType` word to `num` in a narrow column (`TestShowTypesKeepsCompleteDataTypeInNarrowColumns`).
- `py`: after the IPC accept loop stopped, the next run was given the dead listener's address (`TestStoppedAcceptLoopInvalidatesSharedListener`). Result field names were folded by a private rule that did not match `strings.EqualFold` (`TestAssignStructUsesEqualFoldForTurkishI`).
- `datafetch`: `ReverseColsContext` returned no error for a context that had already ended when the batch was empty or invalid (`TestReverseColsContextReportsCancellationBeforeEmptyOrInvalidBatch`).
- Docs: `FilterRowsWhere`'s row shares slice and map cells, which `Docs/DataTable.md` did not say. The CCL docs said a registration under a built-in's name replaces it whatever its kind, and that a scalar function is called for each row. The registry keeps one map per kind and checks sequence, then aggregate, then scalar functions (`internal/ccl/ccl_functions.go`, `internal/ccl/ccl_evaluator.go`).

## What Changes

- `Filter`, `FilterRows` and `FilterRowsWhere` build their result from a copy of the column list taken before the callback runs. A column dropped meanwhile no longer changes what is read.
- `SliceCols` pads each selected column shorter than the table's longest column with `nil`, so the result keeps every row. The source table is not changed.
- `FilterColsByColIndexLessThanOrEqualTo` compares the index before adding one, so the largest parsed letter keeps every column and does not panic.
- `DataList.DataType()` treats a NaN of any floating-point type, including `float32` and named float types, as missing.
- `ShowTypes` sizes a column so its `DataType` word prints whole.
- `py`: the IPC server forgets a listener whose accept loop stopped, so the next run opens a new server. Result field names match with `strings.EqualFold`, and the first field in declaration order that matches takes the key.
- `datafetch`: `ReverseColsContext` checks the context after its input checks, so an already-ended context returns its error for an empty or invalid batch. `ReverseTableContext` inherits this once its columns resolve.
- Documentation only: `Docs/DataTable.md`, `Docs/DataList.md`, `Docs/py.md`, `Docs/CCL.md`, and the doc comments of `engine/ccl`. Both changelogs gain entries under Core, `py` and `datafetch`.

No breaking changes are declared, and the changelog entries are not marked BREAKING. The behaviour changes are listed here so a reader can judge that:

- A `SliceCols` on a table with columns of different lengths now returns every row, padded with `nil`, where it returned fewer.
- A `float32` or named-float NaN is now missing for `DataType`, where it counted as a number.
- An explicit DataFrame name is kept when the DataFrame also has a column called `name`, where it was dropped.
- A key `İ` no longer matches a struct field `I`, where the old folding rule matched it.
- An already-ended context makes `ReverseColsContext` return its error for an empty or invalid batch, where it returned nil.
- A predicate that drops a column gets the columns it was given at the call's start, where the old code read shifted or panicked.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `datatable-filtering`: a predicate that drops a column does not shift the result; `FilterRowsWhere`'s row copy is stated with its shared slice and map cells.
- `datatable-slicing`: `SliceCols` pads short columns with `nil`.
- `datatable-bounds-and-aliasing`: the largest parsed column index does not overflow.
- `data-type`: a NaN of any float type is missing; `ShowTypes` keeps the `DataType` word whole.
- `py-ipc-server`: a run after the accept loop stopped opens a new server.
- `py-run-api`: result field names match by `strings.EqualFold`.
- `datafetch-cancellation`: an ended context is reported before an empty or invalid batch.
- `ccl-function-registry`: the documented replacement rule is per function kind, with the evaluation order.

## Impact

- Code: `datatable_filters.go`, `datalist_datatype.go`, `show.go`, `datafetch/geocoding.go`, `py/builtin.go`, `py/pyresult.go`, `py/pyresult_decode.go`.
- Tests: `datatable_filter_rows_where_test.go`, `datatable_slice_test.go`, `datalist_datatype_test.go`, `showtypes_datatype_test.go`, `datafetch/geocoding_context_test.go`, `py/builtin_test.go`, `py/environment_setup_test.go`, `py/ipc_server_test.go`, `py/large_integer_test.go`.
- Docs: `Docs/DataTable.md`, `Docs/DataList.md`, `Docs/py.md`, `Docs/CCL.md`, `engine/ccl/ccl.go` comments, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- No public signature changes. No new dependencies.
