# Proposal: lp-pure-go-default

## Why

[#257](https://github.com/HazelnutParadise/insyra/issues/257) (LP-1, SEC-9, SEC-19) and [#372](https://github.com/HazelnutParadise/insyra/issues/372) (LP-2).

- **The first `SolveFromFile` or `SolveModel` call installs a C program.** When `glpsol` is not found, `lp` downloads GLPK 5.0 source from ftp.gnu.org (on Windows, whatever SourceForge's `latest/download` redirects to), runs `configure`, `make` and `make install` on the user's machine, and rewrites the process `PATH`. The download has no checksum, lands in a predictable temp path, and has no timeout.
- **The results are not results.** The first table is GLPK's printable report copied line by line into one column, so no variable value can be read by name. That report prints activities with `%13.6g` (GLPK 5.0 `prsol.c`, `prmip.c`), so every value a caller has read from it was rounded to six significant digits: 123456.789 prints as `123457`. Success and failure are strings in a second table, and a timeout kills `glpsol` and returns no solution at all.
- **`lp` cannot be tested without a solver.** Every solve path starts with `initGLPK()`, so the tests only reach code that returns before it.

The owner decided on 2026-09-13:
1. Remove the auto-install. The documentation teaches how to install GLPK.
2. Make [go-milp](https://github.com/daniel-sullivan/go-milp) v0.2.0, a pure-Go MILP solver, the default engine. `require-go-1-26` already moved `0.4` to Go 1.26 for it.
3. Keep GLPK as an optional engine that uses an installed `glpsol`, never one the library installs.
4. Replace `SolveFromFile` and `SolveModel` in 0.4 rather than keep deprecated wrappers: their first table is GLPK's report text, which go-milp cannot produce, so a wrapper would have to require `glpsol` while the docs say the default needs nothing.

## What Changes

- **New API.** `lp.Solve(model *lpgen.LPModel, opts lp.Options) (*lp.Solution, error)` and `lp.SolveFile(path string, opts lp.Options) (*lp.Solution, error)`. `Options` holds `Engine` (zero value: go-milp) and `TimeLimit`. `Solution` holds `Status`, `Objective`, `Values` (variable name to value), `Engine`, `Elapsed`, `Nodes` and `Log`, and `ToDataTable()` gives `Variable` and `Value` columns in model order.
- **Statuses and errors do not overlap.** `Optimal`, `Feasible` (the time limit stopped the search with a solution in hand), `Infeasible`, `Unbounded` and `Stopped` (the time limit came before any solution) are outcomes with a nil error. An error means there is no `Solution`: `ErrInvalidModel`, `ErrEngineUnavailable` or `ErrSolverFailed`, with the cause wrapped. go-milp returns an incumbent together with `ErrTimeLimit`; `lp` turns that into `Feasible` with a nil error so a caller who stops at `if err != nil` keeps the solution.
- **go-milp workarounds.** go-milp v0.2.0 has two defects this line works around. Its presolve rounds coefficients to 1e-6 when it looks for duplicate rows, so presolve is off and a row whose largest coefficient is below 1 is scaled up before it reaches the solver. It reports an integer model unbounded as soon as its relaxation is, so an unbounded integer model is first checked for an impossible integer equality and then for any integer-feasible point, under the caller's time limit or 30 seconds. `TimeLimit` is documented as best effort.
- **A CPLEX LP reader in Go** turns an LP file, or the text `lpgen` writes, into a go-milp model. It accepts what GLPK 5.0's reader accepts for the supported subset and reports anything else as `ErrInvalidModel` naming the line.
- **The GLPK engine** finds `glpsol` through `GLPK_PATH` (a file or a directory) and then `PATH`, runs it once with `--output` and `--write` (and `--tmlim` for a time limit), reads each value at full precision from the `--write` file and its name from the report, and decides the status from the `--write` status letter, using `glpsol`'s fixed messages when that letter is `u`. Measured on GLPK 5.0: an infeasible or unbounded LP reports `UNDEFINED` with presolve on and prints `LP HAS NO PRIMAL FEASIBLE SOLUTION` or `LP HAS UNBOUNDED PRIMAL SOLUTION`; a time limit prints `TIME LIMIT EXCEEDED; SEARCH TERMINATED`, with the incumbent when there is one; every one of these exits 0.
- **`Solve` and `SolveFile` use `lpgen`'s `WriteLP`** to produce the LP text, so a model has one text form. `WriteLP` itself is added by the `lpgen-reports-errors` change.
- **`SolveFromFile` and `SolveModel` stay, deprecated.** They keep their GLPK behaviour, including the automatic install, and are marked `Deprecated` with a note that they are removed in 0.4. Nothing else changes for their callers, so this line is not breaking.
- `Docs/lp.md` documents the new API, the statuses and errors, and engine selection.

Left on 0.4:
- Removing the auto-install, `SolveFromFile`, `SolveModel`, the report-copying parser and the additional-info table. On this line they remain as deprecated functions and the `lp additional-info table` requirement in `deterministic-and-atomic-output` still describes them.
- The general rule that a failing function returns a usable `*insyra.DataTable`. `SolveFromFile` and `SolveModel` still return nil on failure on this line, so that rule is not part of this line's `error-philosophy` spec. `Solution.ToDataTable` never returns nil, and `lp-solve` states that.
- Tutorial `Docs/tutorials/capacity-planning-with-lp-and-lpgen.md` still calls `lp.SolveModel`.
- The CI step that installs GLPK on Ubuntu and macOS. This line's workflows do not install it, so the GLPK end-to-end and both-engine tests skip in CI here.

Known gaps on this line:
- The function that reads `glpsol`'s solution file for `Solve` (`readGLPKResult`) has no test with recorded `glpsol` output. Its tests need `glpsol` installed and skip without it.
- `Docs/lp.md` has no per-system instructions for installing GLPK.

## Capabilities

### New Capabilities

- `lp-solve`: how `lp` solves a model or an LP file, what `Solution` and its statuses mean, which outcomes are errors, how an engine is chosen, and what the LP reader accepts.

### Modified Capabilities

- `test-suite-integrity`: the solver-free requirement now names the go-milp engine and the LP reader as tested without `glpsol`, and the end-to-end tests as skipping when it is missing.

## Impact

- Not breaking on this line. `Solve` and `SolveFile` are additive, and `SolveFromFile` and `SolveModel` are unchanged apart from their deprecation notices.
- `lp` gains `Options`, `Solution`, `Status`, `Engine`, three sentinel errors, `Solve`, `SolveFile` and `Solution.ToDataTable`.
- `go.mod` gains `github.com/daniel-sullivan/go-milp v0.2.0`, which requires Go 1.26.
