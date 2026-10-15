# Proposal: datalist-quantile

## Why

[#221](https://github.com/HazelnutParadise/insyra/issues/221) (api-review D-13, QU-3) found one concept on three scales: `Quartile(q)` takes 1..3, `Percentile(p)` takes 0..100, and `DescribeOptions.Percentiles` takes 0..1. There was no 0..1 quantile on `DataList` at all, and a 0..1 value given to `Percentile` is accepted silently: `Percentile(0.25)` on `[1 2 3 4 10]` returns 1.01, the 0.25th percentile.

Measured on `0.4` at 6493910b, the family also mishandled missing values:
- `NaN` was sorted into the data. On `[1, 2, 3, NaN]`, `Quartile(1)` was `NaN`, `Quartile(3)` was 2.25 and `Median()` was 1.5, while `Describe` and pandas 3.0.5 give 1.5, 2.5 and 2.
- `Percentile(math.NaN())` panicked with `index out of range`.
- A text cell made `Percentile` record an error and still return a number, while `Quartile` only warned.

The owner ruled on 2026-10-15:
- Add `Quantile(p)` on 0..1, the scale of pandas `Series.quantile`, R `quantile` and gonum `stat.Quantile`.
- Keep `Percentile` on 0..100 and `Quartile` on 1..3, as NumPy keeps both `quantile` and `percentile`. This is recorded in `AGENTS.md` as an exception to the one-name rule of #211.
- Do not add a DataTable form of `quant.WalkForward`, because `DataTable.SliceRows` already takes the callbacks' half-open range.

## What Changes

- **New method.** `DataList.Quantile(p float64) float64` returns the R type-7 quantile (pandas' and NumPy's default `linear`) on 0..1. `IDataList` gains it.
- **Missing values.** `Quantile`, `Percentile`, `Quartile` and `IQR` share one read path. It skips `nil` and `NaN` and fails on any other non-numeric cell, naming the row and returning `NaN`. An out-of-range or `NaN` argument records an error and returns `NaN` instead of panicking. A list with no numeric values returns `NaN` without an error.
- **IQR.** `IQR` reports its own errors under its own name instead of through `Quartile`.
- **Median.** `Median` skips `NaN` cells, so it equals `Quantile(0.5)`.
- **Callers.** `Summary` and the `plot` box plot compute their statistics from the numeric cells. A list with text cells therefore keeps a clean `Err()` after being displayed or plotted.
- **CLI.** New CLI command `quantile <var> <p>`.
- **Docs.**
  - `Docs/DataList.md` gains a section for `Quantile` with the scale table, and the `Quartile`, `IQR`, `Percentile` and `Median` sections are corrected.
  - `Docs/quant.md` states `PercentileBands`' 0..100 scale and shows `WalkForward` with `SliceRows`.
  - `Docs/cli-dsl.md` lists the `quantile` command.

## Capabilities

### New Capabilities

- `datalist-quantile`: the quantile family's scales, definition and treatment of missing and non-numeric cells.

### Modified Capabilities

- `dsl-commands`: the DataList statistics commands include `quantile`.

## Impact

- **Code.** `datalist.go`, `interfaces.go`, `datalist_summary.go`, `plot/boxplot.go` and `internal/dsl/commands/stats_dl_extra.go`, plus tests.
- **Behaviour on fully numeric input.** Unchanged.
- **Behaviour that changes.** Results on lists holding `NaN`, and the error shape for text cells. Both are BREAKING entries in the changelog.
