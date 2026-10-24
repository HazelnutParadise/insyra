## ADDED Requirements

### Requirement: Any dataset is returned with its own columns
`FinMindClient.Dataset` SHALL return the named dataset for the security and date range as a table whose columns are FinMind's, in FinMind's order, with integers as int64, other numbers as float64, text as text, null as nil, and a column named date as a time when every value is a date.

#### Scenario: A price history
- **WHEN** TaiwanStockPrice for 2330 is read for three days
- **THEN** the columns run date, stock_id, Trading_Volume and on, Trading_Volume is an int64 and date a time

#### Scenario: The token
- **WHEN** a token is configured
- **THEN** it is sent in an Authorization header and not in the URL

### Requirement: Limits and plan errors are recognisable and final
`FinMindClient.Dataset` SHALL return an error matching ErrFinMindLimit for HTTP or body status 402 and one matching ErrFinMindLevel for status 400 with a message about the account's level, SHALL NOT retry either, and SHALL return an error for an empty dataset name, a body that is not JSON and a client not made by FinMind.

#### Scenario: A free account asking for a whole dataset
- **WHEN** no data_id is given and FinMind answers 400 "Your level is free"
- **THEN** the error matches ErrFinMindLevel and one request was sent even with Retries set
