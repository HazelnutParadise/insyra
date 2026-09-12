# Proposal: fix-clear-defects-core

## Why

The decision-free half of the review's remaining core findings: #337 (IN-10, IN-12, IN-13, IN-14, IN-18), #338 (IN-16), #340 (IN-21) and the first half of #228 (T-15). Each has one right answer, and each currently returns something that looks fine.

- **A 16-digit timestamp read as seconds.** `ConvertToDateString(1700000000000000)` gave `53872825-06-17`. The millisecond window also stopped at 10^14, so 10^14 to 10^15 fell through to the seconds branch and produced the year 3,170,843.
- **`CalcColIndex` produced names its own parser refused.** `ParseColIndex` accumulated the 1-based value and subtracted at the end, so the largest index needed a 1-based value of `maxInt+1` and the overflow guard rejected it. Everything below round-tripped, so the break was only at the top.
- **A value that is not whole printed as a whole number.** `FormatValue(9999.99999)` showed `10000` — rounded to four places and then trimmed of its trailing zeros. In a table cell that reads as an exact integer.
- **`ConvertDateFormat` rewrote letters inside words.** It ran `strings.ReplaceAll` for each pattern over the whole string, so `MMM` became `011` and `Date:` became `2ate:`. `MMM`, `MMMM` and AM/PM were not supported at all, and there was no way to write a literal.
- **`IsNumeric` and `ToFloat64Safe` disagreed about every named numeric type.** `type Celsius float64` was a number to the first (which goes through reflection) and not convertible to the second (a plain type switch). 82 call sites read values through the second.
- **`BiIndex.Set` leaked an id.** Moving a name onto another id deleted the old mapping without freeing the id, so it became a hole no `Assign` could ever hand out.
- **Three interpolations answered a NaN x with a plausible number.** The issue named `NearestNeighborInterpolation`, which returned `data[0]` because every comparison against NaN is false; `Lagrange` and `Newton` turned out to return `NaN` with no error as well. `Linear` and `Quadratic` already refused it.
- **Two unnamed tables could not be merged vertically.** `NewDataTable(NewDataList(...))` leaves every column unnamed, and the duplicate-name check counted the empty name as a duplicate of itself — so the plainest constructor produced a shape that could not be merged with another of its kind.

## What Changes

- `ConvertToDateString` reads 16 to 18 digits as microseconds, and the millisecond window runs to 10^15 so there is no gap between the two.
- `ParseColIndex` accumulates the 0-based index directly, so the whole range `CalcColIndex` can produce reads back. Values it genuinely cannot represent are still refused.
- `FormatValue` falls back to the full representation when rounding to four places would leave something that reads as a whole number and is not.
- `ConvertDateFormat` is a scanner: a maximal run of one letter is one token, `[text]` is copied verbatim, and anything else passes through. `MMMM`, `MMM` and `A`/`a` are added. `hh` and `h` still mean the 24-hour layout — that predates this rewrite and callers pass their own patterns, so changing it would silently alter what their charts print; it is documented instead.
- `ToFloat64` and `ToFloat64Safe` fall back to reflection for a named type over a numeric kind, which is what `IsNumeric` already did and the shape `accel`'s projection settled on. The concrete type switch still runs first, so the fast path is unchanged.
- `BiIndex.Set` puts the name's old id on the free list.
- `NearestNeighborInterpolation`, `LagrangeInterpolation` and `NewtonInterpolation` refuse a NaN with `ErrOutOfBounds`, like `Linear` and `Quadratic`.
- Vertical merge lines unnamed columns up by their position among the unnamed ones and named columns by name. A duplicate name is still refused, though no public path can build one: both the constructor and `SetColNameByNumber` rename the second to `n_1`, which the tests now pin.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `date-parsing-coverage`: the microsecond window, and a format pattern that does not rewrite letters inside words.
- `datalist-numeric-input`: a named type over a numeric kind is a number everywhere or nowhere.
- `interpolation-correctness`: a NaN is refused by every interpolation.
- `datatable-row-operations`: unnamed columns merge by position.

## Impact

- `internal/utils/utils.go`, `internal/core/biindex.go`, `internal/algorithms/interpolation.go`, `datatable_merge.go`, and tests beside each.
- User-visible, so both changelogs get entries: `Show` prints a near-whole number in full; `mkt.RFM`, `mkt.CustomerActivityIndex` and `plot.CreateKlineChart` take a better date pattern but now need `[...]` around literal letters; `DataList.LagrangeInterpolation`, `NearestNeighborInterpolation` and `NewtonInterpolation` report a NaN instead of answering; a column of a named numeric type is now read as numeric throughout.
- `api-review.md` rows IN-10, IN-12, IN-13, IN-14, IN-16, IN-18, IN-21 and the first half of T-15.
- The second half of T-15 — `Merge(other IDataTable)` asserting `*DataTable` immediately — is the `IDataList`/`IDataTable` interface decision (K-7, #208) and is left alone.
