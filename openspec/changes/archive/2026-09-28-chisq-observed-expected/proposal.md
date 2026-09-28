# Proposal: chisq-observed-expected

## Why

Two defects in the chi-square tests (#242, ST-6 in `api-review.md`):

- `ChiSquareTestResult.ContingencyTable` stores each cell as a `[2]float64{observed, expected}` array inside a `DataTable`. Nothing else in the library can read such a cell: `Sum()` on a column logs a warning per cell and returns `NaN`, `Show` prints `[5 4.5]`, and `Docs/stats.md` has to tell readers to type-assert every cell by hand. A table of counts that no table operation can use is not a table.
- `ChiSquareGoodnessOfFit(input, p []float64, rescaleP)` matches `p` to the categories after sorting their labels as text, not in the order they appear in the data. Its doc comment carries an IMPORTANT warning about it. A caller who writes the probabilities in the order they think of the categories (`red, green, blue`) gets a test against the wrong distribution and no error: the lengths match, so nothing notices.

## What Changes

- **BREAKING** `ChiSquareTestResult.ContingencyTable` is replaced by two tables of plain `float64` counts, `Observed` and `Expected`, with the same rows and columns. For goodness of fit each has one row per category, named by the category, and one column (`Observed` or `Expected`); for independence, one row per row category and one column per column category, as before. The statistic, p-value and degrees of freedom do not change.
- **BREAKING** `ChiSquareGoodnessOfFit` takes the expected probabilities as `map[string]float64`, keyed by category label. A key that matches no category in the input is an error naming it, and so is a category with no key; nil or an empty map still means equal probabilities. `rescaleP` keeps its meaning. The order of the input no longer matters anywhere.
- `ChiSquareTestResult.Show` prints both tables.
- **BREAKING (CLI)** `chisq gof <var> [label=p ...]` names each proportion's category, `chisq gof colors red=0.5 green=0.3 blue=0.2`, instead of listing bare numbers matched to the sorted labels. A bare number is refused with a message showing the new form.
- Both tests are checked against R's `chisq.test` itself (goodness of fit with `p` and `rescale.p`, independence with `correct = FALSE`), not only against the formula.
- `Docs/stats.md`, `Docs/cli-dsl.md`, both CHANGELOGs, `api-review.md` (ST-6) and `delivery-status.md` are updated.

## Capabilities

### New Capabilities

- `stats-chi-square`: the shape of a chi-square result, how goodness-of-fit probabilities are matched to categories, and the CLI form that passes them.

### Modified Capabilities

(none)

## Impact

- Code: `stats/chi_square.go`, `cli/commands/hypothesis.go`.
- Tests: `stats/chi_square_test.go`, `stats/crosslang_public_methods_test.go`, `stats/fallback_guard_test.go`, a new R reference script calling `chisq.test` with its committed output, and a CLI test for `chisq gof`.
- Docs: `Docs/stats.md`, `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`. The skills teach no chi-square details and need no change.
- No new dependencies.
