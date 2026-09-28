# Proposal: nan-only-fill-path

## Why

`FillNaNWithMean` fills `NaN` and leaves `nil`, and has been Deprecated since v0.2.19 with the advice "use `FillWithMean`", which fills `nil` too. `legacy-transforms-pinned` showed the two differ, so the advice does not preserve behaviour. The library needs no new parameter to fill `NaN` alone with a statistic: `ReplaceNaNsWith` replaces `NaN` and nothing else, and the mean of the values that are there is `dl.Clone().ClearNilsAndNaNs().Mean()`. Adding a missing-kind parameter to the twelve `Fill*` methods was considered and rejected: every one already spends its trailing optional slot (`limit`, `extrapolate`, `cols`), so all twelve signatures would break for a need nobody has raised beyond this legacy method, and neither pandas (one missing marker) nor Polars (whose `fill_nan` takes a value, as `ReplaceNaNsWith` does) offers more.

The CLI's `fillna … missing nan|nil` fills only one kind by filling both through the library and then putting the other kind back. That is exact for `mean`, `median`, `mode` and `interpolate`, whose results for the target cells do not depend on how the other cells were filled. It is not for `ffill` and `bfill` with a `limit`: the other kind's cells are counted toward the limit before they are put back, so `fillna x ffill limit 1 missing nan` on `[1, nil, NaN]` leaves the `NaN` unfilled although it is the first `NaN` after a value, and `Docs/cli-dsl.md` shows exactly this command as an example.

## What Changes

- `FillNaNWithMean`'s Deprecated notice and its section in `Docs/DataList.md` give the replacement that keeps its behaviour: `dl.ReplaceNaNsWith(dl.Clone().ClearNilsAndNaNs().Mean())`, and the same with `Median()`. A test shows the two give the same values.
- `fillna … ffill|bfill … missing nan|nil` skips the cells of the other kind: they are not filled, give no value to copy, and do not count toward `limit`. Without a `limit`, results are unchanged.
- `Docs/cli-dsl.md` says what `missing` does to `limit`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datalist-numeric-input`: adds "A NaN-only mean fill has a replacement that keeps its behaviour".
- `cli-timeseries-commands`: adds "fillna counts only the chosen kind of missing toward limit".

## Impact

- `datalist.go` (`FillNaNWithMean`'s doc comment), `cli/commands/fillna.go`, tests beside each.
- `Docs/DataList.md`, `Docs/cli-dsl.md`, both changelogs (CLI), `api-review.md` if a row covers it, `delivery-status.md`.
