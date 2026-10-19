# Design: twstock-tpex-odd-lot

## What the TPEx offers (checked 2026-10-19)

The first version of this note looked only at the OpenAPI and wrongly said the TPEx has no history. The TPEx website has both reports by date, through the JSON the pages themselves call (found by watching the pages' network requests, then called with `curl`):

| Report | Endpoint | Earliest date with data | Columns |
| --- | --- | --- | --- |
| After-hours odd lot (盤後零股) | `/www/zh-tw/afterTrading/odd?date=YYYY/MM/DD&type=Daily&response=json` | between 2010-12-01 (empty) and 2011-01-03 (562 rows) | 代號, 名稱, 成交股數, 成交筆數, 成交金額, 成交價格, 未成交買價, 未成交買量, 未成交賣價, 未成交賣量 |
| Intraday odd lot (盤中零股) | `/www/zh-tw/afterTrading/oddQuote?date=YYYY/MM/DD&type=Daily&response=json` | 2020-10-26, as on the TWSE (2020-10-20 is empty, 2020-11-02 has 879 rows) | 代號, 名稱, 最後成交價, 漲跌, 首筆成交價, 最高, 最低, 成交股數, 成交金額(元), 成交筆數, 最後買價, 最後買量(股), 最後賣價, 最後賣量(股) |

Both answer in the same envelope the package already reads for other TPEx reports (`stat`, `date`, `tables[0].fields` and `.data`), numbers have thousands separators, and a day with no data gives an empty table. The OpenAPI (`/openapi/v1/tpex_odd_stock`) gives only the latest after-hours day and ignores a date, so it is not the source to use. The earliest after-hours date was bracketed by spot checks (2010-12-01 is empty, 2011-01-03 has data), not pinned to the day; the implementation should find it.

`OddLotQuotes`'s doc comment says the TPEx publishes only the latest day. That is wrong and is corrected by this change.

## Proposed API

```go
// TWSE, already there
func (t *TWStockClient) OddLotQuotes(date time.Time, session TWOddLotSession) (*insyra.DataTable, error)

// new: the same shape for the TPEx
func (t *TWStockClient) TPExOddLotQuotes(date time.Time, session TWOddLotSession) (*insyra.DataTable, error)
func (t *TWStockClient) TPExOddLotQuotesContext(ctx context.Context, date time.Time, session TWOddLotSession) (*insyra.DataTable, error)
```

The tables use the columns `OddLotQuotes` already gives for each session (Date, Code, Name, Volume, Transactions, Turnover, then Open, Close, High, Low for intraday and Price for after-hours, and the last best bid and ask price and volume), so the two exchanges' tables can be stacked. The TPEx intraday table's `漲跌` (change) column is left out, as the TWSE's has none.

Usage:

```go
client, _ := datafetch.TWStock(datafetch.TWStockConfig{Interval: 3 * time.Second, Retries: 5})
tse, _ := client.OddLotQuotes(day, datafetch.TWOddLotAfterHours)
otc, _ := client.TPExOddLotQuotes(day, datafetch.TWOddLotAfterHours)
```

## Decisions for you

1. **Name and shape.** A new `TPExOddLotQuotes(date, session)` beside `OddLotQuotes` keeps the TWSE method's signature and meaning (no break) and reads like the package's other market-specific methods. The alternative, adding a `market` argument to `OddLotQuotes`, changes a released signature, which I would not do on this line. I recommend the new method.
2. **Drop `漲跌` from the intraday table?** It can be recomputed from the close and the previous close, and the TWSE table does not have it. I recommend dropping it so the two tables match.

## Tests

Parse saved responses of both reports (a normal day, an empty day, a holiday), an HTML throttling page gives the existing throttled error, the unknown session value is refused like `OddLotQuotes` does, and the live test behind `INSYRA_LIVE_TESTS` as for the other methods.
