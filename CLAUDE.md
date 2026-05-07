# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Repo Is

**Insyra** (`github.com/HazelnutParadise/insyra`) is a Go data analysis library (v0.2.x, "Pier-2") providing:
- `DataList` and `DataTable` as the core data structures
- **CCL** (Column Calculation Language) — a domain-specific expression language for column transforms
- A CLI/REPL (`insyra` binary) built with Cobra
- Sub-packages for stats, plotting, LP, Python interop, parallel processing, file I/O, etc.

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
- Error handling uses an instance-level `Err()` pattern rather than returning errors from every method (check `.Err()` after chained calls).
- The `isr` package is the recommended public API for new projects; the root `insyra` package is the implementation layer.

## Agent Skills

[skills/insyra/](skills/insyra/) — for AI agents writing Go code using Insyra APIs.  
[skills/use-insyra-cli/](skills/use-insyra-cli/) — for AI agents operating via the CLI/REPL or `.isr` scripts.
