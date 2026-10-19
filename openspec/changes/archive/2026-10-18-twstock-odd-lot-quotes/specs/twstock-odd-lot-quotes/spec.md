## ADDED Requirements

### Requirement: Odd-lot reports are read into typed tables
`TWStockClient.OddLotQuotes` SHALL return one row per TWSE security for the chosen session with the date, code, name, volume, transactions, turnover, last best bid and ask, and the session's prices as numbers, SHALL give nil for a value shown as "--", and SHALL leave out the totals row.

#### Scenario: Intraday columns
- **WHEN** the intraday session is read for a trading day
- **THEN** the table has Open, Close, High and Low and no Price column

#### Scenario: A price of "--"
- **WHEN** a security did not trade after hours
- **THEN** its Price is nil

### Requirement: A day with no report is an empty table
`TWStockClient.OddLotQuotes` SHALL return an empty table with its columns, not an error, for a day the exchange has no report for, including a date before the intraday report began, and SHALL return an error for an unknown session or a client not made by TWStock.

#### Scenario: A Saturday
- **WHEN** the date is a Saturday
- **THEN** the table has no rows and the call returns no error
