# Proposal: stats-test-settings

## Why

The hypothesis tests in `stats` take the same two settings, the alternative hypothesis and the confidence level, four different ways (#240, ST-4 in `api-review.md`):

| Functions | Alternative | Confidence level |
| --- | --- | --- |
| `SingleSampleTTest`, `TwoSampleTTest`, `PairedTTest` | none, always two-sided | optional trailing `...float64` |
| `SingleSampleZTest`, `TwoSampleZTest` | required positional | required positional `float64` |
| `SingleSampleWilcoxon`, `PairedWilcoxon`, `MannWhitneyU` | required positional | optional trailing `...float64` |

R's `t.test` and SciPy's `ttest_1samp`, `ttest_ind` and `ttest_rel` all take an alternative, so a one-sided t-test is ordinary analysis that insyra cannot do today. The positional forms also let a bad value through without a word. Measured on `0.4` at 43cb6519 on 2026-09-28 with seven observations:

- `SingleSampleTTest(x, 50, math.NaN())` returns the 95% interval `[46.747, 58.939]` with a nil error.
- `SingleSampleWilcoxon(x, 50, stats.TwoSided, math.NaN())` returns the 95% interval `[46.8, 59.3]` with a nil error.
- `SingleSampleZTest(x, 50, 10, stats.TwoSided, math.NaN())` returns the interval `[NaN, NaN]` with a nil error.
- `MannWhitneyU(x, y, "")` fails with `invalid alternative hypothesis`, although two-sided is the default everywhere else.

The groups of the k-sample tests are passed two ways. `LeveneTest` and `BartlettTest` take a `[]IDataList`, while `OneWayANOVA`, `TwoWayANOVA`, `RepeatedMeasuresANOVA`, `KruskalWallis` and `FriedmanTest` take `...IDataList`. And `KMeans(dt, k, opts ...KMeansOptions)` can be called without options while `FactorAnalysis(dt, opt FactorAnalysisOptions)` cannot, although the zero `FactorAnalysisOptions` already means every default: `FactorAnalysis(dt, FactorAnalysisOptions{})` and `FactorAnalysis(dt, DefaultFactorAnalysisOptions())` return the same loadings, measured on the same commit.

`AGENTS.md` ("How a function takes settings") already says how this should look: a statistical setting goes in an options struct taken as an optional trailing `opts ...XxxOptions`, a setting with no safe default stays a required parameter, and more than one options value is an error.

## What Changes

- **BREAKING** The t-tests take `opts ...TTestOptions` in place of `confidenceLevel ...float64`. `TTestOptions` has `Alternative` and `ConfidenceLevel`. The t-tests gain one-sided tests: `Greater` and `Less` give R's `t.test` p-value and one-sided confidence bound, with the open end at `+Inf` or `-Inf` as the z-tests already report it.
- **BREAKING** The z-tests take `opts ...ZTestOptions` in place of the positional `alternative` and `confidenceLevel`. `mu` and `sigma` stay required.
- **BREAKING** `SingleSampleWilcoxon` and `PairedWilcoxon` take `opts ...WilcoxonOptions`, and `MannWhitneyU` takes `opts ...MannWhitneyUOptions`, in place of the positional `alt` and trailing `confidenceLevel`.
- In every one of these options structs, an empty `Alternative` means `TwoSided` and a zero `ConfidenceLevel` means 0.95. Any other alternative, and any other level outside (0, 1), `NaN` included, is an error. A second options value is an error.
- **BREAKING** `OneWayANOVA`, `TwoWayANOVA`, `RepeatedMeasuresANOVA`, `KruskalWallis` and `FriedmanTest` take their groups, cells or subjects as one `[]insyra.IDataList`, as `LeveneTest` and `BartlettTest` already do. `TwoWayANOVA(aLevels, bLevels, cells)` keeps its two level counts in front.
- `FactorAnalysis` takes `opts ...FactorAnalysisOptions`, so `FactorAnalysis(dt)` runs with the defaults. Existing calls that pass one options value compile unchanged. `DefaultFactorAnalysisOptions` stays.
- Results for every call that the old signature could express are unchanged, bit for bit.
- The CLI's `ttest`, `ztest`, `anova`, `ftest` commands call the new signatures and print what they printed before.
- `Docs/stats.md`, the two tutorials that call these functions, `skills/insyra/SKILL.md` if a principle in it changes, both CHANGELOGs, `api-review.md` (ST-4) and `delivery-status.md` are updated.

### Why a slice for the groups

A Go function has one variadic parameter and it must come last. `AGENTS.md` puts a function's options in a trailing `opts ...XxxOptions`, so a function whose groups are variadic can never take options without breaking every caller again or growing a second name. That is how `stats` already ended up with `GLM(opts, y, xs...)`, options first and required, and with `LogisticRegression` beside `LogisticRegressionWithOptions`. The k-sample tests have settings that R and SciPy expose and that insyra does not have yet: the centre of Levene's test (`center` in `car::leveneTest` and `scipy.stats.levene`), and Welch's one-way ANOVA (`var.equal = FALSE`, the default of R's `oneway.test`). With a slice, each of them can arrive as an options struct without a further break. R's own k-sample functions take a list the same way (`kruskal.test(list(a, b, c))`, `bartlett.test(list(a, b, c))`).

The cost is that `OneWayANOVA(a, b, c)` becomes `OneWayANOVA([]insyra.IDataList{a, b, c})`. Code that already holds its groups in a slice changes from `OneWayANOVA(groups...)` to `OneWayANOVA(groups)`.

### What stays required

`mu`, `sigma`, `sigma1`, `sigma2` and `equalVariance` stay positional. The first four state the hypothesis or a known population value, so they have no safe default. For `equalVariance` the two reference implementations disagree on a default: R's `t.test` uses Welch's test and SciPy's `ttest_ind` uses Student's. Giving it one would be a user-visible default this change does not choose. The question goes to #240 with a recommendation.

## Capabilities

### New Capabilities

- `stats-test-settings`: how the hypothesis tests take their alternative and confidence level, what the zero values mean, which values are refused, how the k-sample tests take their groups, and that `FactorAnalysis` can be called without options.

### Modified Capabilities

- `stats-test-input`: its scenarios call `KruskalWallis`, `OneWayANOVA`, `TwoWayANOVA` and `PairedTTest` with the old signatures. They are rewritten with the new ones; what they require does not change.

## Impact

- Code: `stats/ttest.go`, `stats/ztest.go`, `stats/nonparam_wilcoxon.go`, `stats/nonparam_mwu.go`, `stats/anova.go`, `stats/nonparam_kw.go`, `stats/nonparam_friedman.go`, `stats/factor_analysis.go`, `stats/distutil.go`, `stats/mathutil.go`, a new `stats/test_options.go`; `cli/commands/hypothesis.go`.
- Tests: new tests for the options, the refusals and the one-sided t-tests; a cross-language test of the t-tests' alternatives against R's `t.test` and SciPy; every existing call site in `stats` tests moved to the new signatures, with the values they assert unchanged.
- Docs: `Docs/stats.md`, `Docs/tutorials/nonparametric-tests-when-normality-fails.md`, `Docs/tutorials/ab-test-decision-with-statistics.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`.
- No new dependencies.
