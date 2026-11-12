# 0006 — `ml` wraps `stats` and reimplements only decision trees

**Status:** accepted
**Date:** 2026-07-29

## Context
`stats`' numerics are validated against R.

## Decision
`ml` delegates every model it can to the `stats` function that already computes it and wraps the result. Only decision trees are written from scratch: the family `stats` lacks, the one with no `float64` legacy, the one the ONNX corpus is full of, and the one whose split finding is shaped like a selection.

## Options rejected
- Reimplement the models inside `ml`. The R-validated numerics would be duplicated instead of inherited.

## Consequences
`stats`' R validation covered fitting but not prediction until 2026-07-29, so a claim that wrapping inherits validation holds only for what the reference scripts call (see the standing assumptions in `/engineering/architecture.md`). Columns are matched to a model's features by name, because `stats`' own predict checks only the count.
