# 0007 — Numeric input that cannot be read is refused, not turned into zero

**Status:** accepted
**Date:** 2026-08-01

## Context
Regression and correlation read every value through `DataList.ToF64Slice`, which yields `0` for anything it cannot parse and returns a full-length slice regardless. One blank among six observations moved a Pearson coefficient from 0.9992 to 0.9879, with no error.

## Decision
Numeric input in `stats` goes through one validating helper. Each analysis family states which of three treatments it applies to a value it cannot read as a finite number: refusal (regression, correlation, clustering, PCA, KNN), listwise deletion (factor analysis), or a learned per-node direction for a missing feature with refusal of a missing target (decision trees). Substituting a zero is not one of them.

## Options rejected
- Keep `ToF64Slice` as the read path. It has no failure channel, so every caller believed it held the caller's data.

## Consequences
- Breaking for anyone whose data contains blanks: what used to return a number now returns an error. The number it used to return was wrong.
- `ToF64Slice` still backs display and reporting paths in `plot`, `gplot` and the CLI, where a zero shows on a chart instead of entering a coefficient. A new numeric analysis does not use it. The remaining callers are tracked in the `AGENTS.md` Follow-ups.
