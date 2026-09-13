# Proposal: ccl-portable-integer-arguments

## Why

The first CI run on `0.4` (2026-09-11) failed on the windows runner with `MID('abc', 2, 10^300) = ""; want "bc"`. The same test passes on a Mac. Go leaves `int(f)` and `time.Duration(f)` undefined when `f` is NaN, infinite or past the integer range, and the platforms disagree: on amd64 `int(1e300)` is the most negative int, on arm64 it saturates to the most positive one. Reproduced locally with `GOARCH=amd64 go test` under Rosetta.

So a CCL expression can give different answers on a laptop and on a server, with no error on either. `wholeIndex` and the sequence functions' argument check already guard against this. These places convert a user's number without a guard:

- `LEFT`, `RIGHT` and `MID` counts and positions: a huge count gives `""` on amd64.
- `ROUND`'s digit count: `ROUND(x, NaN)` rounds to an integer on arm64 and returns NaN on amd64.
- `DATEADD` by day, month or year, and by hour, minute or second.
- `DAY`, `HOUR`, `MINUTE` and `SECOND` given a number of seconds.
- A date plus or minus a number of days.
- The bounds of a row range, `A.(0:1)`.

## What Changes

- A character count, character position or digit count is clamped to the int32 range before it becomes an int. A count past the end of a string keeps meaning "to the end", as in Excel, on every platform. NaN is refused.
- A date shift or duration that a `time.Duration` or `AddDate` cannot hold is an error instead of a platform-dependent date. That covers a calendar shift past the int32 range and a duration past about 292 years.
- A row range bound goes through `wholeIndex`, like a row index. It is refused when it is not a finite whole number within the int32 range.
- Results for ordinary values are unchanged. A fractional `DATEADD` count still truncates, as it did before.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `ccl-evaluation-safety`: adds that a numeric argument gives the same answer on every platform.

## Impact

- `internal/ccl/ccl_evaluator.go`, `stdlib.go`, `stdlib_datetime.go`, `stdlib_math.go`, `stdlib_string.go`; a test that has to pass under `GOARCH=amd64` as well as natively.
- `Docs/CCL.md`; `CHANGELOG.md` and `CHANGELOG_TW.md`.
- Found by `ci-hygiene`, which made CI run on `0.4` for the first time. The Test workflow's windows and ubuntu legs are amd64.

## Backport to dev (0.3.x)

Dev received:
- `LEFT`, `RIGHT` and `MID` counts and positions and `ROUND`'s digit count clamped to the int32 range, with `NaN` an error (`clampedInt`).
- `DATEADD` refusing a day, month or year shift that is `NaN` or past the int32 range, and an hour, minute or second shift, like `DAY`, `HOUR`, `MINUTE` and `SECOND` given seconds, past what a Duration holds (`durationOf`).
- A date plus or minus a number of days refusing `NaN`, the infinities and a shift past 106,751 days, and a row range bound refusing `NaN` and values outside the int32 range.
- The `Docs/CCL.md` notes and the changelog entry, with a test run natively and under `GOARCH=amd64`.

Adapted on dev:
- Dev has no `wholeIndex`: a row range bound goes through `rangeBound`, which refuses only `NaN`, the infinities and values outside int32, and still truncates a fractional bound as dev always has.
- Dev has no `daysToDuration`: date ± number keeps dev's whole-hour arithmetic, `time.Duration(days*24) * time.Hour`, through `dayShift`, which refuses only the inputs where that conversion is undefined or the product wraps.
- A range inside `A.( )` is converted in `evaluateRange`, not in the `:` operator, so both places go through `rangeBound`, as does the numeric side of a mixed name and index range. A single row index, `A.(n)`, keeps its conversion: it is not a range bound.
- Measured on dev before the change: the guard cases gave different wrong answers on arm64 and under `GOARCH=amd64` (`MID('abc', 2, 10^300)` was `"bc"` and `""`, `DATEADD(B, 10^300, 'day')` 2023-12-31 and 2024-01-01, `SUM(A.(0:C))` with `C` NaN `3` and an error), `SUM(A.(0:10^300))` already failed on both, and a separate table of ordinary values, truncating ones included, passed on both before and after.

Stayed on 0.4:
- `shiftDays` keeping the sub-hour part of a fractional day count: it depends on batch 8's `daysToDuration` and changes returned values.
- A row range bound going through `wholeIndex`: a fractional bound would become an error, which batch 8 introduced.
