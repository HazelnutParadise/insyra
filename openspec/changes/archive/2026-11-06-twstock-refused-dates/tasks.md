# Tasks

- [x] `ErrTWStockRefused`: classify 查詢日期小於 / 查詢日期大於 as a final refusal; name the stock and month for `STOCK_DAY` (`datafetch/twstock.go`).
- [x] `TWMarketAuto` falls through to TPEx on a refusal and reports it when TPEx has no data.
- [x] Map the TPEx header 成交仟股 to `Volume` (`datafetch/twstock_parse.go`).
- [x] Tests: 2009-12 refused and not retried, a range across the floor fails rather than shortens, 2010-01-04 unchanged, `DailyPricesAdjusted` refused, Auto to TPEx with data and without, a future month refused, another status is not a refusal, a 成交仟股 month reads with volume in shares.
- [x] `Docs/datafetch.md`: how far back daily prices go.
- [x] Entries under `## Unreleased` in `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] `openspec validate twstock-refused-dates --strict` passes.
