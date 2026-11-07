## ADDED Requirements

### Requirement: Listings returns each company's stints on each exchange
`TWStockClient.Listings` SHALL return one row per company per exchange stint with Code, Name, Market, Listed, Delisted, Reason and MovedTo: a company still listed with its listing date and a nil Delisted, and an ended listing with a nil Listed and its end date.

#### Scenario: A company that moved to the TWSE
- **WHEN** the TPEx lists 6214 among the listings it ended on 2010-12-30 because the company moved to the TWSE, and the TWSE lists 6214 as listed on 2010-12-30
- **THEN** the table has a TPEx row for 6214 with Delisted 2010-12-30 and MovedTo "TWSE", and a TWSE row with Listed 2010-12-30

#### Scenario: A listing the TWSE ended
- **WHEN** the TWSE's list of ended listings has 2867 ending on 2026-09-01
- **THEN** its row has Market "TWSE", Listed nil, Delisted 2026-09-01 and Reason nil

### Requirement: Listings asks only the exchanges named
`TWStockClient.Listings` SHALL read only the TWSE for TWMarketTWSE, only the TPEx for TWMarketTPEx and both for TWMarketAuto, the TWSE's rows first, then by code.

#### Scenario: The TWSE alone
- **WHEN** Listings is called with TWMarketTWSE
- **THEN** no request goes to the TPEx

### Requirement: Listings fails rather than returning part of an exchange
`TWStockClient.Listings` SHALL return an error, not a partial table, when a list cannot be read, its headers are not the expected ones, or the list of listed companies is empty.

#### Scenario: A year whose headers changed
- **WHEN** the TPEx answers one year with headers that lack 終止上櫃日期
- **THEN** the call returns an error naming that year
