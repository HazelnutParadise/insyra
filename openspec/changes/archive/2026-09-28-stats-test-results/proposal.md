# Proposal: stats-test-results

## Why

Nine result types in `stats` share one embedded struct, `testResultBase`. Because it is unexported, a caller cannot name the shared part: there is no type to write `func report(r ???)` against, no way to keep a t-test, a Mann-Whitney U and a chi-square result in one slice, and no way to adjust their p-values together. Ported from `0.4` (#241); the part that changes field types (`TTestResult.Mean`, `WilcoxonTestResult.Z`, `MannWhitneyUResult.Z`, `FTestResult.DF2`) is breaking and stays on `0.4`.

## What Changes

- `testResultBase` is exported as `TestResult`. Field access such as `r.PValue` does not change, and code outside `stats` could not name the old embedded field, so nothing outside the package breaks.
- New interface `HypothesisTestResult` with `Base() *TestResult`, satisfied by every result type that embeds `TestResult`.
- Every number the tests report is unchanged.

## Capabilities

### New Capabilities

- `stats-test-results`: the exported shared result type and the interface every hypothesis test result satisfies.

### Modified Capabilities

None.

## Impact

- `stats/structs.go` and the files that embed the type, `Docs/stats.md`, both changelogs.
