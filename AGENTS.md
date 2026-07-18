# AGENTS.md

This file is the primary guidance for AI coding agents working in this repository. `CLAUDE.md` re-exports it via `@AGENTS.md`.

## What This Repo Is

**Insyra** (`github.com/HazelnutParadise/insyra`) is a Go data analysis library (v0.3.x, "Huashan") providing:
- `DataList` and `DataTable` as the core data structures
- **CCL** (Column Calculation Language) — a domain-specific expression language for column transforms
- A CLI/REPL (`insyra` binary) built with Cobra
- Sub-packages for stats, plotting, LP, Python interop, parallel processing, file I/O, etc.

## Change Workflow — OpenSpec (required)

**Every non-trivial change goes through the OpenSpec workflow.** This repo is
OpenSpec-initialized (see `openspec/`). Do NOT implement a feature, breaking
change, new package, or substantial fix straight into the code — drive it
through a change first:

1. **Propose** — create the change (proposal + spec deltas + tasks) under
   `openspec/changes/<change-id>/` (`openspec-propose` / `openspec-new-change`).
2. **Apply** — implement the tasks (`openspec-apply-change`).
3. **Verify** — confirm the implementation matches the artifacts
   (`openspec-verify-change`).
4. **Archive** — once the change is implemented and merged, archive it into
   `openspec/changes/archive/` (`openspec-archive-change`). **Completed changes
   must be archived, not left sitting in `openspec/changes/`.**

Trivial edits (typos, formatting, comment/doc-only touch-ups) may skip OpenSpec.
When in doubt, propose a change. The docs/skills-in-sync rule below still applies
to whatever the change touches.

`CLAUDE.md` (and `cli/CLAUDE.md`, `stats/CLAUDE.md`) are `@AGENTS.md` includes,
so this policy applies to every agent tool that reads either file.

## Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./stats/...
go test ./isr/...

# Run a single test by name
go test -run TestFunctionName ./path/to/package/...

# Build the CLI binary
go build -o insyra ./cmd/insyra/

# Run the CLI REPL
go run ./cmd/insyra/

# Lint (CI uses golangci-lint)
golangci-lint run

# Vulnerability check (CI uses govulncheck)
govulncheck ./...
```

## Architecture

### Core Package (`github.com/HazelnutParadise/insyra`)

The root package defines everything central:
- [interfaces.go](interfaces.go) — `IDataList` and `IDataTable` interfaces (authoritative API surface)
- [datalist.go](datalist.go) / [datatable.go](datatable.go) — concrete implementations
- [ccl.go](ccl.go) — wires the CCL engine into `DataTable.AddColUsingCCL` etc.
- [config.go](config.go) — global `Config` singleton (log level, colored output, thread safety, panic behavior)
- [atomic.go](atomic.go) — actor-model serialization for thread-safe operations on DataList/DataTable
- [logger.go](logger.go) — structured logging used throughout

### Sub-packages

| Package | Purpose |
|---|---|
| `isr/` | Syntax-sugar wrappers — **preferred entry point for new code** |
| `stats/` | Statistical tests (t-test, ANOVA, chi-square, PCA, regression, …) |
| `plot/` | Interactive charts via go-echarts |
| `gplot/` | Static publication charts via gonum/plot |
| `csvxl/` | CSV and Excel I/O |
| `parquet/` | Parquet file I/O via Apache Arrow |
| `datafetch/` | HTTP data fetching helpers |
| `parallel/` | Parallel map/reduce over DataList/DataTable |
| `lp/` / `lpgen/` | Linear programming |
| `mkt/` | Market data helpers |
| `py/` | Python interop (runs Python via embedded env) |
| `pd/` | Pandas-like wrappers built on `py/` |
| `engine/` | Re-exported stable internals (BiIndex, Ring, AtomicDo, CCL compiler, sort utils) |
| `allpkgs/` | Blank-import of all packages for `go get` convenience |

### CLI (`cli/`)

Built with Cobra. Entry point: [cmd/insyra/main.go](cmd/insyra/main.go) → `cli.Execute()`.
- `cli/commands/` — individual subcommand implementations
- `cli/repl/` — interactive REPL and DSL session (`engine/dsl` exposes `DSLSession` for programmatic use)
- `cli/env/` — named environment management (variable persistence between sessions)
- `cli/style/` — terminal styling

### Internal packages (`internal/`)

Not exported to consumers. Key ones:
- `internal/ccl/` — CCL lexer, parser, AST, evaluator; `MapContext` for testing CCL without a DataTable
- `internal/core/` — `BiIndex` (bidirectional id↔name index), `Ring` (circular buffer)
- `internal/algorithms/` — parallel stable sort, `CompareAny` for mixed-type ordering

### CCL (Column Calculation Language)

CCL has two modes used by different DataTable methods:
- **Expression mode** (`AddColUsingCCL`, `EditColByIndexUsingCCL`, `EditColByNameUsingCCL`) — pure expressions only, no assignment
- **Statement mode** (`ExecuteCCL`) — supports assignment syntax and `NEW()` for creating columns

Column references use Excel-style indices (`A`, `B`, … `AA`, `AB`, …) or named references `['ColName']`.

## Key Conventions

- Column indexing is Excel-style alphabetic (`"A"`, `"B"`, `"AA"`), not numeric, for `GetCol`/`UpdateCol` etc.
- `GetRowIndexByName` returns `(-1, false)` when not found — always check the boolean, because `-1` is also a valid "last element" index in many `Get` methods.
- Thread safety is on by default via the actor model. `Config.Dangerously_TurnOffThreadSafety()` exists but is explicitly discouraged.
- `AtomicDo` serializes access to ONE instance (same-instance nesting is safe, e.g. `Stdev`→`Var`). To read/operate on MULTIPLE instances atomically, use `insyra.AtomicDoAll(func(){...}, a, b, ...)` — it locks all given DataList/DataTable instances together in a deadlock-free order. Do NOT nest `AtomicDo` on a *different* instance inside a callback: that inner call runs WITHOUT locking the other instance and can race a concurrent mutation. (`engine/atomic.AtomicDoN([]*Actor, f)` is the same primitive for arbitrary user structs holding an `*atomic.Actor`.)
- Error handling uses an instance-level `Err()` pattern rather than returning errors from every method (check `.Err()` after chained calls).
- The `isr` package is the recommended public API for new projects; the root `insyra` package is the implementation layer.

## Docs & Skills Must Stay in Sync

Docs and skills are part of a change, not a follow-up. A feature is not done until these are updated in the **same** change.

**When adding a new package:**
- Create its doc page `Docs/<pkg>.md` (follow an existing page such as [Docs/finance.md](Docs/finance.md) / [Docs/stats.md](Docs/stats.md) for structure).
- Add a row to the package table in **both** README entry points — `## Packages` in [README.md](README.md) **and** `## 套件` in [README_TW.md](README_TW.md) — linking to `/Docs/<pkg>.md`.
- Update the docs index [Docs/README.md](Docs/README.md) (the docsify home). `Docs/_sidebar.md` is generated — don't edit it by hand.
- Register the package in [allpkgs/allpkgs.go](allpkgs/allpkgs.go).

**When adding or changing any feature (new or existing package):**
- Update the relevant `Docs/*.md` page(s) to match the new/changed API.
- Update the agent skills so they reflect the change: [skills/insyra/](skills/insyra/) (Go API usage — `SKILL.md` and `references/`), and [skills/use-insyra-cli/](skills/use-insyra-cli/) when CLI/DSL usage is affected.
- When the change touches the CLI/REPL or the DSL, update the CLI (`cli/`) and its doc [Docs/cli-dsl.md](Docs/cli-dsl.md).

Keep the English ([README.md](README.md), `Docs/`) and Traditional-Chinese ([README_TW.md](README_TW.md)) docs in lockstep — never update one side without the other.

## Agent Skills

[skills/insyra/](skills/insyra/) — for AI agents writing Go code using Insyra APIs.  
[skills/use-insyra-cli/](skills/use-insyra-cli/) — for AI agents operating via the CLI/REPL or `.isr` scripts.

## Follow-ups

Out-of-scope issues discovered during development, waiting for a decision. Delete an entry once it is resolved.

### [2026-07-11] — chromedp chain left at pre-refresh versions (two independent blockers)
- **Where**: `go.mod` — `chromedp v0.11.2`, `cdproto v0.0.0-20241208230723-d1c7de7e5dd2` (pulled in via `go-echarts/snapshot-chromedp`)
- **What**: The 2026-07-11 dependency refresh could not move these. (1) `chromedp v0.15.0+` and newer `cdproto` require go >= 1.26, while the module's `go` directive stays on 1.25.x (minimum-Go promise to downstream users). (2) The newest go1.25-compatible version, `chromedp v0.14.2`, hard-requires `go-json-experiment/json`, whose generic-variadic code panics govulncheck's symbol-level scan ("got jsontext.Value, want variadic parameter of unnamed slice or string type" in x/tools go/ssa — still broken as of x/tools v0.48.0 / x/vuln v1.6.0), which would permanently break the Govulncheck CI workflow.
- **Suggestion**: When raising the minimum Go version to 1.26, retry upgrading the whole chain and re-verify `govulncheck ./...` completes (the x/tools SSA bug may be fixed by then).
- **Status**: pending

### [2026-07-10] — `types` CLI command prints to stdout, bypassing `ctx.Output`
- **Where**: `cli/commands/types.go:28,30` → `DataTable/DataList.ShowTypes` (`show.go:437,1041`)
- **What**: Same bug class as M20 (fixed for `show`/`describe`/`summary`). `ShowTypes`/`ShowTypesRange` were intentionally left stdout-only, so the `types` command's output escapes `ctx.Output` (not captured by the REPL/tests).
- **Suggestion**: Extend the writer-aware pattern — add `ShowTypesTo(w)` / `ShowTypesRangeTo(w, ...)` (thread `w` through `printTypeRows`), have the old methods delegate to `os.Stdout`, and route `types.go` through `ctx.Output`.
- **Status**: pending

### [2026-07-10] — `go updateTimestamp()` spawns an unbounded goroutine per mutation
- **Where**: ~102 sites across `datalist.go` / `datatable.go` (mutating methods call `go x.updateTimestamp()`)
- **What**: Every mutation fires a fire-and-forget goroutine to bump the last-modified timestamp. Under heavy/tight-loop mutation this floods the scheduler (the `-race` stress tests had to be bounded because of it). Latent scalability/perf concern, not a correctness bug.
- **Suggestion**: Update the timestamp synchronously (it is a cheap atomic store) instead of spawning a goroutine, or coalesce. First confirm the goroutine is not there to avoid re-entering the actor lock.
- **Status**: pending
