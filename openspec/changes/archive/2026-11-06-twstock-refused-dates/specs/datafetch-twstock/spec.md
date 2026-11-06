## ADDED Requirements

### Requirement: A date the source refuses is reported as a refusal
The system SHALL return an error matching `ErrTWStockRefused`, without retrying it, when a TWSE or TPEx payload status says the queried date is outside what the site serves (查詢日期小於… or 查詢日期大於…). From `DailyPrices` and `DailyPricesAdjusted` the error SHALL name the stock and the refused month, and SHALL never be replaced by a shortened table.

#### Scenario: A month before 2010 on TWSE
- **WHEN** `DailyPrices("2330", 2009-12-01, 2009-12-31, TWMarketTWSE)` gets 「查詢日期小於99年1月4日，請重新查詢!」 with `Retries` 3
- **THEN** the error matches `ErrTWStockRefused`, names 2330, 2009-12-01 to 2009-12-31 and the 2010-01-04 start
- **AND** only one request was sent

#### Scenario: The first served month is unchanged
- **WHEN** `DailyPrices("2330", 2010-01-04, 2010-01-31, TWMarketTWSE)` is called
- **THEN** it returns the rows from 2010-01-04 with a nil error

#### Scenario: Another payload status is not a refusal
- **WHEN** the payload status is any other text that is not no data
- **THEN** the error does not match `ErrTWStockRefused`

### Requirement: Auto falls through to TPEx on a refused month
The system SHALL, under `TWMarketAuto`, ask TPEx for a month TWSE refuses, return TPEx's rows when it has them, and return TWSE's refusal when TPEx has no data for the month.

#### Scenario: A TPEx stock before 2010
- **WHEN** `DailyPrices("8069", 2009-12-01, 2009-12-02, TWMarketAuto)` is refused by TWSE and TPEx answers with rows headed 成交仟股
- **THEN** the rows are returned with `Market` TPEx and `Volume` in shares

#### Scenario: A TWSE stock before 2010
- **WHEN** TWSE refuses the month and TPEx has no data for it
- **THEN** the error matches `ErrTWStockRefused`
