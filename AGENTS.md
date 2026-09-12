# AGENTS.md

This file is the primary guidance for AI coding agents working in this repository. `CLAUDE.md` re-exports it via `@AGENTS.md`.

## What This Repo Is

**Insyra** (`github.com/HazelnutParadise/insyra`) is a Go data analysis library (v0.3.x, "Huashan") providing:
- `DataList` and `DataTable` as the core data structures
- **CCL** (Column Calculation Language) — a domain-specific expression language for column transforms
- A CLI/REPL (`insyra` binary) built with Cobra
- Sub-packages for stats, plotting, LP, Python interop, parallel processing, file I/O, etc.

## Required artifacts

Each entry says *when* reading it stops being optional. That trigger is the part that only exists here — the files describe their own contents.

- [`ENG.md`](ENG.md) — the project's standing technical decisions: architecture, test seams, the precision contract, the measured thresholds, the assumptions everything rests on. **Read it before changing architecture, choosing a test seam, deciding what precision something runs at, or writing a migration.** Do not re-derive any of it from scratch; if a decision there is wrong, change it there.
- [`delivery-status.md`](delivery-status.md) — where the project currently stands: phase, blockers, next output, next ticket. **Read it first on arrival.** It is deltas only, not a roadmap or a changelog.
- [`openspec/changes/`](openspec/changes/) — the tickets. **Pick up anything whose blockers are all done**; the milestone order in `delivery-status.md` carries the blocking edges, because OpenSpec has none between changes. Completed changes live in `archive/`.

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
When in doubt, propose a change. The docs/changelog/skills-in-sync rule below
still applies to whatever the change touches.

`CLAUDE.md` (and `cli/CLAUDE.md`, `stats/CLAUDE.md`) are `@AGENTS.md` includes,
so this policy applies to every agent tool that reads either file.

## Acceleration (`accel`) Operating Contract

Applies to every accel-related task. It extends the OpenSpec workflow above; it does not replace it.

### Required Entry Sequence
- Read `delivery-status.md` before doing any accel-related work.
- Use `delivery-status.md` as the source of truth for current phase, blockers, next verifiable output, and next OpenSpec change.
- Read the named OpenSpec change before proposing implementation or writing code.

### Required Artifacts
- `delivery-status.md` is the shared progress and handoff surface.
- `openspec/changes/` holds the executable units of work.

### Planning Discipline
- The accel phase may not use umbrella proposals. One change must produce one verifiable result.
- Do not start implementation for uncovered accel scope. Missing proposal coverage means the work is out of bounds.
- Full GPU string kernels are not a deferred track any more; the change that deferred them was withdrawn on 2026-08-01. A string kernel produces new values, which the result-shape rule below places behind an explicit opt-in rather than in a later phase. Do not reintroduce the deferral.
- Preserve the fixed architecture defaults unless a new decision is logged in `delivery-status.md`:
  - optional `insyra/accel` package family
  - `CUDA + Metal + WebGPU native`
  - heterogeneous multi-GPU only for shardable columnar operations in v1
  - observable CPU fallback by default, strict GPU-only as opt-in

### Update Discipline
- Update `delivery-status.md` after every milestone, blocker, or handoff.
- Change the named next OpenSpec change when the recommended pickup point changes.
- Update this contract only when operating rules change.

### Accel OpenSpec Rules
- Every active accel stage item must map to one OpenSpec change.
- Every OpenSpec change must map to one milestone and one verifiable output.
- Validate changed proposals with `openspec validate <change-id> --strict` before handoff.
- Do not merge unrelated capability slices into one change.

### Handoff Requirements
Every accel handoff must include:
- current phase
- blocker status
- next verifiable output
- next OpenSpec change
- decision delta since previous handoff
- source links for critical context
- whether `delivery-status.md` changed
- whether `AGENTS.md` changed

### How the CPU and the GPU Divide the Work

Acceleration is meant to be on by default and to change no numbers. Those two goals only hold together under one arrangement: **the GPU proposes, the CPU decides.** The device does the bulk arithmetic in `f32` and narrows the answer down; the CPU settles what is left in `float64`. Never send a `float64` result to the device and hand back what comes off it.

Whether an operation may be accelerated by default is decided by the **shape of its result**, not by how hot it is:

| Result shape | Default | Why |
| --- | --- | --- |
| A selection — which row, which index, what order | on | The device's `f32` ranking is a proposal. The CPU recomputes the shortlist in `float64` and picks, so the answer is exact. |
| Values in a type the device holds exactly — native `float32`, integers inside `int32` range, `bool` | on | Bit-identical outright; no verification needed. |
| New `float64` values — elementwise math, CCL value expressions | opt-in only | Nothing verifies them more cheaply than recomputing them, and WebGPU has no `f64`. Requires an explicit `PrecisionFloat32`. |

Implementing a selection-shaped operation:

1. The device returns the best *k* candidates per row, not the winner, plus enough information to tell when the boundary between candidate *k* and candidate *k+1* falls inside the `f32` error bound.
2. The CPU recomputes those *k* candidates in `float64` and decides.
3. Rows whose boundary is untrustworthy are recomputed against every candidate. Measured on cluster assignment this was 12 rows in 200,000, so the exact path costs almost nothing.
4. The result is asserted equal to the pure-`float64` reference, not merely close to it.

Two rules follow and are not negotiable:

- **A missing or broken device is a performance event, never a correctness one.** The verification half is a complete implementation, so no device means every row takes the full CPU path — the code that already runs for the untrustworthy rows. Every operation returns its answer whether or not a device ran; `Accelerated` and `FallbackReason` report where the work happened, not whether it worked. The exceptions are a request the caller's own terms made ineligible, and strict GPU mode, which exists to fail.
- **Do not write a kernel for an operation measurement says the device loses.** Memory-bound work — column sums, means, simple scans — stays on the CPU permanently. Measure before proposing a kernel, and record the number in `delivery-status.md`.

### Implementation Constraints
- Do not silently reinterpret existing `DataList.Map(func...)` or `DataTable.Map(func...)` as GPU kernels.
- Keep accel runtime opt-in and package-scoped until the relevant OpenSpec changes are implemented and approved.
- Treat Apple shared-memory residency separately from discrete VRAM in specs and docs.
- Keep CLI/DSL exposure aligned with the named accel change; do not implement commands outside validated proposal scope.

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

# Lint (CI uses golangci-lint; it also fails on a file gofmt would change)
golangci-lint run

# Vulnerability check (CI uses govulncheck)
govulncheck ./...
```

## Dependency Refresh Before a Release

Dependencies move on their own schedule; ours is the release. **Before `dev` is merged into `main`, refresh every dependency to the newest version that leaves the `go` directive in `go.mod` unchanged.**

- Do it as its own change, ahead of the release commit, so a bump that misbehaves is visible on its own and can be reverted without touching the release.
- Verify with `go build ./...`, `go test ./...` and `govulncheck ./...`.
- **Never let a bump raise the `go` directive.** The minimum-Go promise to downstream users is a separate, explicit decision. Stop at the newest version that keeps the current directive.
- Anything held back — its newest version needs a newer Go, or it breaks a tool CI depends on — goes into the Follow-ups below with the reason, the way the chromedp chain already is.

Why this is a rule and not a habit: dependencies only moved when Dependabot filed an alert, which means the graph only moved once something was already broken, and the fix was taken under time pressure. Dependabot also reads the default branch, so an alert raised against a released version stays open until the next merge to `main` no matter how quickly it is fixed on `dev`.

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
- `AtomicDo` serializes access to ONE instance (same-instance nesting is safe, e.g. `Stdev`→`Var`). To read/operate on MULTIPLE instances atomically, use `insyra.AtomicDoAll(func(){...}, a, b, ...)` — it locks all given DataList/DataTable instances together in a deadlock-free order. Call it from the outermost level: inside an `AtomicDo` on one of those instances it runs inline without locking the others (the same trust-zone rule as nested `AtomicDo`), because locking there would deadlock against a goroutine doing the mirror image. Do NOT nest `AtomicDo` on a *different* instance inside a callback: that inner call runs WITHOUT locking the other instance and can race a concurrent mutation. (`engine/atomic.AtomicDoN([]*Actor, f)` is the same primitive for arbitrary user structs holding an `*atomic.Actor`.)
- **The library never terminates or panics by default.** `LogFatal` records and returns; `Config.SetPanicOnError(true)` is the opt-in that turns any recorded error into a recoverable `panic`. Never call `os.Exit` or `panic` from library code — record the error and return something usable.
- **Two error shapes, no others.** An ordinary function returns `(T, error)`. A chainable method (`DataList`, `DataTable`, `isr`) returns a usable receiver or result and records the error on it — never `nil`. `TestChainableMethodsNeverReturnNil` in the root package enforces the second half across the whole module: it parses every non-test file and fails on a method that returns its receiver's type with no `error` result and a literal `return nil`. Inside those types use `fail(...)` for a call that could not do what it was asked — a bad argument, an unreadable cell, a mutation that could not be applied, or addressing a column/row/index that is not there — and `warn(...)` for a normal outcome: searching for a value and not finding it, an empty input, a value skipped by a documented convention. `warn` must not touch `Err()`. Because `Err()` is sticky, a public method that reports its own failure must look things up through the silent internal helpers (`colSilently`, `colByNameSilently`, `rowSilently`, `findFirstIndex`), or the inner lookup records first and the error points at an internal step instead of the call the user made.
- Error handling uses an instance-level `Err()` pattern rather than returning errors from every method. `Err()` is **sticky**: it keeps the first failure until `ClearErr()` or `PopErr()`, so a chain is checked once at the end and reports the root cause. Wrapper packages record through the exported `SetErr(packageName, funcName, msg, args...)`.
- The `isr` package is the recommended public API for new projects; the root `insyra` package is the implementation layer.

## Docs, Changelog & Skills Must Stay in Sync

Docs, the changelog, and skills are part of a change, not a follow-up. A feature is not done until these are updated in the **same** change.

**When adding a new package:**
- Create its doc page `Docs/<pkg>.md` (follow an existing page such as [Docs/finance.md](Docs/finance.md) / [Docs/stats.md](Docs/stats.md) for structure).
- Add a row to the package table in **both** README entry points — `## Packages` in [README.md](README.md) **and** `## 套件` in [README_TW.md](README_TW.md) — linking to `/Docs/<pkg>.md`.
- Update the docs index [Docs/README.md](Docs/README.md) (the docsify home). `Docs/_sidebar.md` is generated — don't edit it by hand.
- Register the package in [allpkgs/allpkgs.go](allpkgs/allpkgs.go).

**When adding or changing any feature (new or existing package):**
- Update the relevant `Docs/*.md` page(s) to match the new/changed API.
- Update the agent skills so they reflect the change: [skills/insyra/](skills/insyra/) (Go API usage — `SKILL.md` and `references/`), and [skills/use-insyra-cli/](skills/use-insyra-cli/) when CLI/DSL usage is affected.
- When the change touches the CLI/REPL or the DSL, update the CLI (`cli/`) and its doc [Docs/cli-dsl.md](Docs/cli-dsl.md).

**When the change is visible to someone using the library or the CLI:**
- Add an entry under `## Unreleased` in **both** [CHANGELOG.md](CHANGELOG.md) and [CHANGELOG_TW.md](CHANGELOG_TW.md), in the same change. Under the OpenSpec workflow this belongs in the change's own `tasks.md` — never as a follow-up, and never written from memory at release time.
- Group entries under the same package headings the release notes use (`### Core`, `### CLI`, `` ### `stats` ``, …), one level deeper than a release note so that promoting `###` to `##` at release time produces the release note as-is. Append to the end of a package section rather than the top.
- Mark breaking changes the way past release notes do.
- Skip the entry when nothing user-visible changed: internal refactors, tests, formatting, assets, dependency bumps with no behavioral effect, and OpenSpec bookkeeping.
- At release time, rename `## Unreleased` to the version number in both files and open a fresh empty `## Unreleased` above it.
- Also at release time, bump `Version` in [version.go](version.go) in the same change — the startup banner and the CLI `version` command read it, and it does not follow the changelog on its own (v0.3.1 nearly shipped with the banner still saying v0.3.0; it was caught at the PR, not by any check).
- A release is always named with both the series name and the version number — "Huashan v0.3.1", never a bare "v0.3.1" — in the GitHub Release title and anywhere else the release is announced. The series name comes from `VersionName` in [version.go](version.go) and changes only when a new series starts.

Keep the English ([README.md](README.md), [CHANGELOG.md](CHANGELOG.md), `Docs/`) and Traditional-Chinese ([README_TW.md](README_TW.md), [CHANGELOG_TW.md](CHANGELOG_TW.md)) docs in lockstep — never update one side without the other.

## Agent Skills

[skills/insyra/](skills/insyra/) — for AI agents writing Go code using Insyra APIs.  
[skills/use-insyra-cli/](skills/use-insyra-cli/) — for AI agents operating via the CLI/REPL or `.isr` scripts.

## Follow-ups

Out-of-scope issues discovered during development, waiting for a decision. Delete an entry once it is resolved.

### [2026-09-12] — a decimal column is exact but is not a number to the rest of the library
- **Where**: `internal/utils` `IsNumeric` / `ToFloat64Safe`, and every numeric path behind them
- **What**: `parquet-foreign-column-types` made `Decimal128` and `Decimal256` read as a go-decimal `decimal.Decimal`, exact and sorting by value. It is a struct, so `IsNumeric` says no and `ToFloat64Safe` cannot read it, which means `Mean`, `Sum`, `Stdev` and CCL arithmetic refuse a decimal column naming the row. That follows the `time.Time` precedent exactly, and it is the reason the change did not go further on its own. But a money column is far likelier to want arithmetic than a date column is.
- **Suggestion**: the decision is whether a decimal is a number in insyra. If it is, `ToFloat64Safe` needs an arm for it, which today means going through `String()` and `strconv.ParseFloat` because go-decimal exposes no `Float64()`; adding one upstream would be cleaner and it is the same author's library. Weigh that against making `internal/utils`, which every value in the library passes through, depend on a decimal package.
- **Status**: pending

### [2026-09-12] — reading a Parquet file and writing it back still changes column types
- **Where**: `parquet/internal.go` `inferArrowType`
- **What**: the reader now handles twenty Arrow types; the writer still emits seven. So a read-then-write round trip downgrades: a `Date32` column comes back as a timestamp, a `Binary` column as a string, a decimal as its text, an `Int16` as an `Int64`. Nothing is silently wrong, but the file is not the file that went in. Found on 2026-09-12 while fixing #371, which only concerned the read half.
- **Suggestion**: `inferArrowType` infers from Go values, so it cannot tell an `int16` that came from a `Date32` column from any other. Carrying the source schema through a read would fix it properly; inferring `time.Time` to `Date64` and `[]byte`-bearing strings to `Binary` would not, and would guess wrong on ordinary data. Worth doing only if round-tripping is a use case someone has.
- **Status**: pending

### [2026-09-12] — the best-of-restarts rule follows GPArotation's engine, not psych's `faRotations`
- **Where**: `stats/internal/fa/psych_faRotations.go`, `preferCandidate` and the `hyper` parameter of `FaRotations`
- **What**: the file is named after `psych::faRotations`, which ranks the starts by hyperplane count (the share of loadings below `hyper = .15` in absolute value), then complexity, then fit, and rerotates from the winner. Ours keeps the lowest criterion value among the starts that converged, which is what `GPArotation::.GPA_RS_engine` does (strict minimum `f`, no convergence preference). `FaRotations`'s `hyper` parameter carries psych's name but oblimin's gamma. Seen on 2026-09-12 while reading both sources (psych 2.6.5, GPArotation 2026.8-2) for `oblimin-honours-its-start`; psych 2.6.5 also changed `fa()`'s `n.rotations` default from 1 to 20.
- **Suggestion**: two decisions, both about which reference this file claims to follow: the selection rule (criterion value versus hyperplane count), and whether `Restarts` should follow psych's new default of 20, which would move the default output. Neither is a defect. Both change results, the first at `Restarts > 1`, the second by default.
- **Status**: pending

### [2026-09-12] — 46 of the 105 archived specs have a `## Purpose` nobody wrote
- **Where**: `openspec/specs/*/spec.md`, the `## Purpose` section
- **What**: `openspec validate --specs --strict` on 2026-09-12 reported 46 failures, and every one is the same shape: 26 specs still carry the placeholder sentence `openspec archive` writes for a new capability (`TBD - created by archiving change <id>. Update Purpose after archive.`) and 20 have a Purpose under 50 characters. Eight also have a requirement over 500 characters. The failures are not new and nothing in CI runs this command, which is why they accumulated — the capability is created by the first change that touches it, the placeholder lands in the main spec, and a `## Purpose` written in a later delta is ignored because deltas only supply one at creation. `verification-integrity` was fixed in `docs-hygiene-and-remaining-partials` as an example of the size the replacement should be.
- **Suggestion**: not a mechanical pass. Each Purpose is one or two sentences saying what the capability is for, which means reading the requirements under it, so this is 46 small judgements rather than one edit. Do it as its own change, and add `openspec validate --specs --strict` to the lint workflow afterwards so the count cannot climb again. Until then, anyone archiving a change that creates a capability must write its Purpose in the same commit.
- **Status**: pending

### [2026-09-11] — `insyra run` exits 0 when a line fails
- **Where**: `cli/commands/run.go`, the loop that prints `line N: <error>` and moves on
- **What**: `Docs/cli-dsl.md` says `run` continues after a failing line, and it does, but it then prints `script complete` and exits 0, so a shell script or CI job running `insyra run job.isr` cannot tell that a step failed. Measured on 2026-09-11: a script whose third line was rejected exited 0. The Go `Session.ExecuteFile` stops at the first error and returns it.
- **Suggestion**: keep continuing if that is the intended behaviour, but exit non-zero when any line failed (for example "script finished, 1 of 3 lines failed"), or add a stop-on-error option. Either way the exit code belongs in the docs.
- **Status**: pending

### [2026-09-11] — `Counter()` still keeps `int(1)` and `int64(1)` apart
- **Where**: `datalist.go` `DataList.Counter`, `datatable.go` `DataTable.Counter`
- **What**: since `match-integers-by-value`, every search, count, replace and drop matches integers by value, and GroupBy, Pivot and Merge already did (`encodeGroupKey` writes every integer as `i:<value>`). `Counter()` is the one place left that does not: it returns a `map[any]int` keyed by the stored value, so a column holding both `int(1)` and `int64(1)` reports two keys while `Count(1)` reports their total. A column only mixes the two when rows are added by hand to loaded data, for example Go literals appended to a CSV table.
- **Suggestion**: merging them means the map key has to be one of the two stored values, which decides which literal a caller can index it with (`counter[1]` or `counter[int64(1)]`). Pick one rule, or document that `Counter` reports stored types, rather than leave it implicit.
- **Status**: pending

### [2026-09-11] — 53 CLI commands accept a trailing argument and ignore it
- **Where**: `cli/commands/`, one `Run` function per command
- **What**: cli/AGENTS.md says an argument a command does not understand is an error, because an ignored one makes a typo look like it worked. `cli-reject-ignored-args` fixed `accel` and the nine DataList statistics. An audit on 2026-09-11 that appended `junk` to a valid call found these still exit 0 and ignore it: `iqr`, `skewness`, `kurtosis`, `counter`, `quartile`, `percentile`, `count`, `cov`, `corr`, `corrmatrix`, `shape`, `types`, `summary`, `cols`, `rows`, `get`, `set`, `find`, `replace`, `swap`, `transpose`, `rename`, `drop`, `clone`, `reverse`, `normalize`, `standardize`, `parsenums`, `parsestrings`, `upper`, `lower`, `capitalize`, `cumsum`, `cumprod`, `cummax`, `cummin`, `diff`, `diffn`, `pctchange`, `movavg`, `expsmooth`, `ttest` (all three forms), `ztest single`, `ftest var`, `pca`, `dbscan`, `vars`, `history`, `version`, `help`, `config`, `run` and `exit`. `show`, `shift`, `rank` and `sort` already reject it. Commands that take a variable number of arguments (`newdl`, `addcol`, `dropcol`, …) were not audited.
- **Suggestion**: declare each command's maximum argument count at registration and check it in one place, instead of fifty hand-written checks. The decision is the count for every command, including how `as <var>` and optional trailing arguments (`sort <var> <col> [asc|desc]`) are counted.
- **Status**: pending

### [2026-09-10] — a nil `*DataList` / `*DataTable` panics even on `Err()`
- **Where**: `datalist.go` `Err()`, `datatable.go` `Err()`
- **What**: the library's stated principle is that it never terminates by default, but its own error accessor is a bare field read on a pointer receiver, so `dl.Err()` on a nil `dl` panics. That makes "ask what went wrong" the thing that crashes. It surfaced while fixing `ml`/`nn`'s `Classes()`, which used to hand out exactly such a nil; three independent analyses of that change landed on this as the underlying flaw rather than the ten methods.
- **Suggestion**: give `Err()` alone a nil-receiver guard returning a fixed `*ErrorInfo` ("nil DataList"), roughly three lines each. Do **not** extend this to every method: a nil receiver that silently answers `Len() == 0` lets the nil travel, which is the failure mode this review has spent most of its time removing. `Err()` is different because it is read-only, side-effect free, and is the one question whose answer should never be a crash. Decide it on its own; it is not a prerequisite for anything.
- **Status**: pending

### [2026-09-07] — remove `Config.SetDontPanic` one release after `SetPanicOnError` shipped
- **Where**: `config.go` (`SetDontPanic`, `GetDontPanicStatus`)
- **What**: The library never terminates or panics by default. `LogFatal` records the error and returns; `Config.SetPanicOnError(true)` is the opt-in that turns any recorded error into a `panic` (never `os.Exit`). `SetDontPanic(v)` remains one release as a Deprecated alias for `SetPanicOnError(!v)` so existing callers keep compiling. Shipped in `make-errors-non-terminating` (2026-09-07).
- **Suggestion**: In the first release after the one that ships `make-errors-non-terminating`, delete `SetDontPanic` and `GetDontPanicStatus`, drop the alias note from `Docs/Configuration.md`, and add a BREAKING changelog entry. Do not let the alias drift into a second release.
- **Status**: pending

### [2026-09-07] — delete the nine Deprecated global error-buffer accessors
- **Where**: `error_buffer.go` (`PopError`, `PopErrorByPackageName`, `PopErrorByFuncName`, `PopErrorAndCallback`, `PeekError`, `GetErrorsByLevel`, `GetErrorsByPackage`, `PopErrorInfo`, `HasErrorAboveLevel`)
- **What**: The global buffer is a diagnostic log, not an error-handling API — it mixes records from every goroutine and object. `make-errors-non-terminating` marked these nine Deprecated and kept `GetAllErrors`, `PopAllErrors`, `HasError`, `GetErrorCount` and `ClearErrors` as the supported surface.
- **Suggestion**: Delete them in the same release that drops `SetDontPanic`, with a BREAKING changelog entry.
- **Status**: pending

### [2026-08-01] — multi-GPU planning and execution coverage
- **Where**: `accel/planner.go` (`PlanShardable`, weighted per-device `ShardAssignment`s), `accel/exact.go` (per-assignment dispatch)
- **What**: the planner retains capability-weighted heterogeneous assignments and its existing `MergePolicy`. `ExecuteNearestExact` now dispatches one worker per assignment, uses the bounded chunk seam, merges by input range, and falls back per assignment without changing the exact CPU decision.
- **Suggestion**: run `INSYRA_ACCEL_GPU_TESTS=1 go test ./accel -run TestMultiDeviceParityConcurrentAndSequentialOnHardware` on a multi-GPU host, then record concurrent-versus-sequential wall clock for the 32k/8k saturation classes. The single-device host verifies correctness only; it cannot supply a multi-GPU speedup number.
- **Status**: single-device correctness verified; multi-GPU wall clock and non-Apple parity remain pending

### [2026-08-01] — `ToF64Slice` still fabricates zeros for 54 callers outside `stats`
- **Where**: `datalist.go` `ToF64Slice`, and its callers in `plot/`, `gplot/`, `cli/`
- **What**: it routes every value through `insyra.ToFloat64`, which has no failure channel and yields `0` for anything it cannot parse, then returns a full-length slice — so a caller cannot tell a real zero from a value that was never read. `stats` was moved off it on 2026-08-01 after a blank among six observations was measured moving a Pearson coefficient from 0.9992 to 0.9879. `quant` followed on 2026-09-05 (`fix-quant-legacy-numeric-input`): its last five call sites — `SharpeRatio`, `MaxDrawdown`, `AnnualizedReturn`, `DeflatedSharpeRatio`, and `PBO`'s column loop — now read through `numericSeries`, so the whole package refuses an unreadable cell instead of zeroing it. On 2026-09-05 (`fix-api-review-batch-1`) `DataList.Rank`, `ExponentialSmoothing`, `DoubleExponentialSmoothing`, the six `*Interpolation` methods, and `stats.Skewness`/`Kurtosis` (which used the sibling `SliceToF64`) were moved off it as well, through the new `numericCells` read path. The remaining callers were left deliberately: they are display and reporting paths, where a fabricated zero shows up as a point on a chart rather than inside a coefficient.
- **Suggestion**: leave them, but decide rather than drift. If they are to stay, the method's doc comment should say what it does with a value it cannot read, because it currently does not. A new numeric analysis must not use it.
- **Status**: pending

### [2026-08-01] — the GPU backend is verified on Apple and Metal only
- **Where**: `accel/`, the whole device path
- **What**: every numeric result the backend produces has been checked bit-for-bit against its CPU reference, but only on an Apple M3 through Metal. `add-accel-gpu-execution` required the same check on a Windows or Linux host with an NVIDIA or AMD GPU before archiving; no such machine was available and the task was closed without it. Bit-parity depends on both toolchains contracting multiply-add identically, which is a property of the platform rather than of the kernel — so it is measured where it runs and cannot be inferred for a platform nobody has run it on. Vulkan and DirectX 12 paths compile and are exercised by no numeric test.
- **Suggestion**: run `INSYRA_ACCEL_GPU_TESTS=1 go test ./accel/...` on a non-Apple host with a discrete GPU. If parity fails there, the fix is not a kernel change but a per-platform gate: the exact-nearest operation already recomputes untrustworthy rows in `float64`, so a platform that cannot match bit-for-bit can widen that path rather than lose the operation. Also worth measuring there is whether the profitability threshold moves — PCIe transfer should push it up, not down, and the value in `accel/exact.go` is calibrated on unified memory.
- **Status**: pending

### [2026-08-01] — accel still transports strings that nothing can consume
- **Where**: `accel/dataset.go` (string buffers with offsets and data), `accel/cache.go` (their byte accounting)
- **What**: projection builds encoded-string buffers and the cache charges for them, but no operation accepts a string column — the sole remaining device operation requires numeric columns. This is left over from `add-accel-string-kernels-phase-2`, withdrawn on the same day: string kernels produce new values, which the acceleration rules place behind an explicit opt-in rather than a future phase.
- **Suggestion**: leave it or remove it, but decide rather than drift. Keeping it costs a little projection work on tables with string columns and would be needed again by any future string operation; removing it trims code that is currently unreachable. Neither is urgent.
- **Status**: pending

### [2026-07-26] — ReadExcelSheet does no type inference (inconsistent with CSV)
- **Where**: [read.go](read.go) `ReadExcelSheet` → `ReadSlice2D`
- **What**: excelize `GetRows` returns strings and `ReadSlice2D` appends them as-is, so Excel loads produce all-string DataTables while CSV loads run column-level inference (`inferCSVColumnTypes`). Opposite defaults for the two spreadsheet formats. Noticed while adding `CSVReadOptions.RawStrings` (issue #188).
- **Suggestion**: Decide whether Excel reads should run the same column inference by default (with the same opt-out), or stay raw; either way document the behavior in `Docs/DataTable.md`.
- **Status**: pending

### [2026-07-11] — chromedp chain left at pre-refresh versions (two independent blockers)
- **Where**: `go.mod` — `chromedp v0.11.2`, `cdproto v0.0.0-20241208230723-d1c7de7e5dd2` (pulled in via `go-echarts/snapshot-chromedp`)
- **What**: The 2026-07-11 dependency refresh could not move these. (1) `chromedp v0.15.0+` and newer `cdproto` require go >= 1.26, while the module's `go` directive stays on 1.25.x (minimum-Go promise to downstream users). (2) The newest go1.25-compatible version, `chromedp v0.14.2`, hard-requires `go-json-experiment/json`, whose generic-variadic code panics govulncheck's symbol-level scan ("got jsontext.Value, want variadic parameter of unnamed slice or string type" in x/tools go/ssa — still broken as of x/tools v0.48.0 / x/vuln v1.6.0), which would permanently break the Govulncheck CI workflow.
- **Suggestion**: When raising the minimum Go version to 1.26, retry upgrading the whole chain and re-verify `govulncheck ./...` completes (the x/tools SSA bug may be fixed by then).
- **Status**: pending
