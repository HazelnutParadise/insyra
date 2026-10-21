# taifex-history Specification

## Purpose
Reading the futures exchange's institutional, large-trader and put/call tables over a date range from its website CSV downloads, in the same columns as the latest-day OpenAPI methods.

## Requirements

### Requirement: Longer histories come from the website
`TAIFEXClient` SHALL provide InstitutionalFuturesHistory, InstitutionalOptionsHistory, LargeTradersFuturesHistory, LargeTradersOptionsHistory and PutCallRatioHistory, each returning every trading day from from to to in the columns of the matching latest-day method, oldest first, read from the website's CSV downloads decoded from Big5.

#### Scenario: A week of put/call ratios
- **WHEN** the range is 2026-10-12 to 2026-10-16
- **THEN** the table has five rows, 2026-10-12 first, with the columns of PutCallRatio

### Requirement: A long range is requested month by month
The history methods SHALL request each calendar month of the range separately, and SHALL return a table with the columns and no rows for a range with no trading day.

#### Scenario: A range across two months
- **WHEN** the range is 2026-09-29 to 2026-10-02
- **THEN** two requests are made and the rows of both are returned in date order

### Requirement: A refused range is an error
The history methods SHALL return an error that quotes the website's message and the range when the website answers with a page instead of a CSV, and SHALL NOT return an empty table for it.

#### Scenario: A date the table does not reach
- **WHEN** the institutional table is asked for June 2023
- **THEN** the error contains the website's "DateTime error" and the date range
