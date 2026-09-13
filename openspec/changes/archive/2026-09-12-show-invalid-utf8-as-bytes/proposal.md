# Proposal: show-invalid-utf8-as-bytes

## Why

A Parquet `Binary` column arrives as a `string` holding raw bytes, because `NewDataList` flattens every `reflect.Slice` and so a cell cannot be a `[]byte`. `FormatValue`'s `string` case then wraps those bytes in quotes and hands them to the terminal, which draws whatever it likes.

Measured on 2026-09-12, a three-byte cell holding `00 ff 41` (NUL, a byte that is never valid in UTF-8, and `A`):

```
0:         'A-01'        nil          ← nil at character 25
1:         ' <?>A'        nil         ← nil at character 26
2:         'C-3'         nil          ← nil at character 25
```

Two things are wrong, and the second is the one that matters. The cell is unreadable, which is unavoidable for bytes that are not text. And the row is **one column out of alignment**: `runewidth` scores NUL at 0 and the invalid byte as one `U+FFFD` at 1, so the five-character cell is budgeted four columns, while the terminal draws it its own way. No width calculation can be right for bytes that are not text, so the table cannot line up as long as they are rendered as text.

`FormatValue` already knows the answer. Its `[]byte` case renders `00ff41`. The string case simply cannot reach it.

## What Changes

- A `string` that is not valid UTF-8 is displayed the way a `[]byte` is: as hex, truncated past 20 bytes with a count. One rule for "these are bytes, not text", rather than a second convention for the same thing.
- The check runs before the multi-line check, because a byte sequence that happens to contain `0x0A` is not a multi-line string.
- A valid UTF-8 string is untouched, CJK and emoji included.

The rendering is also the only one whose width is unambiguous: hex is ASCII, so the column lines up.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `cell-display-and-export`: a string that is not text is shown as bytes.

## Impact

- Any cell holding a string that is not valid UTF-8 changes from quoted mojibake to unquoted hex, in `Show` and every other display path. In this library that means a Parquet `Binary` column; a CSV load decodes its charset, so cells from there are text.
- The value itself does not change. `[]byte(cell)` still recovers the bytes, `ToCSV` still writes them, and `ToJSON` still replaces the invalid ones because a JSON string must be valid UTF-8.
- Not changed: a valid UTF-8 string holding control characters. `runewidth` scores those 0 and terminals disagree, so the same alignment argument applies, but such a value is still text and quoting it is still right. Recorded as an observation rather than fixed here.

## Backport to dev (0.3.x)

Dev received all of the code: `FormatValue`'s `string` case sends a string that fails `utf8.ValidString` to the `[]byte` rendering, before the multi-line check; the four tests; both changelog entries; the requirement, merged into `cell-display-and-export`.

Adapted:
- The changelog, the code comment and the test comment do not say a Parquet `Binary` column arrives as a string of raw bytes. That was 0.4's intermediate; on this line a binary column reads as `[]byte` once `parquet-binary-is-bytes` is backported, and before that as the text of the whole array.

Left on 0.4:
- The `Docs/parquet.md` row saying `Show` prints a binary string cell as hex: it describes 0.4's binary-as-string cells. The row for `[]byte` comes with `parquet-binary-is-bytes`.
