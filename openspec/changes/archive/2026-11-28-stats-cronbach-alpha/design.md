# Design: stats-cronbach-alpha

Draft for #734. Nothing is implemented yet.

## Problem

A Likert questionnaire is checked for internal consistency before its items are summed into a score or factor-analysed. `stats` has `CorrelationMatrix` and `FactorAnalysis` (checked against R's `psych`), but no reliability coefficient, so a user leaves Insyra for R or SPSS for the first step.

## How the reference tools do it

- **R `psych::alpha`** (psych 2.6.5, the version the project's baselines use): raw alpha `n/(n-1) * (1 - tr(C)/sum(C))` from the item covariance matrix `C`; standardized alpha `n/(n-1) * (1 - n/sum(R))` from the correlation matrix `R`; per item `r.drop`, the correlation of the item with the sum of the others, `cov(i, rest)/sqrt(var(i) var(rest))`; `alpha.drop`, every statistic with the item removed; item mean and SD (`sd`, n - 1). Missing values: `use = "pairwise"` by default. Items with no variance are dropped with a warning. `keys` reverses named items, as `min + max - x` over all items; `check.keys` does it automatically. (Read from `?alpha` and the function's source in psych 2.6.5.)
- **SPSS RELIABILITY** (IBM SPSS Statistics 31 syntax reference): `MODEL=ALPHA` reports alpha and "Standardized item α"; `SUMMARY=TOTAL` reports, per item, the scale mean and variance without it, the item's correlation with the scale without it, and alpha without it. Missing values are excluded listwise: it "deletes cases from analysis if they have a missing value for any variable named on the VARIABLES subcommand." No interval is offered for alpha. (https://www.ibm.com/docs/en/spss-statistics/31.0.0?topic=reference-reliability and its MISSING and SUMMARY pages.)
- **Formulas**: Cronbach (1951), "Coefficient alpha and the internal structure of tests", *Psychometrika* 16, 297–334, for raw alpha; standardized alpha is the Spearman–Brown form `k r̄ / (1 + (k - 1) r̄)` of the mean inter-item correlation `r̄`, algebraically the expression psych uses.

## Proposed API

Revised after an independent review: the items are chosen in the call, reversal no longer shifts means, and a constant item is refused.

```go
// CronbachAlphaOptions are the optional settings of CronbachAlpha.
type CronbachAlphaOptions struct {
	// ReverseScored are the items scored in the opposite direction, as column
	// selectors of the table passed to CronbachAlpha.
	ReverseScored []any
}

// CronbachAlphaResult is the internal consistency of a scale.
type CronbachAlphaResult struct {
	Alpha                    float64 // raw alpha, from the item covariances
	StandardizedAlpha        float64 // from the mean inter-item correlation
	MeanInterItemCorrelation float64
	NumItems                 int
	N                        int // rows used: those with every item a finite number
	RowsDropped              int // rows left out because an item was NaN or ±Inf
	Items                    []CronbachAlphaItem
}

// CronbachAlphaItem is one item's part of the scale.
type CronbachAlphaItem struct {
	Name     string
	Reversed bool
	Mean, SD float64 // of the item as it is in the table, SD with n - 1
	// ItemRestCorrelation is the correlation of the item with the sum of the
	// other items: SPSS's Corrected Item-Total Correlation, psych's r.drop.
	ItemRestCorrelation float64
	AlphaIfDeleted      float64 // raw alpha of the other items; NaN for a two-item scale
}

func CronbachAlpha(dt insyra.IDataTable, items []any, opts ...CronbachAlphaOptions) (*CronbachAlphaResult, error)
```

`items` are column selectors of `dt`; nil or empty means every column. Selectors in `ReverseScored` are read against the same table, so `"D"` is the table's column D whichever items are chosen. The result has `String()` and `Show()` like every `stats` result.

Example:

```go
survey, _ := insyra.ReadCSVFile("survey.csv")
res, err := stats.CronbachAlpha(survey,
	[]any{insyra.Name("q1"), insyra.Name("q3"), insyra.Name("q4"), insyra.Name("q7")},
	stats.CronbachAlphaOptions{ReverseScored: []any{insyra.Name("q4")}})
if err != nil {
	return err
}
res.Show()
```

## Decisions in the draft

- **Missing values: listwise**, as SPSS does and as `stats.FactorAnalysis` already does: a row with a `NaN` or `±Inf` in any item is left out and counted in `RowsDropped`. A blank numeric cell in a CSV is read as `NaN`, so a skipped question drops its row. A text cell is an error naming the column and the row, from 0. psych's default is pairwise, which can give a covariance matrix no data set has; the parity test runs psych on complete data, where the two agree.
- **Reverse-scored items are negated internally.** Alpha, SD and the item-rest correlation do not depend on the constant psych adds (`min + max - x` over all items), which only shifts the mean, wrongly when items have different ranges. `Mean` is the item's own mean and `Reversed` marks the item. There is no automatic `check.keys`: psych turned it off by default because users reported alphas without noticing items had been reversed.
- **An item with no variance is an error naming it.** Its correlations are 0/0, so standardized alpha and the mean inter-item correlation would be undefined; the caller drops the column knowingly. psych drops it with a warning.
- **Two items:** alpha is defined, but alpha with an item deleted is the alpha of one item, which is not defined, so `AlphaIfDeleted` is `NaN`.
- **Errors**: a nil or empty table; fewer than two items; an item or reverse-scored selector that is not a column, or given twice; a reverse-scored item that is not among the items; fewer than two usable rows; more than one options value. A negative alpha is returned as it is.
- **Out of scope for the first version**: confidence intervals (psych's Feldt and Duhachek–Iacobucci intervals; SPSS offers none), Guttman's lambda 6, `raw.r`/`std.r`/`r.cor`, scale-if-deleted means and variances, and response frequencies.

## Both lines

The function is new, so it ships on `0.4` and on `dev` (0.3.6) with the same signature, cherry-picked from `0.4`.

## Testing

A cross-language parity test against `psych::alpha()` in `stats/testdata/crosslang_baseline.R`'s style: raw and standardized alpha, `r.drop`, `alpha.drop$raw_alpha`, item mean and SD, on a six-item scale, two items (alpha only), a negatively correlated item and a reversed item with equal ranges, and a data set with `NaN` rows compared with psych on its complete cases; the constant item is an error, tested on its own.
