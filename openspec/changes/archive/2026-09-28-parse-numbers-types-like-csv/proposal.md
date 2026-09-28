# Proposal: parse-numbers-types-like-csv

## Why

`DataList.ParseNumbers` turns every value it can read into a `float64`, including values that were already integers. An `int64` above 2^53 cannot be held exactly by a `float64`, so parsing a column of 19-digit IDs, or re-parsing a column that already held them, silently changes the numbers: `"9007199254740993"` comes back as `9007199254740992`. The CSV reader settled this the other way on purpose: its column-level inference (`inferCSVColumnTypes` in `read.go`) keeps a column whose values are all integers as `int64` and makes a column with any decimal `float64`, the way `pandas.read_csv` does. The same text should not get a different type depending on whether it arrived in a CSV file or went through `ParseNumbers`. This is finding D-16 of the API review ([#223](https://github.com/HazelnutParadise/insyra/issues/223)).

The same finding notes that `Capitalize` hardcodes `language.English`. Measured on golang.org/x/text v0.41.0, the casing package has tailorings only for `und af az el lt nl tr`, so English falls back to the root rules and `cases.Title(language.English)` and `cases.Title(language.Und)` give the same output for every rune (0 differences over 778,253 inputs). Naming `language.Und` says what the code does without changing any output.

## What Changes

- **BREAKING**: `ParseNumbers` types the list as a whole with the CSV reader's rule. When every value it can read is an integer and no string is empty, every one becomes `int64`; otherwise every one becomes `float64` and an empty string becomes `NaN`. `NewDataList("1", 2, "3", 8).ParseNumbers()` gives `int64` values where it gave `float64` ones, and an integer text above 2^53 keeps every digit.
- A value that is already a number takes part in the same decision: an integer of any width that fits `int64` counts as an integer, a float as a decimal, and all of them are converted to the chosen type. A number insyra reads but Go does not store as an integer or float (a decimal) is left as it is.
- `nil` is left as it is and is no longer recorded as a failure. A string that is not a number and a value that is not a number (a `bool`, a `time.Time`) are left as they are and recorded as one error that counts them and names the first, where each used to be recorded separately.
- Text is trimmed of surrounding white space before it is read, as it always was; the CSV reader trims only with `TrimLeadingSpace`.
- `Capitalize` uses `language.Und`. Its output does not change.

## Capabilities

### New Capabilities

- `datalist-conversions`: how `DataList.ParseNumbers` types what it parses, and which casing rules `Capitalize` follows.

### Modified Capabilities

(none)

## Impact

- `datalist.go` (`ParseNumbers`, `Capitalize`), `datalist_test.go` (`TestDataListParseNumbers` expected `float64` for an all-integer list), a new test file.
- `Docs/DataList.md` (`ParseNumbers`, `Capitalize`), `Docs/cli-dsl.md` (`parsenums`), both changelogs, `api-review.md` (D-16), `delivery-status.md`.
- The CLI's `parsenums` calls `ParseNumbers` and inherits the new typing; its code does not change.
- Not in this change: making `Capitalize` take a language, which would change its signature and is a choice between APIs; and whether a list with an unreadable string should be left entirely as text the way a CSV column is, which would remove the per-value conversion `ParseNumbers` documents.
