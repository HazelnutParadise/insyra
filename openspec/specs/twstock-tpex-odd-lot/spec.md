# twstock-tpex-odd-lot Specification

## Purpose
The Taipei Exchange's intraday and after-hours odd-lot reports by date, in the same shape as the TWSE's, so the two exchanges can be studied together.

## Requirements

### Requirement: TPEx odd-lot reports are available by date
`TWStockClient.OddLotQuotes` with `TWMarketTPEx` SHALL return the TPEx intraday or after-hours odd-lot report of the given date with the columns of `OddLotQuotes` for that session, numbers parsed and a missing price as nil, SHALL return an empty table for a date with no data, and SHALL return the throttled error and not parse a throttling page as data.

#### Scenario: An after-hours day
- **WHEN** the date is a trading day and the session is after-hours
- **THEN** the table has one row per security with Date, Code, Name, Volume, Transactions, Turnover, Price and the four bid and ask columns

#### Scenario: A date before the session existed
- **WHEN** the session is intraday and the date is 2020-10-20
- **THEN** the table is empty and there is no error

#### Scenario: A throttling page
- **WHEN** the TPEx answers with its HTML security page
- **THEN** the call returns the throttled error
