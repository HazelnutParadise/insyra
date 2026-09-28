# Proposal: window-failures-carry-the-error

## Why

Finding D-18 of the API review ([#224](https://github.com/HazelnutParadise/insyra/issues/224)) observed that the `Rolling`, `EWM` and `Expanding` reducers return an empty list when the options are invalid, and asked for a list of the source's length filled with `nil` instead, so that appending the result to a table would not fail a second time on the length.

That request contradicts a recorded rule. `make-errors-non-terminating` settled the shape of a failed transform: it returns "an empty, usable list carrying the error (the receiver records it too)" (the `chainable-never-nil` spec, the `Two error shapes` convention in `AGENTS.md`, and the Core entry of the 0.4 changelog). A same-length list of `nil` would be indistinguishable from a window that simply had too few observations everywhere, which is the more dangerous failure: it goes into a table without complaint.

Measured against that rule, the reducers break the half of it that matters. The empty list they return does not carry the error, so a caller checking the result sees nothing:

- `NewDataList(1.0, 2.0).Rolling(RollingOptions{Window: 0}).Mean().Err()` is `nil`; only the source list recorded it.
- The same holds for `MinObs > Window`, `Weights` of the wrong length, `Apply(nil)`, `Corr`/`Cov`/`Beta` with a `nil` list, and `EWM` without exactly one decay parameter.
- `DataTable.RollingCol` and `EWMCol` run on a private snapshot of the column, so an invalid option is recorded on the snapshot and on nothing the caller holds: `dt.RollingCol("A", RollingOptions{Window: 0}).Mean()` returns an empty list, `dt.Err()` stays `nil`, and the only trace is a log line.
- `DataTable.ShiftCol`, `DiffCol`, `PctChangeCol` and the four `Cum*Col` return a bare `NewDataList()` for a column that is not there; the table records the error but the result does not.
- `GroupedColumnTransform.As` returns an empty list without the error it records on the table. Inside a grouped transform, an invalid option (`RollingCol` with `Window: 0`, `DiffCol` with `periods` 0, `ShiftCol` with two fill values) fails once per group on a throwaway sub-list, and `As` scatters the empty results into a full-length column of `nil` that carries no error and leaves the table's `Err()` empty: exactly the silent all-`nil` column D-18 wanted to avoid.
- The CLI's `rolling` does not check the result, so `rolling x 0 mean` prints `saved as $result` and stores an empty list.

## What Changes

- Every reducer of `RollingDataList`, `EWMDataList` and `ExpandingDataList` whose view could not be built returns an empty list carrying the error, as `failedResult` does for the other transforms. `Apply(nil)` and `Corr`/`Cov`/`Beta` with a `nil` list do the same.
- `DataTable.RollingCol` and `EWMCol` record an invalid option on the table as well as on the result, as `DiffCol` already does. The per-column transforms and the builders return an empty list carrying the table's error for a column that is not there.
- `GroupedColumnTransform.As` returns an empty list carrying the error it records, and a failure inside the per-group transform stops the call, is recorded on the table, and is carried by the empty result instead of turning into a column of `nil`.
- The CLI's `rolling`, `ewm` and `expanding` report a failed reducer as an error and store nothing; `ewm` stops inferring failure from the result's length.
- The empty result stays empty. The request for a same-length `nil` list is not taken, for the reason above.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `chainable-never-nil`: adds "A window view hands its failure to every reducer".
- `datalist-ewm`: the "Exactly one decay parameter" scenario says the reducers' empty list carries the error, and that the error is recorded (it has been an error, not a warning, since `make-errors-non-terminating`).
- `cli-timeseries-commands`: adds "A failed window reducer is an error".

## Impact

- `datalist_window.go`, `datalist_ewm.go`, `datatable_window.go`, `datatable_groupby_window.go`, `cli/commands/timeseries.go`; new tests beside each.
- `Docs/DataList.md` (Rolling, EWM, Expanding and the Error Handling section), `Docs/DataTable.md` (the per-column and grouped transforms' Errors paragraphs), `Docs/cli-dsl.md` if the commands' behaviour is described there, both changelogs, `api-review.md` (D-18), `delivery-status.md`.
- Not in this change: `ExpandingDataList` has no invalid option (`minObs <= 0` means 1), so only its column-not-found path changes.
