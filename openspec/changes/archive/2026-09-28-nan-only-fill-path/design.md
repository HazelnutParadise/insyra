# Design: nan-only-fill-path

## Why the CLI is only wrong for ffill and bfill with a limit

"Fill both, then restore the other kind" is exact whenever a target cell's fill value does not depend on what the other cells were filled with. `mean`, `median` and `mode` compute from the observed values before writing anything. `interpolate` draws each gap from the observed points on either side, and the other kind's cells are not observed points. `ffill` and `bfill` without a limit copy the last observed value into every missing cell of a run, so the target cells get the same value either way. Only the limit reads the run as a count, and there the other kind's cells were counted.

## The rule for the other kind

With `missing nan`, a `nil` cell is not the kind the command fills, so it is left as it is. It is not a value either, so it cannot be copied into a `NaN`, and since it is not being filled it does not use up the `limit`. The simplest correct implementation for `ffill`/`bfill` is to fill the list with the other kind's cells taken out and put them back at their positions afterwards; that also leaves every no-limit result unchanged. `interpolate` must not work on the compacted list, because it measures distance by position, so it keeps the fill-and-restore path.

## Why not a library parameter

See the proposal: twelve signatures would break for a need that `ReplaceNaNsWith` already covers for mean and median, and the one case it does not cover (a forward fill of `NaN` alone) is served by the CLI.
