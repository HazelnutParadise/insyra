# Design: fix-api-review-batch-6

## Context

The three defects share a shape: the failure produces something that looks like data. All were reproduced by a failing test first.

## Decisions

1. **One decoder resolver, shared.** The root reader and `csvxl` each had their own `switch` with a `default: reader = file`, which is how undecoded bytes reached the table. `internal/csv.DecodingReader` replaces both and returns an error for an unknown name, listing what is supported. *Alternative*: keep passing bytes through and warn — rejected, a warning does not stop the broken text from being compared, joined and exported.
2. **BOM order.** `FF FE 00 00` (UTF-32LE) starts with `FF FE` (UTF-16LE), so the longer marks must be tested first. Nothing else in the function depended on the old order.
3. **Detector failure is not a read failure.** chardet cannot name a charset from a one-byte sample; the old code turned that into "failed to detect encoding for file X" and the read died. It now warns and assumes UTF-8, and the decoder reports any byte it cannot handle.
4. **Formula sanitisation is opt-in, via a new options struct.** `ToCSV`'s signature is four positional bools with no room for another, so `ToCSVWithOptions` mirrors the existing `ReadCSV_FileWithOptions` pattern. Default off is deliberate: prefixing a cell with `'` changes the value, and a table written and read back would no longer match. Safe-by-default is the better rule for a web application exporting untrusted input; for a data library whose CSV is as often a round trip, silently altering cells is the worse failure. The doc comment states the trade-off so the caller can choose, and the default can be flipped later in one line if the owner prefers it.
5. **Only `=`, `+`, `-`, `@`** (after leading whitespace, which spreadsheets skip) start a formula, so only those are prefixed; an ordinary value is written unchanged.
6. **SQLite identifier.** `quoteSQLIdent` already existed and was used everywhere else in the file; the `PRAGMA table_info` call simply had not been converted.

## Risks / Trade-offs

- [A file in an exotic encoding that used to "load" now errors] → it was loading broken text; the error names the supported encodings.
- [`SanitizeFormulas` off by default leaves the injection open for callers who do not know about it] → documented on the option, in `Docs/DataTable.md` and in the changelog; flipping the default is a one-line change if the owner decides otherwise.
