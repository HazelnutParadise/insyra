# Design: window-failures-carry-the-error

## Which error the empty result carries

`failedResult` hands back the receiver's `Err()`, which is sticky and so holds the chain's first failure, not necessarily this one. The window views do the same, for the same reason: the result should tell the caller what went wrong first. The view remembers the `*ErrorInfo` at the moment it failed (`dl.Err()` right after `dl.fail`), and every reducer builds its empty result from that. The views' `err string` field becomes `err *ErrorInfo`.

## The snapshot problem

`RollingCol`, `EWMCol` and `DiffCol` compute on a copy of the column, so a failure is recorded on a list nobody holds. `DiffCol` already copies the failure to the table. `RollingCol` and `EWMCol` do the same, but through `dt.setError`, not `SetErr`: the snapshot has already logged the failure, and `SetErr` would log it a second time and call the error hook twice.

## Grouped transforms

A grouped transform applies a per-group function to a throwaway sub-list per group. Every failure those functions can have today is a bad argument (`Diff`/`PctChange` periods, `Shift`'s extra fill value, `Rolling`'s options, `Apply(nil)`), identical for every group. So `As` stops at the first group whose result carries an error, records it on the table with `setError` (the sub-list has logged it), and returns an empty list carrying it. Stopping at the first group also keeps a table with thousands of groups from logging the same failure thousands of times. A table with no groups produces no failure, as before.

## Why not the same-length `nil` list

A list of `nil` the source's length is what a valid window produces when no position has `MinObs` observations. Returning it for an invalid option would make the two cases indistinguishable to a caller who does not check `Err()`, and it would append into a table silently. An empty list fails loudly at `AppendCols`, and with this change it also says why.
