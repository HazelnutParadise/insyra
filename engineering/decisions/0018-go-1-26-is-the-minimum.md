# 0018 — Go 1.26 is the minimum

**Status:** accepted
**Date:** 2026-09-13

## Context
`lp` was to stop installing GLPK and default to a solver written in Go (record 0024). The solver it depends on, go-milp, declares `go 1.26` in its `go.mod` (#257, #372).

## Decision
The `go` directive on `0.4` moved from 1.25.12 to 1.26.8, the newest 1.26 release on go.dev that day (`require-go-1-26`). The owner decided it on 2026-09-13. `dev` moved to Go 1.26 separately, on 2026-10-10, for standard-library advisories that had no fix on 1.25.

## Options rejected
- Stay on 1.25 and keep GLPK as `lp`'s default. That keeps the installation step the pure-Go solver exists to remove.

## Consequences
The minimum Go version is part of the promise to downstream users. Raising it again is an explicit decision, not a side effect of a dependency refresh (`AGENTS.md`, Dependency Refresh Before a Release).
