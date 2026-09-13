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

- **New API.** `lp.Solve(model *lpgen.LPModel, opts lp.Options) (*lp.Solution, error)` and `lp.SolveFile(path string, opts lp.Options) (*lp.Solution, error)`. `Options` holds `Engine` (zero value: go-milp) and `TimeLimit`. `Solution` holds `Status`, `Objective`, `Values` (variable name to value), `Engine`, `Elapsed`, `Nodes` and `Log` (GLPK's output; empty for go-milp), and `ToDataTable()` gives `Variable` and `Value` columns in model order.
- **Statuses and errors do not overlap.** `Optimal`, `Feasible` (the time limit stopped the search with a solution in hand), `Infeasible`, `Unbounded` and `Stopped` (the time limit came before any solution) are outcomes with a nil error. An error means there is no `Solution`: `ErrInvalidModel`, `ErrEngineUnavailable` or `ErrSolverFailed`, with the cause wrapped. go-milp returns an incumbent together with `ErrTimeLimit`; `lp` turns that into `Feasible` with a nil error so a caller who stops at `if err != nil` keeps the solution.
- **A CPLEX LP reader in Go** turns an LP file, or the text `lpgen` writes, into a go-milp model. It accepts what GLPK 5.0's reader accepts for the supported subset and reports anything else as `ErrInvalidModel` naming the line.
- **The GLPK engine** finds `glpsol` through `GLPK_PATH` (a file or a directory) and then `PATH`, runs it once with `--output` and `--write` (and `--tmlim` for a time limit), reads each value at full precision from the `--write` file and its name from the report, and decides the status from the `--write` status letter, using `glpsol`'s fixed messages when that letter is `u`. Measured on GLPK 5.0: an infeasible or unbounded LP reports `UNDEFINED` with presolve on and prints `LP HAS NO PRIMAL FEASIBLE SOLUTION` or `LP HAS UNBOUNDED PRIMAL SOLUTION`; a time limit prints `TIME LIMIT EXCEEDED; SEARCH TERMINATED`, with the incumbent when there is one; every one of these exits 0.
- **`lpgen` writes LP text to an `io.Writer`** (`WriteLP`), used by `GenerateLPFile` and by `Solve`, so a model has one text form. `SolveModel` used to write its own, with the objective sense spelled as given and no row labels.
- The auto-install code, the report-copying parser and the additional-info table are removed.
- `Docs/lp.md` documents the new API and how to install GLPK on macOS, Debian/Ubuntu, Fedora, Windows and conda. The capacity-planning tutorial uses the new API.
- A known gap is documented: go-milp v0.2.0 can report `Optimal` after skipping a node whose relaxation failed numerically ([go-milp#2](https://github.com/daniel-sullivan/go-milp/issues/2)).

## Capabilities

### New Capabilities

- `lp-solve`: how `lp` solves a model or an LP file, what `Solution` and its statuses mean, which outcomes are errors, how an engine is chosen, and what the LP reader accepts.

### Modified Capabilities

- `error-philosophy`: the DataTable requirement no longer names `lp`, whose solve functions no longer return DataTables, and the requirement about the additional-info table's `Status` is removed.
- `deterministic-and-atomic-output`: the requirement on the additional-info table's row order is removed with the table.
- `test-suite-integrity`: the solver-free requirement describes the new split: the go-milp engine and the LP reader need no external tool, the GLPK parser is tested on recorded `glpsol` output, and GLPK end-to-end tests skip when `glpsol` is missing.

## Impact

- **BREAKING**: `SolveFromFile` and `SolveModel` are gone. `result, info := lp.SolveModel(model, 10)` becomes `sol, err := lp.Solve(model, lp.Options{TimeLimit: 10 * time.Second})`, and the first table becomes `sol.ToDataTable()`.
- **BREAKING**: `lp` no longer installs GLPK. The default engine needs nothing installed; `Engine: lp.EngineGLPK` needs `glpsol` and returns `ErrEngineUnavailable` without it.
- Values are full precision on both engines.
- `lpgen` gains `WriteLP`. `GenerateLPFile`'s output is unchanged.
- `go.mod` gains `github.com/daniel-sullivan/go-milp v0.2.0`.
