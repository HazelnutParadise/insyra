# Proposal: cli-anova-table-forms

## Why

`long-format-entry-points` gave the library `TwoWayANOVAFromTable`, `RepeatedMeasuresANOVAFromTable` and `FriedmanTestFromTable`, which read a table with one row per observation, the shape R's `aov` and `friedman.test`, statsmodels and pingouin all take. The CLI did not follow: `anova twoway` still wants its data split by hand into a×b lists in row-major order and `anova repeated` one list per subject, although a CLI user's data almost always arrives as a table from `load file.csv`. Splitting a 2×3 design means filtering the table once for each of the six combinations, extracting the value column each time, and passing the six lists in the right order with nothing to catch a mistake. The CLI also has no Friedman test at all, so a repeated-measures design that does not meet ANOVA's assumptions cannot be tested from it.

## What Changes

- `anova twoway <table> <value> <factorA> <factorB>` runs the two-way ANOVA on a table with one row per observation. It is the table form when the first argument after `twoway` is a DataTable variable; otherwise the existing list form `anova twoway <aLevels> <bLevels> <cell1> ...` runs unchanged.
- `anova repeated <table> <value> <condition> <subject>` does the same for the repeated-measures ANOVA; with DataList variables the existing `anova repeated <subject1> <subject2> ...` runs unchanged.
- A new `friedman` command takes the same two shapes: `friedman <table> <value> <condition> <subject>` and `friedman <subject1> <subject2> [subjectN]`, one list per subject as `anova repeated` takes them. It prints `Q`, the degrees of freedom and the p-value.
- Column tokens follow the CLI's one-token rule (#315): a bare token is read as a 0-based number, as letters and as a name, refused when two readings disagree, and `number:`, `index:` and `name:` force one.
- Every command's `Usage`, `Forms` and `Examples`, `Docs/cli-dsl.md` (the command index, the command groups and a section on the table forms) and both CHANGELOGs are updated. The other rank tests (`SingleSampleWilcoxon`, `PairedWilcoxon`, `MannWhitneyU`, `KruskalWallis`) are still library-only and are not part of this change.

## Capabilities

### New Capabilities

- `cli-long-format-tests`: the CLI forms that run the two-way ANOVA, the repeated-measures ANOVA and the Friedman test on a table with one row per observation, and the `friedman` command.

### Modified Capabilities

(none)

## Impact

- Code: `cli/commands/hypothesis.go`.
- Tests: a new CLI test file comparing each table form with the library called on the same table and with the list form on the same data, plus the refusals.
- Docs: `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`. `skills/use-insyra-cli/` teaches no command list and needs no change.
- No new dependencies. No library change.
