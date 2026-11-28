## Why

`TAIFEXClient.InstitutionalFuturesHistory` and `InstitutionalOptionsHistory` read the exchange's website, which serves the trading days of the last three years, rolling (measured 2026-11-07: 2023-11-07 answers, 2023-11-01 is refused). A study of the futures basis against institutional positions needs the series from 2018 (#687). FinMind's `TaiwanFuturesInstitutionalInvestors` and `TaiwanOptionInstitutionalInvestors` have it from 2018-06-05, but under other column names, other trader names and in another row order, so every caller had to map them by hand. The documentation also still said the website serves the institutional tables "from 2024".

## What Changes

- `FinMindClient.InstitutionalFutures(contract, from, to)` and `InstitutionalOptions(contract, from, to)`, each with a Context form, return one contract's days in the columns of `TAIFEXClient.InstitutionalFutures` and `InstitutionalOptions`: the exchange's contract name (read once per client from `TaiwanFutOptDailyInfo`), its trader names and row order, `CALL`/`PUT`, and the Net columns computed as Long minus Short.
- The godoc of the TAIFEX history methods, `ErrTAIFEXRefused` and `Docs/datafetch.md` describe the rolling three-year window and point to the FinMind methods for earlier days.

## Impact

- `datafetch/finmind.go` (the request is shared by `Dataset` and the new methods), new `datafetch/finmind_institutional.go`, `datafetch/taifex_history.go` comments, `Docs/datafetch.md`, both changelogs. No existing behaviour changes.
