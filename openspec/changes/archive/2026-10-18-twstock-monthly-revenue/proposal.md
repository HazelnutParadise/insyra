# Proposal: twstock-monthly-revenue

## Why

[#435](https://github.com/HazelnutParadise/insyra/issues/435): monthly revenue is a Taiwan-specific input to many quantitative studies, and `TWStock` had none; the MOPS summary pages carry every company of a market on one page, saving two thousand per-company requests.

## What Changes

- `TWStockClient.MonthlyRevenue(year, month, market)` reads the MOPS pages (Big5) for the TWSE, the TPEx or both into a typed table with the month, company, industry, revenues in thousands of NT dollars, the page's own percent changes, the year-to-date figures, the company's remark and the page date.
- The page layouts differ between years: from about 2011 a domestic page (`_0`) and a foreign-company page (`_1`) exist, older months have one page with no suffix and no remark column. Each is tried in that order, tested live for 2003-01, 2016-06 and 2025-09.
- `PageDate` is the page's production date, not an announcement date, and the documentation says so.
- Not done: the other MOPS pages (emerging-market and public-company revenues).

## Capabilities

### New Capabilities

- `twstock-monthly-revenue`: monthly company revenue from the MOPS.

## Impact

`datafetch/twstock_revenue.go`, `datafetch/twstock.go` (a MOPS base URL), fixtures, tests, `Docs/datafetch.md`, both CHANGELOGs.
