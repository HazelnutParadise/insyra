# Proposal: stats-cronbach-alpha-ci

## Why

The first `CronbachAlpha` left out a confidence interval for alpha and Guttman's lambda 6, which the owner asked for on 2026-11-28 after #734 closed. Both are what R's psych reports (`alpha.ci`, `G6(smc)`).

## What Changes

- `CronbachAlphaResult.CI` and `ConfidenceLevel`: Feldt's interval (Feldt, Woodruff and Salih, 1987), at `CronbachAlphaOptions.ConfidenceLevel`, 0 meaning 0.95, a level outside (0, 1) refused.
- `CronbachAlphaResult.GuttmanLambda6`: from the correlation matrix and the items' squared multiple correlations, as psych's `G6(smc)`.
- The same on `0.4` and `dev`. Both are added fields, so no caller changes.

## Capabilities

### Modified Capabilities

- `stats-reliability`: the interval and lambda 6.

## Impact

- Code: `stats/cronbach_alpha.go`. Tests: `stats/cronbach_alpha_test.go`, `stats/testdata/crosslang_baseline.R`. Docs: `Docs/stats.md`, both changelogs.
