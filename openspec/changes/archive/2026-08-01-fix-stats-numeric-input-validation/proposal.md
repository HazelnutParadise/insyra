# Change: Stop `stats` from fabricating zeros for values it cannot read

## Why
`stats` currently has three different answers to "what happens to a value that is not a number", and one of them silently invents data.

Measured on the tree as it stands:

| Entry point | `nil` in an input | a string in an input |
| --- | --- | --- |
| `LinearRegression`, `PolynomialRegression`, `PoissonRegression`, `GLM` | accepted as `0` | accepted as `0` |
| `Correlation`, `CorrelationMatrix`, `CorrelationAnalysis` | accepted as `0` | accepted as `0` |
| `KMeans`, `DBSCAN`, `Hierarchical`, `Silhouette`, `PCA`, `KNN` | rejected | rejected |
| `FactorAnalysis` | listwise deletion | rejected |

The rejecting group and the listwise-deleting group are both defensible: one refuses to guess, the other applies a documented statistical convention. The first group is not a policy at all. It is `DataList.ToF64Slice`, which routes through `insyra.ToFloat64` — a conversion with no failure channel that yields `0` for anything it cannot parse.

The consequence is not a crash but a plausible wrong number. A Pearson correlation over six observations with one `nil` returns `0.9879` where the complete data give `0.9992`, with no error and no warning at the default log level. Nothing downstream can tell that answer apart from a real one, and a regression on a freshly loaded CSV with a blank cell is the ordinary case, not an exotic one.

This is the same failure this project has already named once: an interface satisfied in form but not in substance. `ToF64Slice` returns a `[]float64` of the right length, so every caller believes it was handed the caller's data.

## What Changes
- Add one shared validating converter in `stats` and route every numeric entry point through it, so a value that is not a finite number is refused rather than replaced
- **BREAKING**: `Correlation`, `CorrelationMatrix` and `CorrelationAnalysis` return an error on non-numeric input instead of scoring fabricated zeros
- **BREAKING**: `LinearRegression`, `PolynomialRegression`, `PoissonRegression` and `GLM` return an error on non-numeric input, in the target as well as the predictors
- Name the offending column and row in the error, so the caller can find the cell
- Document, per family, which of the three policies applies, so the surviving differences are stated rather than discovered
- Leave the decision tree alone: it neither rejects nor fabricates, it learns a direction for missing values per node, which is the correct treatment for that family and is already implemented

## Impact
- Affected specs: `stats-regression`
- Affected code: `stats/regression_shared.go`, `stats/regression.go`, `stats/correlation.go`, `stats/glm_predict.go`, `stats/clustering.go`, `Docs/stats.md`
- Behaviour change visible to anyone whose data contains blanks: what used to return a number now returns an error. The number it used to return was wrong.
