# Proposal: stats-errors-count-positions-from-zero

## Why

An error that names a bad cell counts its position differently depending on the function: the hypothesis tests say `row 3` for the third cell (one-based, since v0.3.4), while regression's `(index 1)` and factor analysis's `row %d column %d` count from zero. DataTable and DataList take zero-based positions (`Get(0)`), so a one-based `row 3` reads as the table's own number 3, one off from the cell it means. The owner decided on 2026-10-18 that positions count from zero.

## What Changes

- Errors that name a cell position count from zero and say `row index N` (or `condition index N`), in the hypothesis tests, regression, correlation, skewness, kurtosis, moments, clustering input and the long-format loaders.
- Regression, GLM and factor-analysis checks that said `(index N)` or `row N column N` say `row index` / `column index`.
- A group, cell or subject named by the order it was passed (`group 2`, `subject 3`) keeps counting from one.
- Only message text changes; values and error types are unchanged. Documented as breaking error text.

## Capabilities

### Modified Capabilities

- `stats-test-input`: position numbering.

## Impact

`stats/numericinput.go`, `long_format.go`, `regression*.go`, `glm_irls.go`, `factor_analysis.go`, `correlation.go`, `clustering.go`, their tests, `Docs/stats.md`, `AGENTS.md`, both CHANGELOGs.
