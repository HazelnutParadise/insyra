# Design: stats-cronbach-alpha

Draft for #734. Nothing is implemented yet.

## Problem

A Likert questionnaire is checked for internal consistency before its items are summed into a score or factor-analysed. `stats` has `CorrelationMatrix` and `FactorAnalysis` (checked against R's `psych`), but no reliability coefficient, so a user leaves Insyra for R or SPSS for the first step.

## How the reference tools do it

- **R `psych::alpha`** (psych 2.6.5, the version the project's baselines use): raw alpha `n/(n-1) * (1 - tr(C)/sum(C))` from the item covariance matrix `C`; standardized alpha `n/(n-1) * (1 - n/sum(R))` from the correlation matrix `R`; per item `r.drop`, the correlation of the item with the sum of the others, `cov(i, rest)/sqrt(var(i) var(rest))`; `alpha.drop`, every statistic with the item removed; item mean and SD (`sd`, n - 1). Missing values: `use = "pairwise"` by default. Items with no variance are dropped with a warning. `keys` reverses named items, as `min + max - x` over all items; `check.keys` does it automatically. (Read from `?alpha` and the function's source in psych 2.6.5.)
- **SPSS RELIABILITY** (IBM SPSS Statistics 31 syntax reference): `MODEL=ALPHA` reports alpha and "Standardized item α"; `SUMMARY=TOTAL` reports, per item, the scale mean and variance without it, the item's correlation with the scale without it, and alpha without it. Missing values are excluded listwise: it "deletes cases from analysis if they have a missing value for any variable named on the VARIABLES subcommand." No interval is offered for alpha. (https://www.ibm.com/docs/en/spss-statistics/31.0.0?topic=reference-reliability and its MISSING and SUMMARY pages.)
- **Formulas**: Cronbach (1951), "Coefficient alpha and the internal structure of tests", *Psychometrika* 16, 297–334, for raw alpha; standardized alpha is the Spearman–Brown form `k r̄ / (1 + (k - 1) r̄)` of the mean inter-item correlation `r̄`, algebraically the expression psych uses.

## Proposed API

```go
// CronbachAlphaOptions are the optional settings of CronbachAlpha.
type CronbachAlphaOptions struct {
	// ReverseItems are the items scored in the opposite direction, as column
	// selectors. Each is replaced by min + max - x, min and max taken over all
	// items, as psych's keys does.
	ReverseItems []any
}

// CronbachAlphaResult is the reliability of a scale.
type CronbachAlphaResult struct {
	Alpha                    float64 // raw alpha, from the item covariances
	StandardizedAlpha        float64 // from the mean inter-item correlation
	MeanInterItemCorrelation float64
	NumItems                 int
	NumObservations          int // the rows used: those with every item a finite number
	Items                    []ItemReliability
}

// ItemReliability is one item's part of the scale.
type ItemReliability struct {
	Name                          string
	Mean, SD                      float64 // SD with n - 1
	CorrectedItemTotalCorrelation float64 // the item against the sum of the other items (psych's r.drop)
	AlphaIfDeleted                float64 // raw alpha of the other items
}

func CronbachAlpha(items insyra.IDataTable, opts ...CronbachAlphaOptions) (*CronbachAlphaResult, error)
```

Every column of `items` is an item and every row a respondent; select the items with `ColRange` first. The result has `String()` and `Show()` like every `stats` result.

Example:

```go
survey, _ := insyra.ReadCSVFile("survey.csv")
res, err := stats.CronbachAlpha(survey.ColRange(insyra.Name("q1"), insyra.Name("q6")),
	stats.CronbachAlphaOptions{ReverseItems: []any{insyra.Name("q4")}})
if err != nil {
	return err
}
fmt.Printf("alpha %.3f\n", res.Alpha)
for _, it := range res.Items {
	fmt.Printf("%s: r.drop %.2f, alpha if deleted %.3f\n", it.Name, it.CorrectedItemTotalCorrelation, it.AlphaIfDeleted)
}
```

## Decisions in the draft

- **Missing values: listwise**, as SPSS does and as `stats.FactorAnalysis` already does: a row with a `NaN` or `±Inf` in any item is left out, and `NumObservations` says how many rows were used; a blank or text cell is an error naming the column and the row, from 0, the way every `stats` function refuses one. psych's default is pairwise, which can give a covariance matrix no data set has; a parity test runs psych on complete data, where the two agree.
- **Reverse-keyed items: `ReverseItems` only**, no automatic `check.keys`. psych turned its automatic reversal off by default because users reported alphas without noticing items had been reversed.
- **An item with no variance stays in the scale**, as in SPSS: alpha is defined, the item adds nothing to it, and its corrected item-total correlation is `NaN`. psych drops it; the parity test calls psych with `delete = FALSE`.
- **Errors**: fewer than two items, fewer than two usable rows, or a total score with no variance (alpha undefined).
- **A negative alpha is returned as it is**, the sign of items scored in opposite directions.
- **Out of scope for the first version**: confidence intervals (psych's Feldt and Duhachek–Iacobucci intervals; SPSS offers none), Guttman's lambda 6, `raw.r`/`std.r`/`r.cor`, scale-if-deleted means and variances, and response frequencies.

## Both lines

The function is new, so it ships on `0.4` and on `dev` (0.3.6) with the same signature, cherry-picked from `0.4`.

## Testing

A cross-language parity test against `psych::alpha()` in `stats/testdata/crosslang_baseline.R`'s style: raw and standardized alpha, `r.drop`, `alpha.drop$raw_alpha`, item mean and SD, on a six-item scale, two items, a constant item, a negatively correlated item and a reversed item, and a data set with `NaN` rows compared with psych on its complete cases.
