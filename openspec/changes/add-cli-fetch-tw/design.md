# Design: add-cli-fetch-tw

## Context

`cli/commands/fetch.go` registers one `fetch` command whose `Run` assumes `yahoo` as the source and switches on the method. `datafetch.TWStock` returns an unexported client type with six methods; its tests inject an `http.RoundTripper`, which is not reachable from `cli/commands`. `config` stores CLI-wide keys through `ctx.Env`.

## Goals / Non-Goals

**Goals:** every `TWStock` method reachable from scripts; tests that never touch the network.
**Non-Goals:** caching fetched tables to disk; batch code lists; intraday.

## Decisions

- **Dispatch on source inside `fetch`.** `runFetchCommand` reads `args[0]`: `yahoo` keeps the existing path untouched; `tw` goes to `runFetchTW`. One command, two sources, one `help fetch` page.
- **A small interface plus a factory variable.** `type twStockClient interface { DailyPrices(...); DailyPricesAdjusted(...); ExRights(...); InstitutionalTrades(...); MarginBalance(...); AllDailyQuotes(...) }` and `var newTWStockClient = func(cfg datafetch.TWStockConfig) (twStockClient, error) { return datafetch.TWStock(cfg) }`. Tests swap the variable for a fake that records calls and returns canned tables — the same seam style as the geocoding live-test gate, without needing datafetch internals.
- **Conservative default throttle.** 300 ms / 2 retries is what the datafetch docs recommend for backfills; scripts that need faster can set `fetch.tw.interval_ms`. The CLI is the surface most likely to run a ten-year loop.
- **Dates parsed as `YYYY-MM-DD` UTC** before the client is built, so a typo never costs a request.

## Risks / Trade-offs

- [`market` default `auto` for `adjprices`] → the library refuses when Auto resolves to TPEx, and the error is surfaced; docs say to pass `twse` explicitly for adjusted prices.
- [Config key namespace] → `fetch.tw.interval_ms` follows the dotted style of existing keys; a bad value errors at command time, not at config time, if `config` does not validate.
