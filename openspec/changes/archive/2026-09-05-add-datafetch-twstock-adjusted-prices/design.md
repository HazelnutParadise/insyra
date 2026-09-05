# Design: add-datafetch-twstock-adjusted-prices

## Context

`datafetch/twstock.go` (archived change `add-datafetch-twstock`, 2026-09-04) has the client, `doJSON` with throttle/retry, `parseROCDate` for `115/09/01` and `1150903`, `mapHeaders` by Chinese field name, and `DailyPrices` paging month by month into `dailyPriceRecord`s. The TWSE ex-rights table was verified on 2026-09-05: `https://www.twse.com.tw/rwd/zh/exRight/TWT49U?startDate=20260601&endDate=20260903&response=json` returns `stat: OK`, `fields` (資料日期, 股票代號, 股票名稱, 除權息前收盤價, 除權息參考價, 權值+息值, 權/息, 漲停價格, 跌停價格, 開盤競價基準, 減除股利參考價, 詳細資料, …) and 920 rows for three months; dates come as `115年06月01日`. A 25-row fixture is recorded at `datafetch/testdata/twstock/twse_twt49u_20260601_20260903.json`. The old `afterTrading/TWT49U` path and the OpenAPI `exchangeReport/TWT49U` both 404. The TPEx OpenAPI lists `tpex_exright_daily` (today only); its dated legacy endpoint was not reachable during probing (TPEx answered `error code: 520` to every request after the fixture-recording burst), so it is unconfirmed.

## Goals / Non-Goals

**Goals:**
- A return series from Taiwan prices that does not fake a loss on ex-dates.
- The ex-rights table itself, since users also want the distribution amounts.
- The Yahoo `Adj Close` convention exactly, so `quant` docs can say "use `AdjClose`" for both sources.

**Non-Goals:**
- Total-return indices, reinvestment of cash dividends at a different price than the reference, capital reductions (減資) and stock splits beyond what the exchange folds into 除權息參考價, intraday adjustment, adjusting `Volume`.
- Guessing the TPEx endpoint: an unverified URL would fail silently in production. TPEx support lands when the endpoint is confirmed; until then it is an explicit error.

## Decisions

### Factor from the exchange's own reference price

`AdjFactor = 除權息參考價 / 除權息前收盤價` is what the exchange computes from the distribution, including rights and the tax-free portion; it needs no dividend arithmetic on our side and matches the price actually used as the ex-day base. Yahoo's factor is the same ratio expressed through dividends and splits.

### Backward adjustment, latest bar unadjusted

Multiplying earlier bars by later factors keeps today's price equal to the quoted price and makes a `PctChange` across the ex-date equal to the true holder return. This is the Yahoo convention and what the `quant` alignment recipe expects. Factors compound across multiple ex-dates; the loop walks ex-dates descending and applies a running product to bars strictly before each.

### Ex-dates come from the same range as the prices

`DailyPricesAdjusted(from, to)` calls `ExRights(from, to)` and filters by `Code`. Distributions after `to` are not applied, so two calls with different `to` give different adjusted histories — inherent to backward adjustment and documented. Callers wanting a stable series extend `to` to today.

### Separate method, not an option on `DailyPrices`

`DailyPrices` shipped a day ago unreleased, so changing its signature is allowed, but an `Adjusted bool` would make the column set depend on a flag. A second method with a superset of columns keeps both tables predictable.

### Long ROC date format in the shared parser

`115年06月01日` joins the two existing formats in `parseROCDate`; a table-specific parser would duplicate the validation.

### TWSE paging by year

TWT49U accepts arbitrary ranges but the server-side cap is undocumented; one-year slices were verified to return and keep requests bounded.

## Risks / Trade-offs

- [TPEx unsupported at first] → explicit error and docs note; a follow-on change adds it once the endpoint is confirmed, with the same columns.
- [Exchange revises a reference price after the fact] → the table is whatever the exchange serves on the day of the call; no caching, documented.
- [Users compare `AdjClose` from Yahoo and TWSE] → both are backward-adjusted ratios but Yahoo's dividend inputs differ (Yahoo uses cash dividends only, TWSE folds rights in), so small differences are expected; documented.

## Open Questions

- The TPEx dated ex-rights endpoint. Probed again on 2026-09-05 after the block lifted: the OpenAPI `openapi/v1/tpex_exright_daily` answers (fields `Date`, `SecuritiesCompanyCode`, `CompanyName`, `ClosePriceBeforeExRightsDiviend`, `ExRightsDiviendQuote`, `StockDividend`, `CashDividend`, `StockDividendPlusCashDividend`, …) but carries only the current announcement list, not history; the guessed dated paths under `www/zh-tw/exright/`, `bulletin/`, and `afterTrading/` redirect or fail. A history endpoint was not found, so TPEx `ExRights` ships as the explicit "not supported" error. Implementation may add the OpenAPI snapshot as a documented "upcoming ex-dates" convenience only if it costs nothing; it must not be used to adjust history.
