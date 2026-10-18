## ADDED Requirements

### Requirement: Any day's closes are returned with the exchange's own mark
`TWStockClient.DailyQuotesOn` SHALL return one row per security for the chosen day with the columns of AllDailyQuotes and a Sign of "+", "-", "=" or "X", with Change signed from the exchange's reference price and nil under "X", and SHALL return an empty table with its columns, not an error, for a day without data.

#### Scenario: A falling stock
- **WHEN** the TWSE marks a security with a green minus and a price difference of 0.50
- **THEN** its Sign is "-" and its Change is -0.50

#### Scenario: A weekend
- **WHEN** the date is a Saturday
- **THEN** the table has no rows and the call returns no error

### Requirement: Advance and decline counts prefer the official table
`TWStockClient.AdvanceDecline` SHALL return the TWSE's published 漲跌證券數合計 counts, the stock column when commonStockOnly is set and the whole-market column otherwise, with the limit-up and limit-down counts and Source "official", and otherwise SHALL count from the closes' marks with Source "computed" and nil limit counts, keeping only four-digit codes starting with 1 to 9 and not 91 when commonStockOnly is set.

#### Scenario: A day with the official table
- **WHEN** the TWSE published 684 stocks up (19 at the limit) for a day
- **THEN** Advancing is 684, AdvancingLimit is 19 and Source is "official"

#### Scenario: A day without it
- **WHEN** the day has no 漲跌證券數合計 table
- **THEN** Source is "computed" and the limit counts are nil

### Requirement: Unusable calls are refused
Both methods SHALL return an error for an unknown market and for a client not made by TWStock.

#### Scenario: A zero client
- **WHEN** a zero TWStockClient is used
- **THEN** an error is returned and no request is sent
