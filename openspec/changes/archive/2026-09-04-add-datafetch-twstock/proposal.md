# Proposal: add-datafetch-twstock

## Why

`datafetch` reaches only Yahoo Finance, whose Taiwan coverage is partial and whose fundamentals lag. The Taiwan Stock Exchange (TWSE) and the Taipei Exchange (TPEx) both publish the data a Taiwan research or risk desk actually uses — per-stock daily bars, institutional (三大法人) trades, margin balances, and the full daily quote table — through public HTTP endpoints that need no login or API key. Verified on 2026-09-04: the TWSE OpenAPI (143 endpoints, licence pointing at the Taiwan open-government data licence) and the TPEx OpenAPI (225 endpoints) both answer unauthenticated; historical per-stock bars and per-day institutional tables come from the exchanges' older JSON endpoints, also unauthenticated. Without a connector, every user re-implements Republic-of-China date parsing, comma-formatted numbers, and month-by-month paging.

## What Changes

- New `datafetch.TWStock(cfg TWStockConfig) (*twStock, error)` client mirroring `YFinance`: `Timeout`, `Interval` (throttle), `UserAgent`, `Retries`, `RetryBackoff`, `Concurrency`, all normalized with the same defaults pattern. Requests are throttled and retried with backoff; HTTP 4xx/5xx and a non-`OK` payload status surface as errors.
- `TWMarket` enum: `TWMarketTWSE`, `TWMarketTPEx`, `TWMarketAuto` (try TWSE, then TPEx on a "no data" response).
- Methods, each returning an `*insyra.DataTable` with typed columns (`time.Time` dates converted from ROC years, `float64` prices, `int64` volumes, nil for `--`/blank):
  - `DailyPrices(code string, from, to time.Time, market TWMarket)`: per-stock daily OHLCV, paged month by month over TWSE `afterTrading/STOCK_DAY` or TPEx `afterTrading/tradingStock`, filtered to `[from, to]`, ascending by date.
  - `InstitutionalTrades(date time.Time, market TWMarket)`: the day's 三大法人 buy/sell/net table per stock (TWSE `fund/T86`, TPEx `insti/dailyTrade`).
  - `MarginBalance(date time.Time, market TWMarket)`: the day's margin and short-sale balances per stock (TWSE `marginTrading/MI_MARGN`, TPEx margin endpoint).
  - `AllDailyQuotes(market TWMarket)`: the latest trading day's quote table for every listed security (TWSE OpenAPI `exchangeReport/STOCK_DAY_ALL`, TPEx OpenAPI `tpex_mainboard_daily_close_quotes`).
- Column names are stable English identifiers (`Date`, `Code`, `Name`, `Open`, `High`, `Low`, `Close`, `Volume`, `Turnover`, `Transactions`, …) documented per method; the exchanges' Chinese headers are mapped inside the package so a header rename upstream is caught by the fixture tests, not by users.
- Parsing is tested against recorded JSON fixtures under `datafetch/testdata/twstock/`; live calls run only under `INSYRA_RUN_LIVE_TWSTOCK=1`.
- Docs (`Docs/datafetch.md` new section, README rows), `skills/insyra`, and both changelogs are updated in the same change. The docs state the data licence (TWSE: Taiwan open-government data licence; TPEx: its own terms page) and that history is paged monthly, so a ten-year backfill is ~120 requests per stock.

## Capabilities

### New Capabilities

- `datafetch-twstock`: unauthenticated access to TWSE and TPEx daily prices, institutional trades, margin balances, and full daily quote tables as typed `DataTable`s.

### Modified Capabilities

(none)

## Impact

- New files `datafetch/twstock.go`, `datafetch/twstock_parse.go`, `datafetch/twstock_test.go`, `datafetch/twstock_live_test.go`, fixtures under `datafetch/testdata/twstock/`.
- `Docs/datafetch.md`, `Docs/README.md`, `README.md`, `README_TW.md`, `skills/insyra/SKILL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- No new dependencies (standard `net/http` and `encoding/json`, as the geocoding client uses). No breaking changes. No CLI surface in this change.
