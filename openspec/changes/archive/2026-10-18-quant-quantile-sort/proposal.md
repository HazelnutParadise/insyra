# Proposal: quant-quantile-sort

## Why

[#458](https://github.com/HazelnutParadise/insyra/issues/458): the usual test of a signal is to sort each day's stocks into groups and compare the group returns. `GroupBy` and `Percentile` cannot cut a different set of groups per date, and cutting 9.6 million stock-days by hand with `GroupBy` is slow.

## What Changes

- `quant.QuantileSort(table, dateCol, signalCol, returnCols, n, opts)` returns the per-day group table, the top-minus-bottom spread table and the skipped days, with options for the fewest stocks per day, tie handling (average rank, or `pandas.qcut` style), a weight column and demeaning.
- `insyra.IsNilTable` is added beside `IsNilList` so a nil table, bare or wrapped, is refused before a method is called on it.
- 1,000,000 rows (500 days of 2,000 stocks) take about 0.27 s.
- Not changed: feeding the spread into a Newey-West t-test, which is `stats.NeweyWestMeanTest`.

## Capabilities

### New Capabilities

- `quant-quantile-sort`: per-date group sorting.

## Impact

`quant/quantile_sort.go`, `datalist_nil.go`, tests, `Docs/quant.md`, both CHANGELOGs.
