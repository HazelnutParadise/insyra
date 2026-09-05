# Proposal: add-datafetch-twstock-adjusted-prices

## Why

`TWStock.DailyPrices` returns raw closes. On an ex-dividend or ex-rights day the price drops by the distribution without any loss to the holder, so a return series built from raw closes shows a fake loss on every ex-date — and `quant.Beta`, `CAPM`, `ValueAtRisk`, and `FactorModel` all consume exactly that series. Yahoo Finance solves this with `Adj Close`; the Taiwan client has nothing equivalent, and the `quant` docs already tell users to choose adjusted prices deliberately. The TWSE publishes the per-ex-date reference prices needed to build the adjustment (verified 2026-09-05: `rwd/zh/exRight/TWT49U` with a date range, unauthenticated), so the gap is a connector and a cumulative-factor computation, not a data-licensing problem.

## What Changes

- New `twStock.ExRights(from, to time.Time, market TWMarket) (*insyra.DataTable, error)`: the exchange's 除權除息計算結果表 for the date range. Columns: `Date` (`time.Time`, the ex-date), `Code`, `Name`, `PrevClose` (除權息前收盤價), `RefPrice` (除權息參考價), `Distribution` (權值+息值), `Kind` (`"dividend"`, `"rights"`, `"both"` from 權/息), `AdjFactor` (`RefPrice / PrevClose`). TWSE is paged in date ranges of at most one year per request. TPEx uses the corresponding TPEx endpoint; if the endpoint cannot be confirmed during implementation, `ExRights` with `TWMarketTPEx` returns an explicit "not supported" error and the docs say so — it is not silently empty.
- New `twStock.DailyPricesAdjusted(code string, from, to time.Time, market TWMarket) (*insyra.DataTable, error)`: `DailyPrices` plus `AdjFactor` (cumulative, 1.0 on and after the last ex-date in range) and `AdjOpen`, `AdjHigh`, `AdjLow`, `AdjClose` (raw price × cumulative factor). Backward adjustment: every bar strictly before an ex-date is multiplied by that ex-date's factor, so the latest bar is unadjusted and earlier bars are scaled down — the same convention as Yahoo `Adj Close`. Ex-dates are fetched for `[from, to]` only; distributions after `to` do not affect the series, and the doc says so.
- The ROC date format `115年06月01日` is added to the date parser.
- Fixture-replayed tests for both methods; the live test gains an `ExRights` and `DailyPricesAdjusted` case. Docs (`Docs/datafetch.md` TWStock section, `Docs/quant.md` alignment recipe now pointing at `AdjClose`), `skills/insyra`, and both changelogs are updated in the same change.

## Capabilities

### New Capabilities

- `datafetch-twstock-adjusted`: ex-rights/ex-dividend tables and backward-adjusted daily prices for Taiwan stocks.

### Modified Capabilities

(none — `DailyPrices` and its columns are unchanged)

## Impact

- `datafetch/twstock.go` (+ two methods), `datafetch/twstock_parse.go` (+ date format, + header map), `datafetch/twstock_test.go`, `datafetch/twstock_live_test.go`, fixture `datafetch/testdata/twstock/twse_twt49u_20260601_20260903.json` (already recorded, 25 rows) and a TPEx fixture if the endpoint is confirmed.
- `Docs/datafetch.md`, `Docs/quant.md`, `skills/insyra/SKILL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- No new dependencies, no breaking changes.
