# Proposal: cli-one-token-rule

## Why

#315 (CLI-6): the commands that pick a column each read the token their own way. On a table with columns `name, price, qty`, `col`, `sort`, `swap` and `dropcol` found `price` and not `B`; `get` and `set` found `B`, and `get t 0 price` printed `<nil>` with no error; `fillna` and `groupby` took both, name first. Digits meant a position in some commands and nothing in others, and no Usage line said which.

The library settled its half on 2026-09-22 (#225): a bare string is an Excel-style index and `Name("price")` is a name. A CLI token has no type to carry that distinction, so the CLI needs its own rule. The owner ruled on 2026-09-27 after comparing csvkit and xsv, which read an index-looking token as a position and anything else as a name. Their positions are digits; ours are letters as well, and letters collide with real column names (`a`, `x`, `ID`), so neither order is safe on its own. pandas deprecated exactly that kind of fallback in `Series.__getitem__` (checked on 2.3.3).

Measuring this also showed a command's failure leaking into the next: after `col t B` failed, a `sort t price` that had just succeeded failed with "no column is named B", because the table kept the error until the next command checked for one.

## What Changes

- **BREAKING**: one rule for every token that picks a column or a row. A bare token is read as a 0-based number (negative from the end), as letters for columns, and as a name. When the readings that land agree, that is the target; when two disagree the command refuses and says which prefix to add. `number:`, `index:` and `name:` pick one reading. Rows take `number:` and `name:`.
- Applies to `col`, `row`, `get`, `set`, `sort`, `swap`, `dropcol`, `droprow` and the column lists of `fillna`, `groupby`, `describe`, `encode`, `parsedates`, `pivot`, `unpivot`, `resample`, `scale` and `merge … on`. `get` and `set` accept row names.
- A command starts with no error recorded on any variable, so one command's failure is never reported by the next.
- Usage lines say `<col>` and `<row>`; `Docs/cli-dsl.md` states the rule.

## Capabilities

### New Capabilities

- `cli-target-tokens`: how a CLI token picks a column or a row.

### Modified Capabilities

(none)

## Impact

- `cli/commands/targets.go` (the resolver), `registry.go` (clearing stale errors), and the commands listed above; new tests in `col_token_test.go`.
- `Docs/cli-dsl.md`, the CLI skill and its references, `cli/AGENTS.md`, both changelogs, `api-review.md` (CLI-6), `delivery-status.md`.
- Not in this change: `Merge` in the library joins on column names, where every other column parameter takes a selector. The CLI resolves `on` tokens to names before calling it.
