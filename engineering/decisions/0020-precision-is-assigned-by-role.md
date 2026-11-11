# 0020 — `insyra/ml` assigns precision by role, with a fixed-point accumulator

**Status:** accepted
**Date:** 2026-08-01

## Context
`ml` needed one precision contract that every model obeys, and acceleration may move the split between CPU and device from run to run. Each row was sourced from what mainstream libraries do, read at source level.

## Decision
Precision is assigned by the role a value plays, not per model. The table is in `/engineering/architecture.md` under "Precision contract". Bulk feature values are `float32` or quantised integers; labels are integers or bools; regression targets entering a residual, stored model parameters and reported values are `float64`; a comparison key deciding a selection uses the widest precision available, on the host; and a reduction accumulator is fixed-point integers.

## Options rejected
- One floating-point precision for everything. A dispatcher that may move the CPU/device split needs an accumulator that is associative, not merely deterministic: every regrouping of a floating-point sum is a different answer. Fixed point is associative. XGBoost reached the same answer for GPU atomics when it removed its single-precision histogram option in 1.7 as "dangerous to use" and replaced it with `GradientPairInt64`.

## Consequences
- Permuting 600 rows spanning magnitudes 10⁻³ to 10³ must fit a bit-identical tree. Floating-point accumulation cannot pass that test.
- Tree sample weights would push float weights into the integer histogram accumulators. That is an open question against this record, tracked in the `AGENTS.md` Follow-ups. Cross-validation weights did not need it: `Estimator` gained an optional `FitWeighted`, and `CrossValidateWeighted` subsets weights with each fold's indices.
