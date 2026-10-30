# datafetch-fugle Specification

## Purpose
A client for Fugle market data, so Taiwan stock candles and intraday quotes reach a DataTable from Go without the Python or Node SDK.
## Requirements
### Requirement: The client sends the key and keeps requests under the limit

`datafetch.Fugle` SHALL send APIKey in the X-API-KEY header unchanged, space requests by Interval (one second when zero, none when negative) and retry a failed request up to Retries times with backoff, stopping when the context is done.

#### Scenario: A key ending in ==

- **WHEN** APIKey is "abc==" and a method is called
- **THEN** the request carries X-API-KEY: abc==

### Requirement: Historical candles cover any range

`Stock.HistoricalCandles` SHALL fetch a day or minute range longer than 364 days in consecutive pieces of at most 364 days and return the rows in ascending date order without duplicates, treat an HTTP 404 piece as having no rows, return an empty table when no piece has rows, and refuse a week or month range over 364 days without sending a request.

#### Scenario: Two and a half years

- **WHEN** a day range of 2.5 years is asked for
- **THEN** three requests are sent, each starting the day after the previous one ended

### Requirement: Intraday methods return tables

`Stock.IntradayQuote` SHALL return one row whose nested objects are joined with underscores and whose bids and asks are numbered from 1; `Stock.IntradayCandles`, `Stock.IntradayTrades`, `Stock.IntradayVolumes` and `Stock.IntradayTickers` SHALL return one row per data item with a field the API leaves out as nil.

#### Scenario: A quote

- **WHEN** the API answers with total.tradeVolume and two bids
- **THEN** the row has total_tradeVolume, bids_1_price and bids_2_size

### Requirement: Failures are typed

The client SHALL return an error matching ErrFugleAuth for HTTP 401 and ErrFuglePlan for HTTP 403 without retrying, and one matching ErrFugleThrottled for HTTP 429 after Retries retries; it SHALL refuse a bad symbol, timeframe, field, range or option before sending a request.

#### Scenario: A plan without the endpoint

- **WHEN** the API answers 403
- **THEN** one request is sent and the error matches ErrFuglePlan

### Requirement: Settings are one optional trailing value

Methods with defaultable settings SHALL take them as one optional trailing options value, refuse more than one before sending a request, and take a required choice such as the ticker type as a parameter.

#### Scenario: Defaults

- **WHEN** `Stock.HistoricalCandles("2330")` is called
- **THEN** it asks for day candles with the API's default range

#### Scenario: Two settings values

- **WHEN** two options values are passed
- **THEN** an error is returned and no request is sent

### Requirement: Documented columns are kept

The tickers, trades and volumes tables SHALL carry every column the documentation lists, as nil cells where the API leaves a field out of every row.

#### Scenario: Tickers without an industry

- **WHEN** every returned ticker lacks an industry
- **THEN** the table still has an industry column of nils

### Requirement: Futures and options intraday methods return tables

`client.FutOpt` SHALL provide IntradayProducts, IntradayTickers, IntradayTicker, IntradayQuote, IntradayCandles, IntradayTrades and IntradayVolumes, each returning a table with one row per data item (one row for a ticker or a quote), the nested objects and arrays of a row flattened into numbered or underscore-joined columns, and a field the API leaves out as nil.

#### Scenario: A stepped price limit

- **WHEN** a ticker has two priceLimitLevels
- **THEN** its row has priceLimitLevels_1_level and priceLimitLevels_2_limitUpPrice columns, nil for a contract without them

### Requirement: The night session is one setting

The futures and options methods SHALL take the night session as AfterHours or an afterHours flag, send session=AFTERHOURS to the products and tickers endpoints and session=afterhours to the others, and send nothing for the day session.

#### Scenario: A night quote

- **WHEN** IntradayQuote is called with afterHours true
- **THEN** the request carries session=afterhours

### Requirement: Spread symbols and timeframes are checked

A futures or options symbol SHALL be accepted with at most one slash between two non-empty parts, each escaped, and refused when empty, with more than one slash, or with a space, a question mark or a hash. Intraday candles SHALL refuse a timeframe other than 1, 5, 10, 15, 30 and 60 minutes before sending a request.

#### Scenario: A spread contract

- **WHEN** IntradayQuote is called with TXFJ6/K6
- **THEN** the request path ends with /TXFJ6/K6

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

### Requirement: Historical statistics are one row

`client.Stock.HistoricalStats` SHALL return one row with the documented columns, and an error, not an empty table, when Fugle answers 404 for an emerging stock or a warrant.

#### Scenario: A listed stock

- **WHEN** HistoricalStats is called for 2330
- **THEN** the row has week52High and week52Low and its date is a time

### Requirement: Snapshots keep their time and take typed choices

`SnapshotQuotes`, `SnapshotMovers` and `SnapshotActives` SHALL take the market, and for movers the direction and the change basis, and for actives the trade basis, as positional values refused unless they are among the documented ones. They SHALL put the snapshot's date, time and market on every row and keep lastPrice as a column that is nil where the API leaves it out.

#### Scenario: A market outside the four

- **WHEN** SnapshotQuotes is given a market that is not TSE, OTC, ESB or TIB
- **THEN** it returns an error and sends no request

### Requirement: Movers take a threshold

`SnapshotMovers` SHALL send a non-zero FugleThreshold as its operator query with the value, refuse an unknown operator and a value that is not finite, and send nothing for the zero value.

#### Scenario: A threshold

- **WHEN** SnapshotMovers is given FugleThreshold{Op: FugleGreaterOrEqual, Value: 5}
- **THEN** the request carries gte=5

### Requirement: The market is typed everywhere

`FugleIntradayTickersOptions.Market` SHALL be a FugleMarket and SHALL be refused unless empty or one of TSE, OTC, ESB and TIB.

#### Scenario: A market outside the four

- **WHEN** IntradayTickers is given the market NYSE
- **THEN** an error is returned and no request is sent

### Requirement: Ownership data is a long table

`OwnershipETFHoldings`, `OwnershipDirectorHoldings` and `OwnershipTDCCDistribution` SHALL return one row per date and listed item with the date on every row, `OwnershipInstitutionalTrades` SHALL return one row per date with each group's buy, sell and net as columns, and every method SHALL keep its documented columns, as nil cells where missing, including in an empty table.

#### Scenario: Two days of ETF holdings

- **WHEN** Fugle lists one holding on the first day and two on the second
- **THEN** the table has three rows and the first day's quantityChange is nil

### Requirement: Ownership periods are cut into pieces

The ownership methods SHALL leave the period to Fugle when neither From nor To is set, run From alone to today, refuse To without From, cut a period of a year or more into requests that each span less than a year (twelve months for director holdings, which are sent as months), and return rows oldest first unless Descending is set.

#### Scenario: Two years of institutional trades

- **WHEN** OwnershipInstitutionalTrades is given From 2024-01-01 and To 2025-12-31
- **THEN** it sends three requests, each spanning less than a year

