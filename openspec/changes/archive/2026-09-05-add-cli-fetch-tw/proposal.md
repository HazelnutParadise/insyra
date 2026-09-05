# Proposal: add-cli-fetch-tw

## Why

`fetch yahoo` is the only fetch source in the CLI. `datafetch.TWStock` (archived 2026-09-04/05) gives Taiwan daily prices, adjusted prices, ex-rights tables, institutional trades, margin balances, and full quote tables, but a `.isr` script cannot call it. For a Taiwan desk the CLI's data step is therefore still Yahoo-only, which is the gap the library change was meant to close.

## What Changes

- `fetch` gains a `tw` source with these forms (dates are `YYYY-MM-DD`; market defaults to `auto`; `[as <var>]` stores the `DataTable`):
  - `fetch tw <code> prices <from> <to> [twse|tpex|auto]` → `DailyPrices`
  - `fetch tw <code> adjprices <from> <to> [twse|auto]` → `DailyPricesAdjusted`
  - `fetch tw exrights <from> <to> [twse]` → `ExRights`
  - `fetch tw institutional <date> [twse|tpex|auto]` → `InstitutionalTrades`
  - `fetch tw margin <date> [twse|tpex|auto]` → `MarginBalance`
  - `fetch tw quotes [twse|tpex|auto]` → `AllDailyQuotes`
- Client construction goes through a package-level factory so tests inject a fake client returning canned tables; the real factory builds `datafetch.TWStock` with `Interval: 300ms`, `Retries: 2` — the backfill-safe values the datafetch docs recommend — and a new `config` key `fetch.tw.interval_ms` can override the interval.
- Library errors (including TPEx "not supported" for `adjprices`/`exrights`) surface with a `fetch tw:` prefix. Bad dates and unknown markets are rejected before any request.
- `Docs/cli-dsl.md` (`fetch` forms, Full Command Index, a Taiwan-stock quickstart from `fetch tw ... adjprices` → `col AdjClose` → `pctchange` → `quant beta`), `skills/use-insyra-cli` (`SKILL.md`, references), and both changelogs (`### CLI`) are updated in the same change.

## Capabilities

### New Capabilities

- `cli-fetch-tw`: the `fetch tw` source in the CLI/REPL/DSL.

### Modified Capabilities

(none — `fetch yahoo` forms are unchanged)

## Impact

- `cli/commands/fetch.go` (dispatch on source), new `cli/commands/fetch_tw.go`, `cli/commands/fetch_tw_test.go`; `cli/commands/config.go` if the interval key needs registration.
- `Docs/cli-dsl.md`, `skills/use-insyra-cli/SKILL.md` and `references/*.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- No library changes, no new dependencies.
