# Proposal: finmind-client

## Why

[#433](https://github.com/HazelnutParadise/insyra/issues/433): FinMind serves a security's whole history in one request, which complements `TWStock`'s whole-market-by-day pages, but `datafetch` had no way to reach it, and the datasets cover things `TWStock` lacks (delisted securities' prices, financial statements, US prices).

## What Changes

- `datafetch.FinMind(cfg)` and `FinMindClient.Dataset(name, dataID, from, to)` return any FinMind dataset as a table with typed columns in FinMind's own order and names, the `date` column as a time. One generic entry point, as the issue asked, not a function per dataset.
- The token goes in an `Authorization` header, not the URL. The request limit (HTTP 402) and a plan without access (HTTP 400 "Your level is ...") are `ErrFinMindLimit` and `ErrFinMindLevel` and are never retried. To support that, `doGet` returns a `finalError` at once and can send fixed headers.
- Documented traps: the month-revenue `date` is the month after (checked live); the others from the issue are marked as reported.
- Not done: typed helper functions for the most used datasets.
- Checked live without a token: a price history, the plan error for a missing `data_id`, and an unknown dataset (HTTP 422).

## Capabilities

### New Capabilities

- `finmind-client`: reading FinMind datasets.

## Impact

`datafetch/finmind.go`, `datafetch/twstock.go` (`finalError`, fixed headers), fixtures, tests, `Docs/datafetch.md`, both CHANGELOGs.
