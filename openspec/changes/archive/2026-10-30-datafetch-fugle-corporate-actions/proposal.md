# Proposal: datafetch-fugle-corporate-actions

## Why

[#440](https://github.com/HazelnutParadise/insyra/issues/440), stage 4c: the third of four changes for the remaining stock endpoints, approved by the owner on 2026-10-30 ("do all the Fugle stock endpoints"). Fugle lists ex-dividend events, capital changes and listing applications by date.

## What Changes

- `client.Stock.CorporateActionsDividends`, `CorporateActionsCapitalChanges` and `CorporateActionsListingApplicants`, each with a Context form, take the first date as a required parameter and `FugleCorporateActionsOptions{To, Descending}`.
- Rows are oldest first by default, like the candles; the nested `raw` object of a capital change becomes `raw_` columns; every documented column is present and the event dates are times.
