## ADDED Requirements

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
