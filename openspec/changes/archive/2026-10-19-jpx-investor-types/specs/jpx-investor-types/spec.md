## ADDED Requirements

### Requirement: Weekly trading by investor type is returned for any range
`JPXClient.InvestorTypes` SHALL return, for each week that overlaps the range, one row per market and investor group with the week's first and last trading day, the market, the investor group, and the sales, purchases, balance (purchases less sales) and total (sales plus purchases) as integers in thousands of yen or thousands of shares as the unit says, using the report week of an older file and the single week of a newer one.

#### Scenario: A week across the year end
- **WHEN** the week of 29 to 30 December 2025 is requested
- **THEN** its WeekStart is 2025-12-29 and its WeekEnd is 2025-12-30

#### Scenario: Both file formats in one range
- **WHEN** the range covers a week from an older .xls file and one from a weekly .xlsx workbook
- **THEN** the rows of both come back in one table in week order with the same fourteen investor groups per market

### Requirement: The fourteen groups are the same in both formats
The investor groups SHALL be Proprietary Cash, Proprietary Margin, Individuals Cash, Individuals Margin, Foreigners Institutions, Foreigners Individuals, Securities Cos., Investment Trusts, Business Cos., Other Cos., Life & Non-Life, City & Regional BK, Trust BK and Other Financials, and a layout that no longer matches the expected headers SHALL be an error rather than misread columns.

#### Scenario: A changed header
- **WHEN** a weekly workbook's group headers differ from the expected ones
- **THEN** an error names the group and what was expected

### Requirement: Unusable calls are refused
`JPXClient.InvestorTypes` SHALL return an error for an unknown unit, a from date after the to date or a zero date, and a client not made by JPX, and SHALL return an empty table with its columns for a range with no files.

#### Scenario: A range after the latest week
- **WHEN** no file covers the range
- **THEN** the table has no rows and no error is returned
