# Design: stats-test-results

## Context

`stats/structs.go` defines `testResultBase` and `EffectSizeEntry`. Nine result types embed `testResultBase`; their constructors spell it out in composite literals (`testResultBase: testResultBase{...}`), 23 of them across `ttest.go`, `ztest.go`, `ftest.go`, `chi_square.go`, `nonparam_wilcoxon.go`, `nonparam_mwu.go`, `nonparam_kw.go` and `nonparam_friedman.go`. `CorrelationResult` embeds it and has no field of its own.

`stats-test-settings` changes the t-test functions first; this change lands on top of it.

## Goals / Non-Goals

**Goals:**

- A caller can write one function, or keep one slice, over every hypothesis test result.
- One way to say that a result field does not apply: a nil pointer.
- No reported number changes.

**Non-Goals:**

- The ANOVA result types, which do not embed the shared part and report a table: the residual row's `F`, `P` and `EtaSquared` are `NaN`, as R leaves them blank. They are a different shape and stay as they are.
- `String()` or `Show()` on every result type (ST-9, #245).
- Replacing `BartlettTest`'s `FTestResult` with a chi-square result type. `DF2` being nil says what is needed without moving the function to another type.

## Decisions

### `TestResult` plus a one-method interface

Exporting the struct alone lets a function take `*stats.TestResult` and be called with `&res.TestResult`, but it still gives no type for a slice of mixed results or a type switch. An interface alone would need methods, and the fields' own names (`Statistic`, `PValue`, …) cannot also be method names on types that promote those fields. So both: the exported struct keeps the fields where they are, and `Base() *TestResult`, defined once on `*TestResult`, puts every result type in the interface by embedding. The name follows what the issue calls the shared part, 基底, and what the struct used to be called, `testResultBase`.

`Base` has a pointer receiver and returns the embedded value itself. A copy would have been as easy, but `DF` and `CI` are pointers anyway, so a copy would share them and look independent while it is not.

### Which fields are pointers

The rule is the one `DF`, `CI`, `Mean2`, `N2` and `MeanDiff` already follow: a field that only some calls fill is a pointer, nil when it does not apply; a field every call fills is a plain value.

- `TTestResult.Mean` is nil only for the paired test, because that test never computed the two means. Filling them there is cheaper for callers than a pointer: reports of a paired comparison give the mean of each condition next to the mean difference. With `Mean` always filled it becomes a `float64`, the same type as `ZTestResult.Mean`. `Mean2` and `N2` describe the second sample, which a paired test has, so they are filled there too.
- `Z` on the rank tests is not computed on the exact path. It was reported as `NaN`, which is a number a caller can carry into arithmetic without noticing. It becomes a pointer.
- `FTestResult.DF2` was 0 for `BartlettTest`, whose statistic follows a chi-square distribution with a single degrees-of-freedom parameter, k − 1, reported in `DF`. 0 is a number a caller can pass to an F distribution. It becomes a pointer, nil for Bartlett.

The paired means are computed from the same finite values the test reads, with `meanOfF64`, the helper the one- and two-sample tests already use.

## Risks / Trade-offs

- `*r.Mean` on a t-test result, and arithmetic or comparisons on `r.Z` and `r.DF2`, stop compiling in callers' code, so the compiler finds those. Passing `r.Z` or `r.DF2` to `fmt` still compiles and now prints a pointer; the changelog entry says so, because the compiler cannot.
- `ftest.go` is also being edited on `dev` for another fix. The change here touches only the struct field and the `DF2:` line of each constructor, so a later merge of `dev` has small, separate hunks to reconcile.
