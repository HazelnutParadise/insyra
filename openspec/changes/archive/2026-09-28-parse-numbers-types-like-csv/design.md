# Design: parse-numbers-types-like-csv

## The CSV rule, restated for a list

`inferCSVColumnTypes` looks at a column of strings and decides one type for all of it:

1. every cell parses with `strconv.ParseInt(s, 10, 64)` and no cell is empty → every cell becomes `int64`;
2. otherwise, every non-empty cell parses with `strconv.ParseFloat(s, 64)` → every cell becomes `float64`, an empty cell `NaN`;
3. otherwise the column stays text.

`ParseNumbers` applies rules 1 and 2 to the values it can read, using the same two `strconv` calls, so for a list of strings with no unreadable value it produces exactly what the CSV reader produces for that column. The differences are the ones `ParseNumbers` has always documented:

- **Unreadable values stay.** `ParseNumbers` converts what it can and leaves the rest ("hello" stays "hello") with an error recorded, where a CSV column with one unreadable cell stays entirely text. Making `ParseNumbers` all-or-nothing would remove the per-value conversion it documents, so it is not part of this change.
- **White space is trimmed first**, as `conv.ParseF64` always did. The CSV reader leaves it unless `TrimLeadingSpace` is set.
- **The list may already hold numbers.** A CSV column never does. A Go integer (any kind, named types included, since a named numeric type is a number everywhere) that fits `int64` counts as an integer; a `uint`/`uint64` above `math.MaxInt64` and any float count as decimals, which is what the CSV reader does with the same number written as text. Every counted value is converted to the chosen type, so the numbers in the list end up one type, as a CSV column does.

## What is left alone, and what is an error

| Value | Result | Error |
| --- | --- | --- |
| `nil` | unchanged | no — it is a missing value, not an unreadable one |
| a number that is not a Go integer or float kind (a decimal) | unchanged | no — it is already a number |
| a string that is not a number after trimming | unchanged | yes |
| any other value (`bool`, `time.Time`, a slice) | unchanged | yes |

All unreadable values are reported by one `fail`, giving how many there were and the first one's row (1-based, as the other `DataList` messages count). A string is quoted with `%q`; any other value is named by its type, never printed, so a huge or self-referential value cannot flood or overflow the message.

## Capitalize

`golang.org/x/text/cases` has tailorings only for `und af az el lt nl tr` (see `supported` in `cases/map.go`, v0.41.0); `language.English` resolves to the root rules. A probe over every valid rune in four positions plus Dutch, Turkish and Afrikaans cases found no input where the two differ. The change is a statement of intent; a test pins the root behaviour (`ijssel` → `Ijssel`, `istanbul` → `Istanbul`) so a later switch to a tailored language is visible.
