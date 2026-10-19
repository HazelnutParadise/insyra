## ADDED Requirements

### Requirement: The latest TPEx odd-lot report is available
`TWStockClient.TPExOddLotQuotesLatest` SHALL return the TPEx after-hours odd-lot report of the latest trading day with the columns of `OddLotQuotes` after-hours, numbers parsed and a missing price as nil, and SHALL return the throttled error and not parse a throttling page as data.

#### Scenario: A normal day
- **WHEN** the TPEx returns its report
- **THEN** the table has one row per security with Date, Code, Name, Volume, Transactions, Turnover, Price and the four bid and ask columns

#### Scenario: A throttling page
- **WHEN** the TPEx answers with its HTML security page
- **THEN** the call returns the throttled error
