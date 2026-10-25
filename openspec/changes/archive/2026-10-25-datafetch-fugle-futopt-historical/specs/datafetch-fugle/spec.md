## ADDED Requirements

### Requirement: Futures and options historical candles keep each contract

`client.FutOpt.HistoricalCandles` SHALL fetch a day or minute range over 364 days in consecutive pieces, refuse a week or month range over 364 days, treat a piece with no data as empty, and return the rows in ascending order of date and contract month with a contractMonth column, so that a date with candles for two contracts has two rows.

#### Scenario: A roll day

- **WHEN** a continuous contract has candles for two contract months on one date
- **THEN** the table has two rows for that date, each with its contract month

### Requirement: Options and fields are checked

`HistoricalCandles` SHALL require Strike and CallPut together, with CallPut CALL or PUT and Strike positive and finite, refuse a product containing a slash, a timeframe other than 1, 5, 10, 15, 30, 60, D, W and M, and a field other than open, high, low, close, volume, average, transaction and change, before sending a request.

#### Scenario: Turnover

- **WHEN** the field turnover is asked for
- **THEN** an error is returned and no request is sent

### Requirement: Daily data is one row per contract

`client.FutOpt.HistoricalDaily` SHALL return one trading day of a product with a row per contract carrying date, contractMonth, strikePrice, callPut, the day's prices, volume, volumeSpread, openInterest and settlementPrice, with a field the API leaves out as nil, and an empty table for a day with no data.

#### Scenario: A put without trades

- **WHEN** a contract has no trade
- **THEN** its prices are nil and its volume is 0
