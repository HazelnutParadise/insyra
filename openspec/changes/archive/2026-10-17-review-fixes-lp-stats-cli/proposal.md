# Proposal: review-fixes-lp-stats-cli

## Why

A review of the 0.4 to 0.3.5 port (commit 8029c140) found that the same code had been copied to both lines, defects included. Each item below is pinned by a test added in that commit.

- **go-milp solver** (`lp/milp.go`). go-milp v0.2.0 merged distinct rows with small coefficients and rejected rows below its absolute tolerances, so a feasible model came back infeasible (`TestMILPPresolveKeepsDistinctCloseConstraints`). It also reported an integer model as unbounded whenever its relaxation was, even when the model had no integer point (`TestMILPUnboundedRelaxationWithNoIntegerFeasiblePointIsInfeasible`).
- **ParseLingo strict mode** (`lpgen/lingo.go`). It read a line as one statement, so a statement after a comment on the same line was dropped (`TestParseLingoReadsInlineCommentFollowingStatement`). It also accepted an empty objective and an unknown function (`TestParseLingoRejectsNonlinearOrUnknownExpressions`).
- **WriteLP** (`lpgen/lpgen.go`) wrote its banner before it refused an unknown objective type (`TestWriteLPDoesNotWriteBeforeValidatingObjectiveType`).
- **Long-format ANOVA and Friedman** (`stats/long_format.go`) allocated one cell per level combination before finding a missing or repeated one (`TestTwoWayLongRejectsMissingCombinationsBeforeGridAllocation`, `TestRepeatedMeasuresLongRejectsMissingCombinationsBeforeGridAllocation`).
- **CLI** (`internal/dsl/commands/`). `anova twoway` gave a DataTable the table form even when the arguments were level counts and the table's name was a number (`TestAnovaTwoWayNumericTableNameStillSelectsListForm`). `wilcoxon`, `ttest` and `ztest single` read a negative `mu` in one-shot mode as a flag (`TestOneShotSingleSampleTestsAcceptNegativeMu`). `Register` accepted a negative argument limit, and the command then panicked with an index out of range on every call, at `core[limit]` in `internal/dsl/commands/arg_limit.go` (`TestRegisterRejectsNegativeArgumentLimits`).
- **Environments** (`internal/dsl/env/manager.go`). A failed write of the default files left the new directory behind, and the next call treated the environment as created (`TestCreateRemovesItsDirectoryWhenInitializationFails`, `TestCreateIdentifiesAlreadyExistingEnvironment`). A directory where a default file belongs was skipped, not reported (`TestWriteDefaultFilesRejectsDirectoryAtDefaultFilePath`).

## What Changes

No breaking changes. The stricter refusals below are behaviour changes, though: input that was silently dropped or accepted now returns an error, and some results change.

- **`lp`, go-milp engine.** Presolve is off, and a row whose largest coefficient is below 1 is scaled so that its largest coefficient is 1 before the solve. An integer model whose relaxation is unbounded is first checked for an integer point, under `TimeLimit` or 30 seconds when none is set. It is then reported `StatusInfeasible` when no point exists and `StatusStopped` when the check cannot decide. An equality over integer variables whose right side is not a multiple of the gcd of its coefficients is reported infeasible without a search. `Options.TimeLimit` is documented as best effort, since, per the `lp/solve.go` comment, the timer starts after the first relaxation. A model that was reported infeasible or unbounded may now be solved, and an unbounded integer model with no integer point is now infeasible.
- **`lpgen`, `ParseLingo` and `ParseLingoFile`.** Strict mode splits statements at `;`, so a statement after a comment on the same line is read. An objective or relation side that is not a linear expression, an empty objective, and `**` are refused with the line number. The non-strict readers, `ParseLingoModel_str` and `ParseLingoModel_txt`, keep their statement handling, but the shared expression cleanup now also trims spaces at both ends for them. No test in the commit covers that trim.
- **`lpgen`, `WriteLP`.** An unknown objective type is refused before any byte reaches the writer.
- **`stats`, two-way, repeated-measures and Friedman long-format functions.** A missing or repeated level combination is found before the grid is allocated. Results and error messages do not change for any input that was accepted.
- **CLI, `anova twoway`.** The list form is chosen when the first two arguments after the mode are both integers, even when a DataTable has that name. Before, a DataTable in the first position always selected the table form.
- **CLI, `wilcoxon single`, `ttest single`, `ztest single`.** These commands disable flag parsing, so a negative `mu` is read as a value in one-shot mode. Behaviour change: an argument beginning with `--` now reaches these commands as a value and is no longer parsed as a flag.
- **CLI registry.** `Register` refuses a negative maximum, a negative form position, and a negative count for any form, with an error, and leaves the command unregistered.
- **Environments.** `Create` removes the new directory and returns the error when the default files cannot be written. `Create` on an existing environment returns `ErrEnvironmentExists`, which `NewSession` and `EnsureDefaultEnvironment` now check with `errors.Is`. A default file path that holds something other than a regular file is an error.
- **Docs.** `Docs/lp.md` (the unbounded check and `TimeLimit`), `Docs/lpgen.md` (strict-mode statements and linear sides), `Docs/cli-dsl.md` (`ArgLimit` does not check required arguments, the custom-command example checks its arguments, and the `anova twoway` form rule).
- **Changelog.** Unreleased entries in `CHANGELOG.md` and `CHANGELOG_TW.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `lp-solve`: the unbounded outcome, the go-milp scaling and presolve setting, and the best-effort `TimeLimit`.
- `lpgen-model-files`: statements after a comment, linear-expression checks in strict mode, and `WriteLP` validating before writing.
- `stats-long-format`: missing and repeated combinations are refused before the grid is allocated.
- `cli-negative-literals`: the three single-sample tests accept a negative `mu`.
- `cli-long-format-tests`: the `anova twoway` form rule for a DataTable whose name is a number.
- `command-registry`: `Register` refuses a negative argument limit.
- `env-management`: a failed creation leaves no directory, a non-regular default file is an error, and `Create` reports `ErrEnvironmentExists`.

## Impact

- Code: `lp/milp.go`, `lp/solve.go` (comment only), `lpgen/lingo.go`, `lpgen/lpgen.go`, `stats/long_format.go`, `internal/dsl/commands/hypothesis.go`, `internal/dsl/commands/nonparam.go`, `internal/dsl/commands/registry.go`, `internal/dsl/commands/arg_limit.go`, `internal/dsl/env/manager.go`, `internal/dsl/session.go`.
- Tests: `lp/solve_test.go`, `lpgen/lingo_strict_test.go`, `lpgen/write_lp_test.go`, `stats/long_format_test.go`, `cli/commands/cobra_test.go`, `internal/dsl/commands/anova_table_test.go`, `internal/dsl/commands/arg_limit_test.go`, `internal/dsl/env/manager_test.go`.
- Users: a model with small coefficients that was reported infeasible is now solved. An unbounded integer model with no integer point reports `StatusInfeasible`. Scripts that pass `--`-prefixed arguments to the three single-sample tests see them as values.
