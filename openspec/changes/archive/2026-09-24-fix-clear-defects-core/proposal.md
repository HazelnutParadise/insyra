# Proposal: fix-clear-defects-core

## Why

The decision-free half of the review's remaining core findings: #337 (IN-10, IN-12, IN-13, IN-18; IN-14 stayed on 0.4), #338 (IN-16), #340 (IN-21) and the first half of #228 (T-15). Each has one right answer, and each currently returns something that looks fine.

- **A 16-digit timestamp read as seconds.** `ConvertToDateString(1700000000000000)` gave `53872825-06-17`. The millisecond window also stopped at 10^14, so 10^14 to 10^15 fell through to the seconds branch and produced the year 3,170,843.
- **`CalcColIndex` produced names its own parser refused.** `ParseColIndex` accumulated the 1-based value and subtracted at the end, so the largest index needed a 1-based value of `maxInt+1` and the overflow guard rejected it. Everything below round-tripped, so the break was only at the top.
- **A value that is not whole printed as a whole number.** `FormatValue(9999.99999)` showed `10000` — rounded to four places and then trimmed of its trailing zeros. In a table cell that reads as an exact integer.
- **`IsNumeric` and `ToFloat64Safe` disagreed about every named numeric type.** `type Celsius float64` was a number to the first (which goes through reflection) and not convertible to the second (a plain type switch). 82 call sites read values through the second.
- **`BiIndex.Set` leaked an id.** Moving a name onto another id deleted the old mapping without freeing the id, so it became a hole no `Assign` could ever hand out.
- **Three interpolations answered a NaN x with a plausible number.** The issue named `NearestNeighborInterpolation`, which returned `data[0]` because every comparison against NaN is false; `Lagrange` and `Newton` turned out to return `NaN` with no error as well. `Linear` and `Quadratic` already refused it.
- **Two unnamed tables could not be merged vertically.** `NewDataTable(NewDataList(...))` leaves every column unnamed, and the duplicate-name check counted the empty name as a duplicate of itself — so the plainest constructor produced a shape that could not be merged with another of its kind.

## What Changes

- `ConvertToDateString` reads 16 to 18 digits as microseconds, and the millisecond window runs to 10^15 so there is no gap between the two.
- `ParseColIndex` accumulates the 0-based index directly, so the whole range `CalcColIndex` can produce reads back. Values it genuinely cannot represent are still refused.
- `FormatValue` falls back to the full representation when rounding to four places would leave something that reads as a whole number and is not.
- `ToFloat64` and `ToFloat64Safe` fall back to reflection for a named type over a numeric kind, which is what `IsNumeric` already did and the shape `accel`'s projection settled on. The concrete type switch still runs first, so the fast path is unchanged.
- `BiIndex.Set` puts the name's old id on the free list.
- `NearestNeighborInterpolation`, `LagrangeInterpolation` and `NewtonInterpolation` refuse a NaN with `ErrOutOfBounds`, like `Linear` and `Quadratic`.
- Vertical merge lines unnamed columns up by their position among the unnamed ones and named columns by name. A duplicate name is still refused, though no public path can build one: both the constructor and `SetColNameByNumber` rename the second to `n_1`, which the tests now pin.

## Capabilities

### New Capabilities

- `datalist-numeric-input`: a named type over a numeric kind is a number everywhere or nowhere. (This capability exists on 0.4; on this line it starts with this requirement.)

### Modified Capabilities

- `date-parsing-coverage`: the microsecond window.
- `interpolation-correctness`: a NaN is refused by every interpolation.
- `datatable-row-operations`: unnamed columns merge by position.

## Impact

- `internal/utils/utils.go`, `internal/core/biindex.go`, `internal/algorithms/interpolation.go`, `datatable_merge.go`, and tests beside each.
- User-visible, so both changelogs get entries: `Show` prints a near-whole number in full; `DataList.LagrangeInterpolation`, `NearestNeighborInterpolation` and `NewtonInterpolation` record a NaN x in `Err()` instead of answering; a cell of a named numeric type is read as numeric throughout.
- The second half of T-15 — `Merge(other IDataTable)` asserting `*DataTable` immediately — is the `IDataList`/`IDataTable` interface decision (K-7, #208) and is left alone.

## Backport to dev (0.3.x)

Dev received:
- the microsecond window and the millisecond window extended to 10^15;
- `ParseColIndex` reading back everything `CalcColIndex` produces;
- `FormatValue` showing a near-whole value in full, through `strconv.FormatFloat(v, 'f', -1, 64)` because `FloatText` is not on this line; a test shows it matches `FloatText`'s rule over every value that reaches the fallback;
- `ToFloat64`/`ToFloat64Safe` reading a named numeric type (0.4's decimal fallback is not here);
- `BiIndex.Set` freeing the old id; the three interpolations refusing a NaN (the `DataList` wrappers return NaN and record the error in `Err()`);
- vertical merge of unnamed columns by position.

Left on 0.4:
- the `ConvertDateFormat` scanner, its test and its changelog entry: breaking, since existing `mkt.RFM`, `mkt.CustomerActivityIndex` and `plot.CreateKlineChart` patterns with literal letters would print differently;
- `api-review.md` and `delivery-status.md` edits.
