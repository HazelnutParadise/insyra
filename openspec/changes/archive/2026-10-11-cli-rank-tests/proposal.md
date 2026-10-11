# Proposal: cli-rank-tests

## Why

The CLI runs the t-, z-, F- and chi-square tests and the ANOVAs, and since `cli-anova-table-forms` the Friedman test, but none of the other rank-based tests: `SingleSampleWilcoxon`, `PairedWilcoxon`, `MannWhitneyU` and `KruskalWallis` exist only in the library. These are the tests to reach for when normality does not hold, and `Docs/stats.md` sends readers to them, so a CLI user who checks an assumption and finds it broken has nowhere to go. The owner asked for them on 2026-09-29.

## What Changes

- `wilcoxon single <var> <mu> [two-sided|greater|less]` and `wilcoxon paired <var1> <var2> [two-sided|greater|less]` run the signed-rank tests and print `W=… p=…`.
- In one-shot mode, `wilcoxon` accepts a negative `mu` without misinterpreting it as a CLI flag (`DisableFlagParsing`).
- `wilcoxon` and `mannwhitney` validate argument count strictly and refuse unread extra arguments with usage errors.
- `mannwhitney <var1> <var2> [two-sided|greater|less]` runs the Mann-Whitney U test and prints `U=… p=…`.
- `kruskal <group1> <group2> [groupN]` runs the Kruskal-Wallis test and prints `H=… df=… p=…`.
- The alternative is read by the parser `ztest` already uses, so a misspelling is refused rather than run as two-sided; leaving it out runs a two-sided test, the library's default. The confidence level is not exposed, as it is not for `ttest` and `ztest`.
- `Docs/cli-dsl.md` (the command index, the command groups and a short section), both CHANGELOGs and `delivery-status.md` are updated. `skills/use-insyra-cli/` teaches no command list and needs no change.

## Capabilities

### New Capabilities

- `cli-rank-tests`: the CLI commands for the Wilcoxon signed-rank, Mann-Whitney U and Kruskal-Wallis tests.

### Modified Capabilities

(none)

## Impact

- Code: `cli/commands/nonparam.go`.
- Tests: `cli/commands/nonparam_test.go` and `cli/commands/port_review_cli_test.go`.
- Docs: `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`.
- No new dependencies. No library change.
