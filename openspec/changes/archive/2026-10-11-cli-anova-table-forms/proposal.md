# Proposal: cli-anova-table-forms

## Why

`long-format-entry-points` gave the library `TwoWayANOVAFromTable`, `RepeatedMeasuresANOVAFromTable` and `FriedmanTestFromTable`, which read a table with one row per observation, the shape R's `aov` and `friedman.test`, statsmodels and pingouin all take. The CLI did not follow: `anova twoway` still wants its data split by hand into a×b lists in row-major order and `anova repeated` one list per subject, although a CLI user's data almost always arrives as a table from `load file.csv`. Splitting a 2×3 design means filtering the table once for each of the six combinations, extracting the value column each time, and passing the six lists in the right order with nothing to catch a mistake. The CLI also has no Friedman test at all, so a repeated-measures design that does not meet ANOVA's assumptions cannot be tested from it.

## What Changes

- `anova twoway <table> <value> <factorA> <factorB>` runs the two-way ANOVA on a table with one row per observation. When the first argument after `twoway` is an integer (level count), it runs the list form `anova twoway <aLevels> <bLevels> <cell1> ...`, even if a table variable in scope happens to be named like an integer. Otherwise, when the first argument is a DataTable variable, it runs the table form.
- `anova repeated <table> <value> <condition> <subject>` does the same for the repeated-measures ANOVA; with DataList variables the existing `anova repeated <subject1> <subject2> ...` runs unchanged.
- A new `friedman` command takes the same two shapes: `friedman <table> <value> <condition> <subject>` and `friedman <subject1> <subject2> [subjectN]`, one list per subject as `anova repeated` takes them. It prints `Q`, the degrees of freedom and the p-value.
- Table column arguments resolve using the long-format string selector convention (Excel letter first, then exact column name).
- Every command's `Usage`, `Forms` and `Examples`, `Docs/cli-dsl.md` (the command index, the command groups and a section on the table forms) and both CHANGELOGs are updated.

## Capabilities

### New Capabilities

- `cli-long-format-tests`: the CLI forms that run the two-way ANOVA, the repeated-measures ANOVA and the Friedman test on a table with one row per observation, and the `friedman` command.

### Modified Capabilities

(none)

## Impact

- Code: `cli/commands/hypothesis.go`.
- Tests: `cli/commands/anova_table_test.go`, `cli/commands/friedman_test.go`, and `cli/commands/port_review_cli_test.go`.
- Docs: `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`.
- No new dependencies. No library change.
