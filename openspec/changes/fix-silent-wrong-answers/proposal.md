# Proposal: fix-silent-wrong-answers

## Why

Three defects found on 2026-09-12 by `test-untested-packages`. None of them fails, errors or warns: each returns something that looks right.

- **`utils.FormatValue` prints a different number on amd64 and arm64, and a wrong one on arm64.** It decides whether a float is a whole number with `v == float64(int(v))`, and `int(v)` is undefined in Go for a value outside `int`'s range. Measured for `2^63`: a Mac prints `9223372036854775807` — an exact-looking integer that is one less than the value — while Linux and Windows print `9.2234e+18`. This is `Show()`'s formatter, so the same table reads differently depending on where it runs. It is the same defect class as `ccl-portable-integer-arguments` fixed in CCL on the same day.
- **`utils.ConvertToDateString` gives a wrong date for a millisecond timestamp after 2262-04-11.** The millisecond branch computes `time.Unix(0, ts*int64(time.Millisecond))`, and that multiplication overflows `int64` above 9223372036854. The branch's own window runs to 10^14, so it accepts inputs it then reads as a different date: `99999999999999` should be 5138-11-16 and comes back as 2216-02-08.
- **`isr.DT.From(map[int]any{…})` never produces a table.** The type is listed as supported in `DT.From`'s and `DT.Of`'s doc comments. It turns each key into a string with `conv.ToString`, so key `0` becomes `"0"`, and hands that to `AppendRowsByColIndex`, which wants an Excel-style index. Every key is rejected and the result is an empty table carrying `Invalid column index '0'`. The sibling path for `Row` gets this right, through `numberToColIndex`, which turns `0` into `"A"`.

## What Changes

- `FormatValue` checks the value is inside `int64`'s range before converting, and converts to `int64` rather than the platform-sized `int`. `2^63` and anything larger take the exponent branch on every architecture. Whole numbers inside the range are unchanged.
- The millisecond branch uses `time.UnixMilli`, which does not multiply.
- `isr`'s `map[int]any` case converts each key with `numberToColIndex`, the same helper the `Row` path uses.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `number-text-format`: gains the rule that a number's text is the same on every architecture.
- `date-parsing-coverage`: gains the millisecond range that must round-trip.

## Impact

- `internal/utils/utils.go`, `isr/dt.go`, and the tests alongside them.
- User-visible: `Show()` prints `9.2234e+18` rather than `9223372036854775807` for `2^63` on arm64 — on amd64 nothing changes, because that is already what it printed. A far-future millisecond timestamp reads as the right date. `DT.From(map[int]any{…})` produces a table instead of an empty one. Both changelogs get entries.
- Verified on both architectures with `GOARCH=amd64 go test` under Rosetta, the way the CCL fix was.
