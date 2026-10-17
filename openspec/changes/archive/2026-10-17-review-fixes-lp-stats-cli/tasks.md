# Tasks: review-fixes-lp-stats-cli

## 1. Solver (lp)

- [x] 1.1 `lp/milp.go`: presolve is off, and each row whose largest coefficient is below 1 is scaled before the solve
- [x] 1.2 `lp/milp.go`: an unbounded integer model is checked for an integer point under `TimeLimit` or 30 seconds, reports `StatusInfeasible` when none exists and `StatusStopped` when undecided, and an impossible integer equality is refused without a search
- [x] 1.3 `lp/solve.go`: the `TimeLimit` comment describes the limit as best effort
- [x] 1.4 `lp/solve_test.go`: `TestMILPPresolveKeepsDistinctCloseConstraints` and `TestMILPUnboundedRelaxationWithNoIntegerFeasiblePointIsInfeasible`
- [x] 1.5 `Docs/lp.md`: `TimeLimit` is best effort, and the integer point check and its limit are described

## 2. LINGO reader and LP writer (lpgen)

- [x] 2.1 `lpgen/lingo.go`: strict mode splits statements at `;`, reads a statement after a comment on the same line, and checks each objective and relation side with `lingoIsLinear`
- [x] 2.2 `lpgen/lingo_strict_test.go`: `TestParseLingoReadsInlineCommentFollowingStatement` and `TestParseLingoRejectsNonlinearOrUnknownExpressions`
- [x] 2.3 `lpgen/lpgen.go`: `WriteLP` validates the objective type before it writes the banner
- [x] 2.4 `lpgen/write_lp_test.go`: `TestWriteLPDoesNotWriteBeforeValidatingObjectiveType`
- [x] 2.5 `Docs/lpgen.md`: strict mode reads statements separated by `;` and refuses non-linear sides

## 3. Long-format designs (stats)

- [x] 3.1 `stats/long_format.go`: `TwoWayANOVAFromTable` checks every level combination before it allocates the cells, and `readRepeatedMeasures` finds duplicate and missing observations before it allocates the grid
- [x] 3.2 `stats/long_format_test.go`: `TestTwoWayLongRejectsMissingCombinationsBeforeGridAllocation` and `TestRepeatedMeasuresLongRejectsMissingCombinationsBeforeGridAllocation`

## 4. CLI and environments

- [x] 4.1 `internal/dsl/commands/hypothesis.go` and `nonparam.go`: `ttest`, `ztest` and `wilcoxon` disable flag parsing, so a negative `mu` is read as a value
- [x] 4.2 `cli/commands/cobra_test.go`: `TestOneShotSingleSampleTestsAcceptNegativeMu`
- [x] 4.3 `internal/dsl/commands/hypothesis.go`: `anova twoway` takes the list form when its first two arguments after the mode are integers
- [x] 4.4 `internal/dsl/commands/anova_table_test.go`: `TestAnovaTwoWayNumericTableNameStillSelectsListForm`
- [x] 4.5 `internal/dsl/commands/arg_limit.go` and `registry.go`: `ArgLimit.validate` is called by `Register`, which refuses a negative limit
- [x] 4.6 `internal/dsl/commands/arg_limit_test.go`: `TestRegisterRejectsNegativeArgumentLimits`
- [x] 4.7 `internal/dsl/env/manager.go`: `ErrEnvironmentExists`, `createEnvironment` removes its directory when initialisation fails, and `writeDefaultFiles` refuses a default file path that is not a regular file
- [x] 4.8 `internal/dsl/session.go` and `EnsureDefaultEnvironment`: check `ErrEnvironmentExists` with `errors.Is`
- [x] 4.9 `internal/dsl/env/manager_test.go`: `TestWriteDefaultFilesRejectsDirectoryAtDefaultFilePath`, `TestCreateIdentifiesAlreadyExistingEnvironment` and `TestCreateRemovesItsDirectoryWhenInitializationFails`
- [x] 4.10 `Docs/cli-dsl.md`: `ArgLimit` does not check required arguments, the custom-command example checks its arguments, and the `anova twoway` form rule is described

## 5. Changelog

- [x] 5.1 `CHANGELOG.md` and `CHANGELOG_TW.md`: Unreleased entries for the solver, the LINGO reader and LP writer, the long-format refusal, the CLI single-sample tests, the `anova twoway` form rule, the environment creation fix and `Register`
