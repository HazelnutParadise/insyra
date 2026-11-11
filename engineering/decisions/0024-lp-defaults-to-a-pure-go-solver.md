# 0024 — `lp` defaults to a solver written in Go

**Status:** accepted
**Date:** 2026-09-13

## Context
`lp` installed GLPK to solve a model. `SolveModel` and `SolveFromFile` returned two tables, which could not tell "the model has no solution" from "the solve failed" (#372).

## Decision
`lp` defaults to a solver written in Go and keeps GLPK as an engine the caller installs and selects. The owner ruled both on 2026-09-13. On `0.4`, `SolveModel` and `SolveFromFile` are replaced with no deprecated wrappers. The LP reader copies GLPK's behaviour, including prefix keywords and a Binary variable keeping its explicit bounds, so one file is one model to both engines; it departs only where GLPK silently reads a malformed bound as a different model (`lp-pure-go-default`).

## Options rejected
- Deprecated wrappers for `SolveModel` and `SolveFromFile`. A wrapper would keep the defect #372 is about.

## Consequences
The pure-Go solver's dependency, go-milp, needs Go 1.26 (record 0018).
