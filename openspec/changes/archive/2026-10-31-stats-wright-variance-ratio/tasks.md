# Tasks

- [x] Reference values from R vrtest 1.2 (`stats/testdata/wright_vrtest.R`) on a series with ties and zeros and on a trending series.
- [x] Tests first: the statistics against vrtest, the smallest p-value on a trend, the seed repeating and differing, the nominal size under heavy-tailed independent returns, refused input, the tables.
- [x] `stats/wright_variance_ratio.go`; the statistic and prefix sums in `stats/mathutil.go`; `formatRows` in `stats/result_text.go`; the result slice types and their `String`/`Show`.
- [x] `Docs/stats.md`: the Wright section and `Show()` on the variance ratio and autocorrelation results.
- [x] Entries under `## Unreleased` / `### `stats`` in both CHANGELOGs.
- [x] `openspec validate stats-wright-variance-ratio --strict` passes.
