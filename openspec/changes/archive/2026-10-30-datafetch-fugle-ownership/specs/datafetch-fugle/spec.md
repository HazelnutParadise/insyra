## ADDED Requirements

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
