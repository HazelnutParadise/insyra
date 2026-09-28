# Proposal: long-format-entry-points

## Why

`TwoWayANOVA(factorALevels, factorBLevels, cells)` makes the caller split the data into a×b lists and pass them in row-major order, and `RepeatedMeasuresANOVA(subjects)` and `FriedmanTest(subjects)` want one list per subject (#243, ST-7 in `api-review.md`). Data rarely arrives that way. A CSV of an experiment has one row per observation, with the measured value in one column and the factor, condition or subject in others: the long format that R's `aov(value ~ A * B)`, `friedman.test(value ~ cond | subj)` and pandas-based tools such as pingouin take directly. Today every insyra user reshapes it by hand, and the row-major order of `cells` is one more thing to get wrong without an error.

## What Changes

- New `TwoWayANOVAFromTable(dt, valueCol, factorACol, factorBCol any)`, `RepeatedMeasuresANOVAFromTable(dt, valueCol, conditionCol, subjectCol any)` and `FriedmanTestFromTable(dt, valueCol, conditionCol, subjectCol any)`. Each takes a table with one row per observation and picks its columns with the library's one column selector (an Excel-style letter, `insyra.Name(...)` or a 0-based position).
- A factor, condition or subject column's distinct values are its levels, compared the way `GroupBy` compares keys (integers of any width by value; floats and text as themselves). A `nil` or `NaN` level, and a value that is not a finite number, is refused with the row, counted from one.
- The two-way test needs at least two levels of each factor and at least one observation in every combination. The repeated-measures tests need exactly one observation for every subject under every condition; a missing or duplicated one is refused, naming the subject and the condition.
- The results are those of the existing list-taking functions called with the levels in the order they first appear, bit for bit; the existing functions do not change.
- The new entry points are checked against R's `aov` and `friedman.test` on long data in shuffled row order.
- `Docs/stats.md` documents the three functions and gains the `RepeatedMeasuresANOVA` section it never had; both CHANGELOGs, `api-review.md` (ST-7) and `delivery-status.md` are updated.

## Capabilities

### New Capabilities

- `stats-long-format`: the long-format entry points to the two-way ANOVA, the repeated-measures ANOVA and the Friedman test.

### Modified Capabilities

(none)

## Impact

- Code: a new `stats/long_format.go`. `stats/anova.go` and `stats/nonparam_friedman.go` are not touched.
- Tests: a new `stats/long_format_test.go`, and an R script calling `aov` and `friedman.test` with its committed output.
- Docs: `Docs/stats.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`. The skills teach no function list and need no change; the CLI's `anova` forms are unchanged, since the list-taking functions they call stay.
- No new dependencies.
