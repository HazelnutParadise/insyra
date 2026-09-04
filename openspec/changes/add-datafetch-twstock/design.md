# Design: add-datafetch-twstock

## Context

`datafetch/yfinance.go` wraps a third-party client behind `YFinanceConfig{Timeout, Interval, UserAgent, Retries, RetryBackoff, Concurrency}` with `normalize()` defaults, `beforeRequest()` throttling, `sleepBackoff(attempt)`, and `normalizeDateColumns` that turns date-like string columns into `time.Time`. `datafetch/geocoding.go` is a hand-written `net/http` client with the same knobs, retry classification (`geocodeRetryable`), a rate-limit error type, and a live test that skips on quota. Endpoints verified on 2026-09-04 without credentials:

| Data | TWSE | TPEx |
| --- | --- | --- |
| Per-stock daily, one month per call | `www.twse.com.tw/rwd/zh/afterTrading/STOCK_DAY?date=YYYYMMDD&stockNo=&response=json` (`stat`, `fields`, `data`) | `www.tpex.org.tw/www/zh-tw/afterTrading/tradingStock?code=&date=YYYY/MM/DD&response=json` (`tables[0].data`) |
| Institutional per day | `rwd/zh/fund/T86?date=&selectType=ALL&response=json` | `www/zh-tw/insti/dailyTrade?type=Daily&date=YYYY/MM/DD&response=json` |
| Margin per day | `rwd/zh/marginTrading/MI_MARGN?date=&selectType=ALL&response=json` (`tables[1]` is per-stock) | TPEx margin endpoint under `www/zh-tw/margin/` |
| Latest full quote table | `openapi.twse.com.tw/v1/exchangeReport/STOCK_DAY_ALL` (JSON array, `Date: "1150903"`) | `www.tpex.org.tw/openapi/v1/tpex_mainboard_daily_close_quotes` |

Dates arrive as ROC strings (`115/09/01`, `1150903`), numbers with thousands separators, and placeholders (`--`, `X`, empty). Three rapid calls were not rate-limited; the real ceiling is unknown.

## Goals / Non-Goals

**Goals:**
- One client, one config shape, the four tables a desk uses daily, typed columns with stable English names.
- Parsing pinned by recorded fixtures so an upstream header change fails a test, not a user's pipeline.
- No network in the default test run.

**Non-Goals:**
- Intraday or real-time quotes, order books, the full 143+225 endpoint catalogue, corporate actions, fundamentals, dividend adjustment, a local cache. Each is a later change if asked for.
- A generic OpenAPI code generator; hand-mapped tables are auditable and the catalogue is not needed.

## Decisions

### Hand-written `net/http` client, shaped like the geocoding client

There is no maintained Go SDK for these endpoints, and the geocoding client already demonstrates the house pattern (config normalize, throttle, retry classification, live-test gate). The base URLs are fields on the private client so fixture tests inject an `httptest.Server`; the exported constructor sets the real hosts.

### Legacy JSON endpoints for history, OpenAPI for the snapshot

The OpenAPI tables carry only the latest day; only the older `rwd`/`www` endpoints accept a date. `DailyPrices` therefore pages month by month through the legacy endpoint (the only way to get history without a licensed feed), and `AllDailyQuotes` uses OpenAPI because it is the only endpoint that returns every security in one call. The docs say a ten-year backfill is roughly 120 requests per stock and recommend `Interval >= 300ms`.

### Header mapping by Chinese field name, not by column position

Both exchanges publish `fields` arrays; mapping by name survives column reordering and fails loudly (error naming the missing header) when a header is renamed. Position-based parsing would silently shift values into the wrong column.

### Typed columns built directly, not via `normalizeDateColumns`

ROC dates and comma numbers need explicit conversion; the yfinance heuristic date normalizer would not recognise `115/09/01`. A small `twstock_parse.go` holds `parseROCDate`, `parseNumber` (comma stripping, placeholder → nil), and the per-table field maps, each unit-tested.

### `TWMarketAuto` tries TWSE first

Listed (上市) codes outnumber OTC (上櫃) ones and TWSE answers "查無資料" (`stat != "OK"`) cheaply, so Auto costs one extra request only for OTC codes. The market that answered is recorded in a `Market` column so callers can cache it.

## Risks / Trade-offs

- [Exchange changes a header or endpoint] → fixture tests fail on the mapping; the live test (gated) shows the new shape. Documented as an operational risk.
- [Rate limiting or IP blocking on large backfills] → throttle default and `Retries`/`RetryBackoff` mirror yfinance; a 4xx that persists after retries is returned, never swallowed. The ceiling is documented as unmeasured.
- [Licence] → docs link the TWSE open-government licence and the TPEx terms; Insyra does not redistribute data.
- [Auto market ambiguity for a code listed on both] → not possible for equities; documented.

## Open Questions

None that block implementation. The TPEx margin endpoint was confirmed on 2026-09-04 as `www.tpex.org.tw/www/zh-tw/margin/balance?date=YYYY/MM/DD&response=json` (`tables[0]`, 920 rows, fields 代號／名稱／資餘額／券餘額…).
