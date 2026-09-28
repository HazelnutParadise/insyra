# Proposal: datafetch-own-yfinance-types

## Why

DF-4 of the API review ([#252](https://github.com/HazelnutParadise/insyra/issues/252)): two of `datafetch`'s Yahoo Finance signatures are types from `github.com/wnjoon/go-yfinance`. `YFHistoryParams` is declared as `= models.HistoryParams`, an alias, so its fields and the `*models.RepairOptions` inside it are whatever that package's version says, and `News(count int, tab models.NewsTab)` takes that package's tab type. When go-yfinance renames or adds a field, insyra's API changes with it, in a release of insyra that may say nothing about it, and a caller who wants to pass a tab has to import go-yfinance themselves.

The same finding also questions the Chrome 117 User-Agent the Yahoo Finance client sends by default. Whether Yahoo still answers an honest User-Agent has to be tried before it is changed, so that part is left on the issue for the owner and is not in this change.

## What Changes

- **BREAKING (for callers that named go-yfinance types):** `YFHistoryParams` becomes insyra's own struct with the same fields, JSON tags and meanings, and `RepairOptions` becomes `*YFRepairOptions`, insyra's own copy of the five repair switches. A composite literal such as `datafetch.YFHistoryParams{Period: "1mo", Interval: "1d"}` compiles unchanged; code that passed a `models.HistoryParams` value, or built the repair options from `models`, has to switch to the insyra types.
- **BREAKING:** `News` takes a `YFNewsTab`, with `YFNewsTabNews`, `YFNewsTabAll` and `YFNewsTabPressReleases`. The empty tab keeps meaning news, as it does in go-yfinance. A tab outside those values is now refused with an error before any request; go-yfinance fetched news for it.
- The conversion to go-yfinance's types happens inside `History` and `News`. A test compares the field lists of the two parameter structs, so an upgrade of go-yfinance that adds or renames a history parameter fails the build's tests instead of being dropped without notice, and another test fails if any exported declaration in `datafetch` names a type from a module outside the standard library and insyra.

## Capabilities

### New Capabilities

- `datafetch-yfinance-api`: the Yahoo Finance API in `datafetch` is expressed in insyra's own types.

### Modified Capabilities

(none)

## Impact

- `datafetch/yfinance.go` and its tests. The CLI's `fetch yahoo` builds `YFHistoryParams{}` and passes the empty news tab, so it compiles and behaves as before.
- `Docs/datafetch.md`, both changelogs, `api-review.md` (DF-4 stays open for the User-Agent), `delivery-status.md`.
