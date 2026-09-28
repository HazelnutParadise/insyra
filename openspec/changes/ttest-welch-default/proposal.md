# Proposal: ttest-welch-default

## Why

`stats-test-settings` moved the t-tests' alternative and confidence level into `TTestOptions`, but left `TwoSampleTTest`'s `equalVariance` a required positional bool, so it is the one t-test setting that still cannot be left out (#240). It could not simply move into the options struct, because an options field needs a default and the references disagree on it: R's `t.test` defaults to Welch's test (`var.equal = FALSE`), SciPy's `ttest_ind` to Student's (`equal_var=True`), and the CLI's `ttest two` to equal variances. The owner ruled on 2026-09-28: the setting goes into `TTestOptions`, leaving it out means Welch's test, and the CLI follows.

Welch's test is the safer thing to get by leaving the field out. When the two variances are equal it gives almost the same answer as Student's test, and when they differ, especially with unequal group sizes, Student's p-value can be misleading while Welch's is not.

## What Changes

- **BREAKING** `TwoSampleTTest(data1, data2 insyra.IDataList, opts ...TTestOptions)`: the positional `equalVariance bool` is removed. `TTestOptions` gains `EqualVariance bool`; its zero value runs Welch's test, and `true` runs Student's pooled-variance test. `TwoSampleTTest(a, b, true)` no longer compiles and becomes `TwoSampleTTest(a, b, stats.TTestOptions{EqualVariance: true})`; `TwoSampleTTest(a, b, false)` becomes `TwoSampleTTest(a, b)`. No call keeps compiling with a different meaning. `SingleSampleTTest` and `PairedTTest` ignore the field, which has no meaning for them.
- **BREAKING** The CLI's `ttest two <var1> <var2>` without a variance token runs Welch's test instead of Student's. `equal` and `unequal` keep their meaning. The command's Forms line says which is the default.
- Every result the explicit old calls produce is unchanged.
- `Docs/stats.md`, `Docs/tutorials/ab-test-decision-with-statistics.md`, both CHANGELOGs (`stats` and `CLI` sections), `api-review.md` (ST-4), `delivery-status.md`; the `AGENTS.md` follow-up that recorded the open question is removed.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `stats-test-settings`: `TTestOptions` holds `EqualVariance`, `equalVariance` is no longer a positional parameter, and a new requirement fixes Welch's test as the default for the library and the CLI.
- `stats-test-input`: the scenario of "Two-sample tests read both samples at one moment" calls `TwoSampleTTest` with the new form.

## Impact

- Code: `stats/ttest.go`, `cli/commands/hypothesis.go`.
- Tests: new tests for the default and for `EqualVariance: true` against R's `t.test`; the CLI default; existing call sites moved to the new form with their asserted values unchanged.
- No new dependencies.
