# Proposal: stats-test-results

## Why

Nine result types in `stats` share one embedded struct, `testResultBase`, which holds `Statistic`, `PValue`, `DF`, `CI` and `EffectSizes` (#241, ST-5 in `api-review.md`). Because the struct is unexported, the fields read fine on each result, but a caller cannot name the shared part. There is no type to write `func report(r ???)` against, no way to keep a t-test, a Mann-Whitney U and a chi-square result in one slice, and no way to adjust their p-values together for multiple comparisons.

The result types also mark "this field does not apply here" three different ways:

| Field | Filled for | When it does not apply |
| --- | --- | --- |
| `TTestResult.Mean` | one- and two-sample t-tests | `nil` (paired test) |
| `ZTestResult.Mean` | every z-test | never happens, so a plain `float64` |
| `DF`, `CI`, `Mean2`, `N2`, `MeanDiff` | some tests | `nil` |
| `WilcoxonTestResult.Z`, `MannWhitneyUResult.Z` | the asymptotic path | `NaN` (exact path) |
| `FTestResult.DF2` | every F-test | `0` (Bartlett's test, whose chi-square statistic has one degrees-of-freedom parameter) |

So `Mean` is read as `*r.Mean` on a t-test and `r.Mean` on a z-test, and code that checks for `nil` to see whether a value exists gets a `NaN` or a `0` from three fields. A `DF2` of 0 is worse than unset: it is a number a caller can pass on to an F distribution.

## What Changes

- `testResultBase` is exported as `TestResult`, embedded under that name in `TTestResult`, `ZTestResult`, `FTestResult`, `ChiSquareTestResult`, `CorrelationResult`, `WilcoxonTestResult`, `MannWhitneyUResult`, `KruskalWallisResult` and `FriedmanTestResult`. Field access such as `r.PValue` does not change, and code outside `stats` could not name the old embedded field, so nothing outside the package breaks.
- New interface `HypothesisTestResult` with one method, `Base() *TestResult`. `TestResult` defines it, so every type above satisfies the interface through embedding.
- **BREAKING** `TTestResult.Mean` becomes a `float64` like `ZTestResult.Mean`. `PairedTTest` fills `Mean` and `Mean2` with the means of `data1` and `data2`, and `N2` with the number of pairs, next to the `MeanDiff` it already reports.
- **BREAKING** A field that only some calls fill is a pointer that is `nil` when it does not apply. `WilcoxonTestResult.Z` and `MannWhitneyUResult.Z` become `*float64`, `nil` on the exact path. `FTestResult.DF2` becomes `*float64`, `nil` for `BartlettTest`.
- Every number the tests report is unchanged.
- `Docs/stats.md`, both CHANGELOGs, `api-review.md` (ST-5) and `delivery-status.md` are updated.

### Why an accessor method

Go interfaces hold methods, not fields, and a type cannot have a method with the same name as a field it promotes, so an interface of `Statistic()` and `PValue()` would need a second set of names for the same values. The pattern used here is the one Kubernetes uses for `ObjectMeta` and `ObjectMetaAccessor`: the shared part is an exported struct, embedded in every type, and a method on that struct returns the struct itself, so embedding alone satisfies the interface. A function over any test takes `stats.HypothesisTestResult` and reads `r.Base().PValue`; a function that only needs the shared fields can also take `*stats.TestResult` and be called with `&res.TestResult`.

## Capabilities

### New Capabilities

- `stats-test-results`: the exported shared result type, the interface every hypothesis test result satisfies, and how a result field says it does not apply.

### Modified Capabilities

(none)

## Impact

- Code: `stats/structs.go`, `stats/ttest.go`, `stats/ztest.go`, `stats/ftest.go`, `stats/chi_square.go`, `stats/correlation.go`, `stats/nonparam_wilcoxon.go`, `stats/nonparam_mwu.go`, `stats/nonparam_kw.go`, `stats/nonparam_friedman.go`.
- Tests: a test that every result type satisfies `HypothesisTestResult`, that `Base` returns the embedded values, and the new paired and nil-field behaviour; existing tests that read `*r.Mean`, `r.Z` or `r.DF2` move to the new types without changing the values they assert.
- Docs: `Docs/stats.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`.
- No new dependencies. Lands after `stats-test-settings`, which changes the same t-test functions.
