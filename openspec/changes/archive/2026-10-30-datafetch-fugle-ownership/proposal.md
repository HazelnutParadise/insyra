# Proposal: datafetch-fugle-ownership

## Why

[#440](https://github.com/HazelnutParadise/insyra/issues/440), stage 4d: the last of four changes for the remaining stock endpoints, approved by the owner on 2026-10-30 ("do all the Fugle stock endpoints"). Fugle serves four after-hours ownership datasets.

## What Changes

- `client.Stock.OwnershipETFHoldings`, `OwnershipInstitutionalTrades`, `OwnershipDirectorHoldings` and `OwnershipTDCCDistribution`, each with a Context form, take a symbol and `FugleOwnershipOptions{From, To, Descending}`.
- The nested lists become long tables with the date on every row; the period works the same way for all four (From alone runs to today, To needs a From), a period of a year or more is cut into pieces, and director holdings are read by month.
