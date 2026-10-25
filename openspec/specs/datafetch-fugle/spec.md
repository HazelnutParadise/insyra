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

`StockHistoricalCandles` SHALL fetch a day or minute range longer than 364 days in consecutive pieces of at most 364 days and return the rows in ascending date order without duplicates, treat an HTTP 404 piece as having no rows, return an empty table when no piece has rows, and refuse a week or month range over 364 days without sending a request.

#### Scenario: Two and a half years

- **WHEN** a day range of 2.5 years is asked for
- **THEN** three requests are sent, each starting the day after the previous one ended

### Requirement: Intraday methods return tables

`StockIntradayQuote` SHALL return one row whose nested objects are joined with underscores and whose bids and asks are numbered from 1; `StockIntradayCandles`, `StockIntradayTrades`, `StockIntradayVolumes` and `StockIntradayTickers` SHALL return one row per data item with a field the API leaves out as nil.

#### Scenario: A quote

- **WHEN** the API answers with total.tradeVolume and two bids
- **THEN** the row has total_tradeVolume, bids_1_price and bids_2_size

### Requirement: Failures are typed

The client SHALL return an error matching ErrFugleAuth for HTTP 401 and ErrFuglePlan for HTTP 403 without retrying, and one matching ErrFugleThrottled for HTTP 429 after Retries retries; it SHALL refuse a bad symbol, timeframe, field, range or option before sending a request.

#### Scenario: A plan without the endpoint

- **WHEN** the API answers 403
- **THEN** one request is sent and the error matches ErrFuglePlan

