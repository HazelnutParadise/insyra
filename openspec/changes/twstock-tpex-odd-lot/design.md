# Design: twstock-tpex-odd-lot

## What the TPEx offers (checked 2026-10-19)

- `https://www.tpex.org.tw/openapi/v1/tpex_odd_stock` ("上櫃股票零股交易資訊") returns 1,010 rows, all dated the latest trading day (ROC `1151019`). Columns: Date, SecuritiesCompanyCode, CompanyName, TradeVolume, NumberOfTransactions, TradeAmount, Price, LastBestBidPrice, LastBestBidVolume, LastBestAskPrice, LastBestAskVolume. One `Price` and no open, high, low: this is the after-hours session.
- A `date` query parameter is ignored: asking for 1151016 still returns the latest day. There is no history through this API.
- The OpenAPI document lists no TPEx intraday odd-lot report. I did not find one on the website either, but did not search exhaustively.

So the TWSE half of #451 can be fetched by date and the TPEx half cannot. `OddLotQuotes`'s doc comment already says so.

## Proposed API

```go
func (t *TWStockClient) TPExOddLotQuotesLatest() (*insyra.DataTable, error)
func (t *TWStockClient) TPExOddLotQuotesLatestContext(ctx context.Context) (*insyra.DataTable, error)
```

The table has Date, Code, Name, Volume, Transactions, Turnover, Price, BidPrice, BidVolume, AskPrice, AskVolume, with the same types as `OddLotQuotes` after-hours (Date as the library's other TWStock tables give it, numbers parsed, a price shown as `--` or empty is nil). It reuses the client's limiter, retry and throttling handling.

## Decisions for you

1. **A separate method, not a new `session` constant.** `OddLotQuotes(date, session)` takes a date; the TPEx call cannot honour one, and hiding that behind a constant would make a past date silently return today's data. A separate name says "latest" up front. Alternative: add `TWOddLotTPExAfterHours` and return an error when `date` is not the latest day. I recommend the separate method.
2. **No history helper.** To build history the caller runs it once a day and stores the result. A helper that stores files would add state to a fetch library. I recommend not adding one.
3. **Do it at all?** It gives one day at a time. Worth it only if you will collect daily from now on. If not, close #451 as done for the TWSE half and note that the TPEx publishes no history.

## Tests

Parse a saved response (fixture, `.csv.txt` rule applies only to CSV), a response with a different date than today is still accepted, an HTML throttling page gives the existing throttled error, an empty array gives an empty table, and the live test behind `INSYRA_LIVE_TESTS` as for the other methods.
