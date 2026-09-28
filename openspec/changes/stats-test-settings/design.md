# Design: stats-test-settings

## Context

`AGENTS.md` ("How a function takes settings") sorts every setting into a statistical or a programming setting. Statistical settings go in an options struct taken as an optional trailing `opts ...XxxOptions`. A setting with no safe default is a required parameter, because an options field always has a default. More than one value for a trailing optional parameter is an error.

The alternative hypothesis and the confidence level are statistical settings: both are written into the methods section of a report. Both have a default every reference implementation shares, two-sided and 0.95, which the t-tests and the rank tests already apply. So both belong in an options struct.

The t-test statistics are computed today by `tTwoTailedPValue`, `tMarginOfError` and `symmetricCI`. The z-tests already support one-sided tests through `zPValue`, `zMarginOfErrorOneSided` and `ciByAlternative`, and their R and SciPy baselines already compare the open end of a one-sided interval.

## Goals / Non-Goals

**Goals:**

- One way to pass the alternative and the confidence level to every hypothesis test that has them.
- A one-sided t-test whose statistic, p-value, degrees of freedom and confidence bound match R's `t.test` and SciPy's `ttest_1samp`, `ttest_ind` and `ttest_rel`.
- A value the old signatures let through silently (`NaN` as a level, an empty alternative that meant an error in one family and was impossible in another) has one meaning everywhere.
- Every result the old signatures could express is unchanged bit for bit.

**Non-Goals:**

- A default for `equalVariance`. See "What stays required".
- New settings for the k-sample tests (Levene's centre, Welch's ANOVA). The slice only makes room for them.
- The regression functions: `GLM` takes its options first and required, and `LogisticRegression` and `PoissonRegression` each have a `...WithOptions` twin. They are recorded as an `AGENTS.md` follow-up.
- The confidence level of the GLM, logistic and Poisson options, which falls back to 0.95 for a value outside (0, 1) where the tests now refuse it. Recorded in the same follow-up.
- The CLI gaining an alternative for `ttest`. Its commands only move to the new signatures.

## Decisions

### One options type per test family

`TTestOptions` serves the three t-tests, `ZTestOptions` the two z-tests, `WilcoxonOptions` the two signed-rank tests and `MannWhitneyUOptions` the rank-sum test. Today each holds `Alternative` and `ConfidenceLevel`.

One shared type was considered. It would have to carry every setting any of the tests later gains: R's `wilcox.test` has `exact`, `correct` and a zero method, and `t.test` has `var.equal`. On a shared type those fields would be settable, and silently meaningless, on tests they do not belong to. One type per family matches `KNNOptions`, which the three KNN functions share for the same reason.

### What the zero values mean, and what is refused

A shared helper reads the two fields for every family:

- `Alternative`: empty means `TwoSided`. `TwoSided`, `Greater` and `Less` are accepted. Anything else is an error: `alternative must be two-sided, greater or less, got "sideways"`.
- `ConfidenceLevel`: 0 means 0.95. A value strictly between 0 and 1 is used. Anything else, `NaN`, negative values, 1 and above, is an error: `confidence level must be strictly between 0 and 1, got NaN`.

Refusing instead of falling back keeps what the t-, z- and Wilcoxon tests already did for an out-of-range level passed by position; the only value whose treatment changes is `NaN`, which the old checks let through. A zero level had to become the default, because an options field is always present and its zero value is what an unset field reads as.

A second options value returns `at most one stats.TTestOptions may be given, got 2`, following the root package's wording for the same rule, from one generic helper.

The validation runs before any sample is read, as the z-tests' checks do now.

### One-sided t-tests

With `t` the statistic, `df` its degrees of freedom, `se` the standard error, `est` the estimate (the mean, the difference of means, or the mean difference) and `cl` the level:

| Alternative | p-value | confidence interval |
| --- | --- | --- |
| `TwoSided` | `2 · (1 − F(|t|))`, as today | `est ± q(1 − (1 − cl)/2) · se`, as today |
| `Greater` | `F(−t)` | `[est − q(cl) · se, +Inf]` |
| `Less` | `F(t)` | `[−Inf, est + q(cl) · se]` |

`F` is the Student-t CDF and `q` its quantile. The greater-side p-value uses `F(−t)` rather than `1 − F(t)`: the two are equal by symmetry, but the subtraction loses digits when `t` is large, and R computes `pt(t, lower.tail = FALSE)` directly. The two-sided branch keeps today's expression so existing results do not move.

These go into `distutil.go` as `tPValue(t, df, alt)` and into `mathutil.go` as `tMarginOfErrorOneSided(cl, df, se)`, next to their z counterparts, and the interval is built with the existing `ciByAlternative`. `stats/AGENTS.md` asks for exactly that layering.

A sample with no spread gives `t = ±Inf`. The single-sample test hard-codes p = 0 for that case today, which is right only for the two-sided test. It now goes through `tPValue` like every other case, and since `F(−Inf) = 0` and `F(+Inf) = 1`, `+Inf` against `Less` gets p = 1. `t = NaN` gives `NaN`, as now.

### Groups as a slice

The proposal gives the reason. The seven functions that take several lists all take `[]insyra.IDataList`; the error messages keep numbering groups, cells and subjects from 1 by their position in the slice, which is the order the variadic parameters had. An empty or nil slice is refused by the existing count checks (`at least two groups are required` and so on).

### `FactorAnalysis` options become optional

`normalizeFactorAnalysisOptions` already fills every zero field with the default `DefaultFactorAnalysisOptions` returns, so the zero value was already "all defaults" and the only change is the parameter's form. A call passing one value compiles unchanged, because Go accepts one argument for a variadic parameter. What breaks is code that stored `stats.FactorAnalysis` in a variable of the old function type, which the changelog mentions.

### What stays required

`mu`, `sigma`, `sigma1` and `sigma2` state the null hypothesis or a known population value; no default is right for everyone. `equalVariance` could have a safe default in principle, because Welch's test stays valid when the variances are equal. The two reference implementations the parity tests use disagree on it, though: R's `t.test` defaults to Welch's test and SciPy's `ttest_ind` to Student's, and the CLI's `ttest two` defaults to equal variances. Choosing one is a user-visible default, so it stays a required bool and the choice goes to #240 with a recommendation: an `EqualVariance` field in `TTestOptions` whose zero value is Welch's test, as in R.

## Risks / Trade-offs

- Every caller of the changed signatures must be edited. The compiler finds all of them; none silently changes meaning, because no parameter kept its position with a different meaning.
- The cross-language test adds R and SciPy calls to the baseline scripts. Editing either script invalidates its whole baseline cache, so the first local run after this change regenerates every cached baseline.
- SciPy's `confidence_interval` on t-test results needs SciPy 1.11 or later for `ttest_ind`. The reference workflow installs the latest release; the local toolchain has 1.13.1.
