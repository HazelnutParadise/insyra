# Design: twstock-tpex-odd-lot

## What the TPEx offers (checked 2026-10-19)

The first version of this note looked only at the OpenAPI and wrongly said the TPEx has no history. The TPEx website has both reports by date, through the JSON the pages themselves call (found by watching the pages' network requests, then called with `curl`):

| Report | Endpoint | Earliest date with data | Columns |
| --- | --- | --- | --- |
| After-hours odd lot (盤後零股) | `/www/zh-tw/afterTrading/odd?date=YYYY/MM/DD&type=Daily&response=json` | between 2010-12-01 (empty) and 2011-01-03 (562 rows) | 代號, 名稱, 成交股數, 成交筆數, 成交金額, 成交價格, 未成交買價, 未成交買量, 未成交賣價, 未成交賣量 |
| Intraday odd lot (盤中零股) | `/www/zh-tw/afterTrading/oddQuote?date=YYYY/MM/DD&type=Daily&response=json` | 2020-10-26, as on the TWSE (2020-10-20 is empty, 2020-11-02 has 879 rows) | 代號, 名稱, 最後成交價, 漲跌, 首筆成交價, 最高, 最低, 成交股數, 成交金額(元), 成交筆數, 最後買價, 最後買量(股), 最後賣價, 最後賣量(股) |

Both answer in the same envelope the package already reads for other TPEx reports (`stat`, `date`, `tables[0].fields` and `.data`), numbers have thousands separators, and a day with no data gives an empty table. The OpenAPI (`/openapi/v1/tpex_odd_stock`) gives only the latest after-hours day and ignores a date, so it is not the source to use. The earliest after-hours date was bracketed by spot checks (2010-12-01 is empty, 2011-01-03 has data), not pinned to the day; the implementation should find it.

`OddLotQuotes`'s doc comment says the TPEx publishes only the latest day. That is wrong and is corrected by this change.

## API (decided, after a review of the design)

```go
func (t *TWStockClient) OddLotQuotes(date time.Time, session TWOddLotSession, market TWMarket) (*insyra.DataTable, error)
func (t *TWStockClient) OddLotQuotesContext(ctx context.Context, date time.Time, session TWOddLotSession, market TWMarket) (*insyra.DataTable, error)
```

`OddLotQuotes(date, session)` and a separate `TPExOddLotQuotes` were first proposed. Every other `TWStockClient` method that reads either exchange takes `market TWMarket`, and none has a `TPEx` prefix. `OddLotQuotes(date, session)` was still under Unreleased in the changelog, so adding the argument breaks nobody. `TWMarketAuto` follows `DailyQuotesOn`: the TWSE first, the TPEx only if the TWSE has no data for the date.

The tables use the columns the TWSE report gives for each session (Date, Code, Name, Volume, Transactions, Turnover, then Open, Close, High, Low for intraday and Price for after-hours, and the last best bid and ask price and volume), so the two exchanges can be stacked. The TPEx intraday table's `漲跌` (change) column is left out. A price the TPEx shows as `0.00` is nil, as the TWSE's `--` is.

## Tests

Parse saved responses of both reports for both markets, an empty day, an HTML throttling page gives the existing throttled error, the unknown session and unknown market are refused, `TWMarketAuto` reads only the TWSE when it has data and falls through to the TPEx when it has none, and the live test behind `INSYRA_LIVE_TESTS` as for the other methods.
