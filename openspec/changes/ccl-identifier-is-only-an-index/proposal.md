## Why

CCL's documented rule is one line: a bare word is an Excel-style column index, and a column name is written `['name']` (`Docs/CCL.md`). The implementation adds two undocumented fallbacks to the name map, one in `Bind` and one in the assignment target, and both fire only when the identifier fails to parse as letters. So whether a bare word can mean a column name depends on whether that name contains a digit or an underscore (#341, CCL-1).

Measured on `0.4` with a table whose columns are `price` and `qty_1`:

| Expression | Today |
| --- | --- |
| `qty_1 * 2` | resolves by name, because `qty_1` is not letters |
| `price * 2` | `column PRICE does not exist: the table has 2 column(s)` |
| `price = ['price'] * 10` | resolves by name |
| `['price'] * 2`, `A * 2`, `['price'] = …`, `A = A * 10` | work, and are the documented forms |

The owner ruled on 2026-09-21 that index-only resolution is the intended design, matching the 2026-09-13 ruling that `ColumnIndex` must not fall back to a name.

## What Changes

- **BREAKING**: a bare identifier resolves only as an Excel-style column index, on both sides of an assignment. `qty_1 * 2` and `price = …` become errors; the bracketed form `['qty_1']`, `['price']` is how a name is written, and it already works everywhere.
- The error teaches the fix. The name map is still consulted, but only to write the message, never to resolve: when the table has a column of that name, the message says to write `['name']`. When it does not, the message stays what it was.
- `Docs/CCL.md` drops "first tried as" and states the single rule; both changelogs carry the breaking entry; the `insyra` skill follows.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `ccl-evaluation-safety`: the keywords-and-out-of-range requirement gains the identifier rule and what the message must offer.

## Impact

- `internal/ccl/ccl_compiler.go` (`Bind`), `ccl.go` (`checkCCLColRange`), `datatable_ccl.go` (`executeAssignment`).
- `Docs/CCL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `skills/insyra/`.
- `api-review.md` CCL-1, `delivery-status.md`, issue #341.
