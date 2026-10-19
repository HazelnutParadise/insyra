# Proposal: jpx-investor-types

## Why

[#468](https://github.com/HazelnutParadise/insyra/issues/468): the JPX's weekly trading by investor type is the only public weekly series in a major market with individual investors' actual purchases and sales, and a research project had to write its own downloader and parser for two file formats, merged headers and holiday-shortened weeks.

## What Changes

- `datafetch.JPX(cfg)` and `JPXClient.InvestorTypes(from, to, unit)` return one row per week, market and investor group for the weeks overlapping the range, in thousands of yen or thousands of shares, for 2016 to the latest week.
- Both formats are read: the one-workbook-per-week `.xlsx` of September 2026 onward, with its header checked against the expected layout, and the older `.xls` pairs (value and shares) that hold the previous week beside the report week. The `.xls` files are parsed by a new internal BIFF8 reader, `datafetch/internal/biff`, because the Go libraries tried either dropped cells of these files (`extrame/xls`) or are GPL (`shakinm/xlsReader`).
- The fourteen investor groups are the same in both formats: the cash and margin rows of individuals and of proprietary trading, foreigners' institutions and individuals, and eight others. The cash and margin rows are checked against the blocks they split.
- File names carry random IDs, so the files are found on the current page and the yearly archive pages (years from `from` to the year after `to`).
- Checked live: 2016-12, 2022-03/04 (the market restructuring), the 2025/2026 year end and September to October 2026, in both units, with no layout warnings.
- Not done: a cache of downloaded files; every call downloads again.

## Capabilities

### New Capabilities

- `jpx-investor-types`: the JPX weekly investor-type trading.

## Impact

`datafetch/jpx.go`, `datafetch/jpx_parse.go`, `datafetch/internal/biff`, fixtures, tests, `Docs/datafetch.md`, both CHANGELOGs.
