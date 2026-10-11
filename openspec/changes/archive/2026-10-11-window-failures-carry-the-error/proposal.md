# Proposal: window-failures-carry-the-error

## Why

Finding D-18 of the API review ([#224](https://github.com/HazelnutParadise/insyra/issues/224)) observed that the `Rolling`, `EWM` and `Expanding` reducers return an empty list when the options are invalid, and asked for a list of the source's length filled with `nil` instead.

That request contradicts a recorded rule: a failed transform returns an empty list carrying the error. A same-length list of `nil` would be indistinguishable from a window that simply had too few observations everywhere.

Measured against that rule, the reducers broke the half of it that matters: the empty list they returned did not carry the error. Furthermore, `DataTable.RollingCol` and `EWMCol` ran on a snapshot without recording on the table, and the CLI's `rolling` stored an empty list instead of reporting an error.

## What Changes

- Every reducer of `RollingDataList`, `EWMDataList` and `ExpandingDataList` whose view could not be built returns an empty list carrying the error. `Apply(nil)` and `Corr`/`Cov`/`Beta` with a `nil` list do the same. Error messages preserve this line's original wording (e.g. "Rolling: Window must be > 0").
- `DataTable.RollingCol` and `EWMCol` record an invalid option on the table as well as on the result. The per-column transforms and the builders return an empty list carrying the table's error for a column that is not there.
- `GroupedColumnTransform.As` returns an empty list carrying the error it records, and a failure inside the per-group transform stops the call, is recorded on the table, and is carried by the empty result instead of turning into a column of `nil`.
- The CLI's `rolling`, `ewm` and `expanding` report a failed reducer as an error and store nothing.
- The empty result stays empty.
- Left on 0.4: sticky Err, and fail()/warn() error routing separation.

## Capabilities

### New Capabilities

- `chainable-never-nil`: adds "A window view hands its failure to every reducer".

### Modified Capabilities

- `datalist-ewm`: the "Exactly one decay parameter" scenario says the reducers' empty list carries the error, and that the error is recorded.
- `cli-timeseries-commands`: adds "A failed window reducer is an error".

## Impact

- `datalist_window.go`, `datalist_ewm.go`, `datatable_window.go`, `datatable_groupby_window.go`, `cli/commands/timeseries.go`; tests beside each.
- `Docs/DataList.md`, `Docs/DataTable.md`, both changelogs, `api-review.md` (D-18), `delivery-status.md`.
