# twstock-monthly-revenue Specification

## Purpose
Reading every Taiwan-listed company's monthly revenue from the MOPS summary pages, one page per market, into a typed table, without the two thousand per-company requests.

## Requirements

### Requirement: Monthly revenue is read from the MOPS summary pages
`TWStockClient.MonthlyRevenue` SHALL return one row per company for the month with the code, name, industry, market, the month's, the previous month's and the year-earlier revenue and the year-to-date figures as integers in thousands of NT dollars, the page's percent changes as numbers, the remark, and the page's production date.

#### Scenario: A cement company
- **WHEN** September 2025 is read for the TWSE
- **THEN** 1101 shows Revenue 13306676 and a YoYChange of -0.13

#### Scenario: Domestic and foreign pages
- **WHEN** a month has a domestic page and a foreign-company page
- **THEN** the rows of both are returned

### Requirement: Old and missing pages are handled
`TWStockClient.MonthlyRevenue` SHALL read the single unsuffixed page of older months, which has no remark column, SHALL return an empty table with its columns for a month with no page, and SHALL return an error for a year before 2003, a month outside 1 to 12, an unknown market and a client not made by TWStock.

#### Scenario: January 2003
- **WHEN** January 2003 is read
- **THEN** the rows come from the unsuffixed page and Remark is empty

#### Scenario: A future month
- **WHEN** no page exists for the month
- **THEN** the table is empty and no error is returned
