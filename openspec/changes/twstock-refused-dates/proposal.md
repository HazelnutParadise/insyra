# Proposal: twstock-refused-dates

## Why

[#688](https://github.com/HazelnutParadise/insyra/issues/688): TWSE's `STOCK_DAY` refuses any month before January 2010 with 「查詢日期小於99年1月4日，請重新查詢!」 and any month after the current one with 「查詢日期大於今日」 (measured on 2026-11-07; the site judges by month, so `date=20100101` is answered). `DailyPrices` and `DailyPricesAdjusted` reported it as `TWStock payload status "…"`, which a caller cannot tell from a network or format failure, retried it under `Retries` although asking again cannot help, and the documentation did not state the limit.

While measuring it, two more things turned up on the same path. `TWMarketAuto` treated the refusal as a failure instead of falling through to TPEx, which serves those months (8069 back to at least 2005). And every TPEx month before about 2025 heads its volume column 成交仟股 rather than 成交張數, so `DailyPrices` on TPEx failed with `required header "成交張數" is missing` for all of them; the Auto fall-through is useless without reading them.

## What Changes

- New exported `ErrTWStockRefused`. A payload whose status is 查詢日期小於… or 查詢日期大於… matches it, is final (not retried), and from `STOCK_DAY` names the stock, the refused month and the site's text, with the 2010-01-04 start when that is the reason.
- `TWMarketAuto` asks TPEx for a month TWSE refuses and returns the refusal only when TPEx has no data for it.
- The TPEx daily header 成交仟股 maps to `Volume`; one thousand shares is one lot, so the conversion to shares is unchanged.
- `Docs/datafetch.md` states the limit and points to FinMind `TaiwanStockPrice` for earlier TWSE history.
- Not changed: requests from 2010-01-04 on, and the issue's optional FinMind-based adjusted-price path, which is a feature of its own.

## Capabilities

### Modified Capabilities

- `datafetch-twstock`: refused dates are named, Auto falls through on them, and older TPEx headers are read.

## Impact

`datafetch/twstock.go`, `datafetch/twstock_parse.go`, `datafetch/twstock_refused_test.go`, `Docs/datafetch.md`, both CHANGELOGs.
