# Proposal: quant-gap-classes

## Why

[#471](https://github.com/HazelnutParadise/insyra/issues/471): the study of the four classic gaps needed them defined so they can be computed before the outcome is known, with fill tracking.

## What Changes

- `quant.ClassifyGaps(high, low, close, volume, opts)` finds every gap up and down and classes it as common, breakaway, runaway or exhaustion from the 20-day range, the earlier move in standard deviations and the relative volume. Anything else is `GapUnclassified`.
- Each gap reports whether it was filled within 5 and within 20 rows, marked as an outcome to study and not a signal.
- `GapKind`, `GapFill` with `String()`.

## Capabilities

### New Capabilities

- `quant-gap-classes`: classifying price gaps as common, breakaway, runaway or exhaustion from information known on the gap day, with fill tracking as a separate outcome.

## Impact

`quant/`, tests, `Docs/quant.md`, both CHANGELOGs.
