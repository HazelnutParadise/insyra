# Proposal: stats-cronbach-alpha

## Why

A Likert questionnaire is checked for internal consistency before its items are summed or factor-analysed, and `stats` had no reliability coefficient, so that first step meant leaving Insyra for R or SPSS (#734). `design.md` has the design, reviewed independently and confirmed by the owner on 2026-11-27.

## What Changes

- `stats.CronbachAlpha(dt, items, opts...)` with `CronbachAlphaOptions{ReverseScored}`, returning `CronbachAlphaResult` and one `CronbachAlphaItem` per item.
- A `cronbach_alpha` method in `stats/testdata/crosslang_baseline.R` against `psych::alpha`.
- The same API on `0.4` and `dev`.

## Capabilities

### New Capabilities

- `stats-reliability`: the reliability of a scale of items.

## Impact

- Code: `stats/cronbach_alpha.go`.
- Tests: `stats/cronbach_alpha_test.go`, `stats/testdata/crosslang_baseline.R`.
- Docs: `Docs/stats.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
