# Proposal: twstock-odd-lot-quotes

## Why

[#451](https://github.com/HazelnutParadise/insyra/issues/451): studying odd-lot trading as a retail-sentiment signal needs the TWSE's daily odd-lot reports, and `datafetch.TWStock` had none, so the research code fetched them itself.

## What Changes

- `TWStockClient.OddLotQuotes(date, session)` and its `Context` form read the TWSE intraday (`TWTC7U`) and after-hours (`TWT53U`) odd-lot reports into a `DataTable` with typed numbers, `--` as nil and the exchange's totals row dropped. A holiday or a date before 2020-10-26 (intraday) gives an empty table.
- It goes through the same client as the other methods, so `Interval`, `Retries` and the throttling error of `twstock-report-throttling` apply.
- Not done from the issue: the TPEx reports, since the TPEx's OpenAPI gives only the latest day, and resumable range backfills. The issue stays open for them.
- Checked against the live TWSE endpoints for 2026-01-02 (1,316 securities each) and against recorded fixtures.

## Capabilities

### New Capabilities

- `twstock-odd-lot-quotes`: the TWSE odd-lot reports.

## Impact

`datafetch/twstock_oddlot.go`, `datafetch/testdata/twstock`, tests, `Docs/datafetch.md`, both CHANGELOGs.
