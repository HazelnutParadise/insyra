# Proposal: lp-deprecated-solvers-never-nil

## Why

`SolveFromFile` and `SolveModel` returned `nil` as the solution table on a timeout, a solver error, a temporary file that could not be written or rejected arguments, and `nil` for the info table too on the last two. `Docs/lp.md`'s own example called `result.Show()` straight away, and a nil `*DataTable` panics there (measured 2026-09-12).

## What Changes

- The solution table of a failed solve is an empty `DataTable` carrying the reason on `Err()`, never `nil`; `parseGLPKOutputFromFile` does the same for an unreadable file.
- A call rejected before the solver runs returns an info table with Status `Error` and the reason in Warnings.
- BREAKING for callers that tested `result == nil`; they test `result.Err() != nil`. The functions are Deprecated and `Solve`/`SolveFile` already return an error.

## Capabilities

### Modified Capabilities

- `lp-solve`: the deprecated GLPK functions never return a nil table.

## Impact

`lp/lp.go`, `lp/parse_test.go`, `lp/solve_args_test.go`, `Docs/lp.md`, both CHANGELOGs, `AGENTS.md` follow-up removed.
