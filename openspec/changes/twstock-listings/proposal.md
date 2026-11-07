## Why

A whole-market study that sums market value by exchange, or splits stocks by exchange, needs to know which exchange each stock was on at each date; the current classification puts a stock that moved from the TPEx to the TWSE, or that left, in the wrong group (#689). The exchanges publish their listed companies with listing dates and their ended listings with end dates, and the TPEx marks the companies that moved to the TWSE, but nothing in `datafetch` read them.

## What Changes

- `TWStockClient.Listings(market)` and `ListingsContext` return one row per company per exchange stint: `Code`, `Name`, `Market`, `Listed`, `Delisted`, `Reason`, `MovedTo`, built from the TWSE's and the TPEx's lists of listed companies, the TWSE's list of ended listings and the TPEx's, asked year by year, all listings and moves to the TWSE.
- No `ListedOn(date)`: `DailyQuotesOn(date, market)` already reads the whole market of one day, including securities that later left, and the documentation points to it for that question.

## Impact

- New `datafetch/twstock_listings.go`, `Docs/datafetch.md`, both changelogs. No existing behaviour changes.
