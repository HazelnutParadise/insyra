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
- Verify with `go build ./...`, `go test ./...` and `govulncheck ./...`, and check every module in `go.mod` against GitHub's advisory database (`gh api graphql` with `securityVulnerabilities(ecosystem: GO, package: …)`).
- **Never move a module into a vulnerable range.** Stop at the newest version outside every range.
- **Never let a bump raise the `go` directive.** The minimum-Go promise to downstream users is a separate, explicit decision. Stop at the newest version that keeps the current directive.
- Anything held back — its newest version needs a newer Go, is inside an advisory, or breaks a tool CI depends on — goes into the Follow-ups below with the reason, the way the chromedp chain already is.

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
| `isr/` | Syntax-sugar wrappers — **for convenient code; the root package for performance** |
| `stats/` | Statistical tests (t-test, ANOVA, chi-square, PCA, regression, …) |
| `plot/` | Interactive charts via go-echarts |
| `gplot/` | Static publication charts via gonum/plot |
| `csvxl/` | CSV and Excel I/O |
| `parquet/` | Parquet file I/O via Apache Arrow |
| `datafetch/` | HTTP data fetching helpers |
| `parallel/` | Run zero-argument functions at once and collect their results (`GroupUp(...).Run().AwaitResult()`) |
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

- **One column selector, everywhere.** Every parameter and config field that picks a column takes the same three forms: a `string` is an Excel-style index (`"A"`, `"B"`, `"AA"`), `Name("price")` is a column name, an `int` is a 0-based position (negative counts from the end). A bare string is never looked up as a name, so a call means the same thing whatever the columns are called; the column names shape only the failure message, which offers the `Name(...)` form when the table has that name. The core accessors keep explicit spellings beside the generic one (`GetColByIndex`/`ByName`/`ByNumber`, and the same for `UpdateCol`), and the analysis operations do not, because the selector already says which kind it was given. `isr.Name` is the same type as `insyra.Name`.
- `GetRowIndexByName` returns `(-1, false)` when not found — always check the boolean, because `-1` is also a valid "last element" index in many `Get` methods.
- Thread safety is on by default via the actor model. `Config.Dangerously_TurnOffThreadSafety()` exists but is explicitly discouraged.
- `AtomicDo` serializes access to ONE instance (same-instance nesting is safe, e.g. `Stdev`→`Var`). To read/operate on MULTIPLE instances atomically, use `insyra.AtomicDoAll(func(){...}, a, b, ...)` — it locks all given DataList/DataTable instances together in a deadlock-free order. Call it from the outermost level: inside an `AtomicDo` on one of those instances it runs inline without locking the others (the same trust-zone rule as nested `AtomicDo`), because locking there would deadlock against a goroutine doing the mirror image. Do NOT nest `AtomicDo` on a *different* instance inside a callback: that inner call runs WITHOUT locking the other instance and can race a concurrent mutation. (`engine/atomic.AtomicDoN([]*Actor, f)` is the same primitive for arbitrary user structs holding an `*atomic.Actor`.)
- **The library never terminates or panics by default.** `LogFatal` records and returns; `Config.SetPanicOnError(true)` is the opt-in that turns any recorded error into a recoverable `panic`. Never call `os.Exit` or `panic` from library code — record the error and return something usable.
- **Two error shapes, no others.** An ordinary function returns `(T, error)`. A chainable method (`DataList`, `DataTable`, `isr`) returns a usable receiver or result and records the error on it — never `nil`. `TestChainableMethodsNeverReturnNil` in the root package enforces the second half across the whole module: it parses every non-test file and fails on a method that returns its receiver's type with no `error` result and a literal `return nil`. Inside those types use `fail(...)` for a call that could not do what it was asked — a bad argument, an unreadable cell, a mutation that could not be applied, or addressing a column/row/index that is not there — and `warn(...)` for a normal outcome: searching for a value and not finding it, an empty input, a value skipped by a documented convention. `warn` must not touch `Err()`. Because `Err()` is sticky, a public method that reports its own failure must look things up through the silent internal helpers (`colSilently`, `colByNameSilently`, `rowSilently`, `findFirstIndex`), or the inner lookup records first and the error points at an internal step instead of the call the user made.
- Error handling uses an instance-level `Err()` pattern rather than returning errors from every method. `Err()` is **sticky**: it keeps the first failure until `ClearErr()` or `PopErr()`, so a chain is checked once at the end and reports the root cause. Wrapper packages record through the exported `SetErr(packageName, funcName, msg, args...)`.
- Use the `isr` package for convenient syntax and the root `insyra` package where performance matters: some `isr` constructors convert or copy their input (building a table from lists measured about 9x slower than `insyra.NewDataTable`), while methods called through the wrapper cost almost nothing extra.
- **The interfaces are sealed but embeddable.** `IDataList` and `IDataTable` carry a hidden method, so nothing outside the module implements them from scratch, and that is what lets a method be added to them without breaking anyone. A type that embeds `*DataList`/`*DataTable` inherits every method, the hidden ones included, so it satisfies them and goes wherever a list or table goes; `Merge` reaches the embedded table through the hidden `coreTable()`. The interfaces list every exported method of the concrete types, except the ones an extension in this module overrides with its own signature: `ClearErr`, `SetErr` (isr returns its own type to keep chaining) and `Pivot`, `Unpivot` (isr's short-syntax versions). `TestInterfacesListEveryMethod` enforces the rule and names each exception with its reason; a new exception needs its reason written there. Sub-packages take the interfaces as parameter types. Ruled by the owner on 2026-09-26 (#208).
- **A file's header row and row names have one spelling each.** Every options struct that reads or writes a table names them `NoHeaderRow` (the file has no header row) and `HasRowNames` (the file's first column holds row names), for reading and writing alike: the name describes the file, not the action, and the zero value is the common file, with column names and without row names. They replace `FirstRowToColNames`/`SetColNamesToFirstRow`/`FirstRow2ColNames`/`ColNames2FirstRow` and their row-name counterparts, and `ToSQLOptions.RowNames`; `RowNames` alone was rejected because `dt.RowNames()` returns the names themselves. A bool passed directly keeps its meaning and is only renamed (`headerRow`, `rowNames`): flipping a positional bool would silently invert every existing call. Ruled by the owner on 2026-09-25 (#213).
- **How a function takes settings.** Ruled by the owner on 2026-09-25 while deciding #213. First sort each setting into one of two kinds; the test is whether you would write it in the methods section of a report.
  - **Statistical settings go in an options struct, whatever their number.** They decide how the analysis is done: KNN's neighbour count and weighting, whether a sample is drawn with replacement and from which seed, the percentiles `Describe` reports, BatchNorm's momentum and epsilon. They grow as a method gains options, they are reused across training, testing and cross-validation and written into reports, and they belong together, so one value holds them. The exception is a function with a single statistical setting that nobody expects to gain another soon: that setting stays an ordinary parameter.
  - **A setting with no safe default is a required parameter**, even a statistical one, and never goes in an options struct, because an options field always has a default. Whether a sample is drawn with replacement is the case that set this: forgetting `true` in a bootstrap would silently shuffle the data instead, so `Sample(n, withReplacement bool, …)` keeps the bool required. The same holds for any choice that changes what the result means rather than how it is computed. Ruled by the owner on 2026-09-25.
  - **Programming settings follow the count.** They only shape how one call runs: sort direction, how far a fill reaches, what a shift fills in, whether a saved chart animates, a file's encoding. Four or more go in an options struct. Three or fewer are ordinary parameters, and only the last one, when it has an obvious default, may be left out through a trailing `...T` (`dl.Sort()` and `dl.Sort(false)`).
  - **An options struct is taken as an optional trailing `opts ...XxxOptions`**, so the plain call stays short (`dt.Describe()`).
  - **More than one value is always an error**, for a trailing `...T` and a trailing `...XxxOptions` alike, reported through the function's normal error path: never "first wins", "last wins" or silently ignored (`dl.Sort(true, false)` fails).
  - **A bool stays a bool**, its meaning carried by the parameter name (`Sort(ascending ...bool)`). A named constant type (`insyra.Descending`) was rejected: it would be thrown away the day the function grows into an options struct.
  - **No option functions.** Insyra does not use the `WithXxx(...)` pattern (`Describe(WithPercentiles(0.1, 0.9))`), and never has. A setting belongs in the function's own signature or options struct, where it shows in the documentation; option functions are scattered across the package, so someone who does not already know the library cannot see what can be set.
  - **`Options` or `Config`.** A settings pack that can be left out, every field usable at its zero value, is named `XxxOptions`; a struct that describes the job itself and has fields that must be filled (`PivotConfig`, a chart config, `RFMConfig`) is named `XxxConfig`. Existing names are not changed to fit: about seventy types would break every caller for no difference in behaviour. The `isr` package keeps its short `Opts` spelling (`CSV_inOpts`, `Excel_inOpts`) because it is the short-syntax layer; its fields still follow the header-row and row-name spelling below.
  - **An options struct that already exists stays one**, even with three or fewer fields: turning it back into parameters would break every caller without making any call clearer.

## Docs, Changelog & Skills Must Stay in Sync

Docs and the changelog are part of a change, not a follow-up. A feature is not done until they are updated in the **same** change. The agent skills follow a different rule: they change only when what they teach does, as described below.

**When adding a new package:**
- Create its doc page `Docs/<pkg>.md` (follow an existing page such as [Docs/finance.md](Docs/finance.md) / [Docs/stats.md](Docs/stats.md) for structure).
- Add a row to the package table in **both** README entry points — `## Packages` in [README.md](README.md) **and** `## 套件` in [README_TW.md](README_TW.md) — linking to `/Docs/<pkg>.md`.
- Update the docs index [Docs/README.md](Docs/README.md) (the docsify home). `Docs/_sidebar.md` is generated — don't edit it by hand.
- Register the package in [allpkgs/allpkgs.go](allpkgs/allpkgs.go).

**When adding or changing any feature (new or existing package):**
- Update the relevant `Docs/*.md` page(s) to match the new/changed API.
- API and command details belong in `Docs/` (and, for the CLI, in each command's `Usage`, `Forms` and `Examples`), never in the agent skills. The skills teach principles, the mental model and where to find documentation; update one only when a principle, a workflow or a documentation location changes. See [Agent Skills](#agent-skills).
- When the change touches the CLI/REPL or the DSL, update the CLI (`cli/`) and its doc [Docs/cli-dsl.md](Docs/cli-dsl.md).

**When the change is visible to someone using the library or the CLI:**
- Add an entry under `## Unreleased` in **both** [CHANGELOG.md](CHANGELOG.md) and [CHANGELOG_TW.md](CHANGELOG_TW.md), in the same change. Under the OpenSpec workflow this belongs in the change's own `tasks.md` — never as a follow-up, and never written from memory at release time.
- Group entries under the same package headings the release notes use (`### Core`, `### CLI`, `` ### `stats` ``, …), one level deeper than a release note so that promoting `###` to `##` at release time produces the release note as-is. Append to the end of a package section rather than the top.
- Mark breaking changes the way past release notes do.
- Skip the entry when nothing user-visible changed: internal refactors, tests, formatting, assets, dependency bumps with no behavioral effect, and OpenSpec bookkeeping.
- At release time, rename `## Unreleased` to the version number in both files and open a fresh empty `## Unreleased` above it.
- Also at release time, bump `Version` in [version.go](version.go) in the same change — the startup banner and the CLI `version` command read it, and it does not follow the changelog on its own (v0.3.1 nearly shipped with the banner still saying v0.3.0; it was caught at the PR, not by any check).
- Archive every OpenSpec change whose work is in the release on `dev` before the release branch is cut. `openspec/changes/` on the release branch holds only work that is not in it.
- Release only when every CI check is green, on the release PR and on the `main` merge commit it produces. A red check blocks the release until it is fixed or excluded by its own change, including one that fails on every branch.
- A release is always named with both the series name and the version number — "Huashan v0.3.1", never a bare "v0.3.1" — in the GitHub Release title and anywhere else the release is announced. The series name comes from `VersionName` in [version.go](version.go) and changes only when a new series starts.

Keep the English ([README.md](README.md), [CHANGELOG.md](CHANGELOG.md), `Docs/`) and Traditional-Chinese ([README_TW.md](README_TW.md), [CHANGELOG_TW.md](CHANGELOG_TW.md)) docs in lockstep — never update one side without the other.

## Agent Skills

[skills/insyra/](skills/insyra/) — for AI agents writing Go code with Insyra.  
[skills/use-insyra-cli/](skills/use-insyra-cli/) — for AI agents working through the CLI, the REPL, `.isr` scripts or the Go DSL.

A skill is installed into an agent's environment and outlives the version it came from, so it teaches what does not change between releases: when to reach for Insyra, how to think about it, the conventions that hold across it, how to verify a result, and how to find the exact API for the version in use (the module's own `Docs/`, `go doc`, `insyra help`). It does not list functions or commands, and it has no reference files that repeat `Docs/`. A detail a skill used to carry lives in `Docs/`; before removing anything from a skill, make sure `Docs/` holds it (`agent-skills` spec).

## Follow-ups

Out-of-scope issues discovered during development, waiting for a decision. Delete an entry once it is resolved.

### [2026-09-28] — remove `Difference`, `MovingAverage`, `MovingStdev` and `WeightedMovingAverage` one release after they were deprecated
- **Where**: `datalist.go`, their lines in `IDataList` (`interfaces.go`), `cli/commands/timeseries.go` (`movavg`, `diff`), and the tests that call them (`datalist_legacy_transforms_test.go`, `datalist_legacy_smoothing_test.go`, `datalist_deprecated_test.go`, `datalist_test.go`, `datalist_numeric_test.go`, `chainable_nil_test.go`, `instance_error_test.go`, `error_philosophy_test.go`)
- **What**: `deprecate-look-alike-window-methods` deprecated the four by the owner's ruling on #222 (2026-09-28), keeping their behaviour for one release instead of turning them into aliases of `Diff(1)` and `Rolling(...)`, which would have changed their results under the same name. The CLI's `movavg` and `diff` still call them.
- **Suggestion**: delete the four and their `IDataList` lines, drop their rows from the characterization tests and from the table in `Docs/DataList.md`, and add a BREAKING changelog entry, in the same release as the other Deprecated removals. Decide first what `movavg` and `diff` become: reproduce their current output from `Rolling(...).Mean()` and `Diff(1)` by dropping the leading `nil` positions, or deprecate them in favour of `rolling … mean` and `diffn` the way `fillnan` points at `fillna`.
- **Status**: pending

### [2026-09-28] — five `DataList` transforms fail in a shape the others do not
- **Where**: `datalist.go` `Normalize` and `Rank`; `datalist_window.go` `Shift`; `datalist_sampling.go` `Sample`, `SampleFrac` and `Shuffle`
- **What**: the `chainable-never-nil` spec says a method returning `*DataList` that fails returns an empty list carrying the error, recorded on the receiver too. Classifying every `DataList` method for #218 on 2026-09-28 found five that do not, by reading the code (not run): `Sample`, `SampleFrac` and `Shuffle` return a bare `NewDataList()` on every failure, so the result's `Err()` is nil; `Rank` and `Shift` given two optional values return the receiver itself, so `r := dl.Shift(1, a, b)` hands back the unshifted list as if it were the shifted one; and `Normalize`, which changes the list in place, returns an empty list when it fails where every other in-place method returns the list, so `dl = dl.Normalize()` throws the data away on a failure. `Docs/DataList.md` states the `Normalize` behaviour meanwhile.
- **Suggestion**: `failedResult()` for the first five and `return dl` for `Normalize`'s failures. Each changes what a failed call returns, so it belongs with a breaking batch on this line.
- **Status**: pending

### [2026-09-28] — remove `SqrtRat`, `PowRat`, `SortTimes` and `F64orRat` one release after they were deprecated
- **Where**: `utils.go`; the constraint behind `F64orRat` in `internal/utils/utils.go`
- **What**: `core-utils-cleanup` deprecated the four (#214, K-15): none has anything to do with data tables and nothing in the module calls them. They stay one release, marked Deprecated, with `SqrtRat` and `PowRat` fixed so they no longer panic and `PowRat` computes negative powers.
- **Suggestion**: delete them and the internal `F64orRat` in the same release as the other Deprecated removals, with `TestSqrtRatNegativeReturnsNil`, `TestPowRatNegativeExponent` and `TestSortTimes`, their sections in `Docs/utils.md`, and a BREAKING changelog entry.
- **Status**: pending

### [2026-09-28] — remove the deprecated `DataTable` filter and header names one release after they were deprecated
- **Where**: `datatable_filters.go`, `datatable_colname.go`, `interfaces.go`
- **What**: under the one-name rule, `filter-rows-where` deprecated `FilterByCustomElement` (#226, T-13), which returns exactly what `Filter` returns, and `datatable-slicing` deprecated the ten `FilterColsByColIndex…`/`FilterRowsByRowIndex…` methods in favour of `SliceCols`/`SliceRows`, and `Headers`/`SetHeaders` in favour of `ColNames`/`SetColNames` (#231, T-21).
- **Suggestion**: delete the thirteen methods, their lines in `IDataTable`, the tests that pin their old meaning (`TestFilterByCustomElementEqualsFilter`, `TestDeprecatedIndexFiltersMatchSlices`, `TestDeprecatedColIndexFiltersDoNotPanicPastTheEnd`, `TestHeadersAreColNames`) and their sections in `Docs/DataTable.md` in the same release as the other Deprecated removals, with a BREAKING changelog entry.
- **Status**: pending

### [2026-09-28] — on MySQL, a failed `ToSQL` can still leave a new table or new columns behind
- **Where**: `datatable_to_sql.go` `saveRowsToDB`, the `!exists` branch (`CREATE TABLE`) and the append branch (`ALTER TABLE … ADD COLUMN`)
- **What**: MySQL commits every DDL statement on its own and ends the transaction it is in (MySQL 8.4 Reference Manual, "Statements That Cause an Implicit Commit"), so the `INSERT`s after a `CREATE TABLE` or an `ALTER TABLE` commit batch by batch. By the manual, not run against a MySQL server: a failed write leaves the table created because it did not exist, or the columns added in append mode, and the batches written before the failure. With `SQLActionIfTableExistsFail`, the next attempt then fails because the table exists. `sql-replace-keeps-old-table` fixed Replace, which lost the old table, and `Docs/DataTable.md` now states these two cases.
- **Suggestion**: create a missing table the way Replace now works, through a staging table renamed to the target (a rename to a taken name fails, so a table created meanwhile is not overwritten). Added columns cannot be taken back once the ALTER commits; decide whether to check the whole write before the ALTER, or accept and document it. Verify both against a real MySQL before changing them, and add an integration test gated on a MySQL DSN in the environment: the Replace path of `sql-replace-keeps-old-table` was reasoned from the MySQL manual and tested only on SQLite through the same code, and whether TiDB, which gorm also names `mysql`, runs a multi-table `RENAME TABLE` atomically is not known.
- **Status**: pending

### [2026-09-28] — `TestMultiDeviceDispatchMergesRangesAndReportsPlacement` fails on Windows when a shard finishes within one clock tick
- **Where**: `accel/multi_device_test.go`, the `assignment.WallTime <= 0` check; `accel/exact.go` sets `WallTime` from `time.Since`
- **What**: the Test workflow on 5226d0e5, a commit that only moved OpenSpec files, failed on `windows-latest` alone with `assignment did not report successful placement and wall time: … WallTime:0`, and a re-run of the same job was requested. The same test passed on Windows for the four commits pushed before it and on macOS and Linux every time. The likely cause, not verified on a Windows machine, is that the test's fake backend finishes a shard within one tick of the Windows clock, so `time.Since` returns 0.
- **Suggestion**: assert `WallTime >= 0` and check placement through `FallbackReason` alone, or give the fake backend a measurable delay. Whether `WallTime` 0 is an acceptable report for a real device is the question to settle first.
- **Status**: pending

### [2026-09-28] — a two-sided t or z p-value in the far tail rounds to 0, below the one-sided one
- **Where**: `stats/distutil.go` `tTwoTailedPValue` and the `TwoSided` arm of `zPValue`, both `2 * (1 - F(|x|))`; `stats/ztest.go`'s `sigma` checks
- **What**: the subtraction loses every digit once the p-value falls near 1e-16, so the two-sided p-value comes out 0 or too small, while the one-sided p-values `stats-test-settings` added use `F(-t)` and stay exact. Measured on 2026-09-28 with 40 observations cycling through 10.0 to 10.4: against `mu` 9.8, t = 17.66, the two-sided p-value is 0 where R's `t.test` gives 3.31e-20, and `Greater` gives 1.656e-20, matching R; against 9.9 the two-sided value is 4.44e-16 where R gives 5.10e-16. A two-sided p-value below the one-sided one is impossible. Separately, `SingleSampleZTest(x, 10, math.NaN())` returns a `NaN` statistic, p-value and interval with a nil error, because `sigma <= 0` is false for `NaN`; `TwoSampleZTest`'s sigmas and both tests' `mu` are not checked either.
- **Suggestion**: compute the two-sided value as `2 * F(-|x|)`, which is R's formula and moves a p-value above about 1e-15 only in its last digits, and refuse a `NaN` or infinite `sigma` or `mu`. Both change what a call returns, so they go in their own change with a changelog entry.
- **Status**: pending

### [2026-09-28] — the regression functions still take their settings three ways
- **Where**: `stats/regression_glm.go` `GLM`, `stats/regression_logistic.go` `LogisticRegression` and `LogisticRegressionWithOptions`, `stats/regression_poisson.go` `PoissonRegression` and `PoissonRegressionWithOptions`, and the `resolveConfidenceLevel` call in each
- **What**: `stats-test-settings` gave the hypothesis tests one form: settings in an optional trailing `opts ...XxxOptions`, several samples in one slice, and an out-of-range confidence level refused. The regression functions were outside #240 and still differ in three ways. `GLM(opts, y, xs...)` takes its options first and required, because its predictors are variadic. Logistic and Poisson regression each have a second name, `...WithOptions`, for the same reason, which the one-name rule of #211 does not allow. And their `ConfidenceLevel` falls back to 0.95 for any value outside (0, 1): measured on 2026-09-28, `LogisticRegressionWithOptions` with `ConfidenceLevel` 1.5 or `NaN` returns `ConfidenceLevel` 0.95 and the 95% interval `[-0.281, 2.624]` for the slope, with a nil error, where the tests now return an error.
- **Suggestion**: take the predictors as a `[]insyra.IDataList` followed by `opts ...XxxOptions`, keep one name per regression with the `...WithOptions` names as Deprecated wrappers for one release, and read the level through `resolveTestSettings`'s rule. Every regression call changes, so it is its own change; the reason for the slice is the one `stats-test-settings` gives for the k-sample tests.
- **Status**: pending

### [2026-09-28] — cutting a tree accepts a node merged twice
- **Where**: `stats/internal/clustering/cluster.go` `validateTree`
- **What**: `validateTree` checks that every merge joins a leaf or an earlier merge, but not that each is used once. Measured on 2026-09-28 with three labels: merges `{-1,-2},{-1,-2}`, `{-1,-2},{1,1}` or `{-1,-1},{-2,1}` all pass, and `CutTreeByK(tree, 1)` returns `[1 1 2]`, two clusters where one was asked for, with no error. A hand-edited `state.json` reaches `cutree` this way. `dev` has the same check and records the same follow-up.
- **Suggestion**: track which leaves and merges have been used and refuse an id seen a second time, a merge joining something with itself included.
- **Status**: pending

### [2026-09-28] — a rebuilt sheet leaves its old comments, drawing and table parts in the file
- **Where**: `internal/excelsheet/replace.go` `Replace`, through excelize's `DeleteSheet`; reached by `csvxl.AppendCsvToExcel` and `DataTable.ToExcel` with `SheetExistsReplace`
- **What**: `DeleteSheet` removes the worksheet and its relationships but not the parts they pointed to. Measured on 2026-09-28: after replacing a sheet that had a comment and a table, `xl/comments1.xml`, `xl/drawings/vmlDrawing1.vml` and `xl/tables/table1.xml` are still in the saved file and in `[Content_Types].xml`, referenced by nothing, so the old comment text is still inside the file; excelize also refuses a new table under the old table's name (`the same name table already exists`). Not checked in Excel or LibreOffice, neither of which was available. `dev` records the same follow-up.
- **Suggestion**: open such a file in Excel and LibreOffice first. If either complains, or if leaving the old comment text in the file matters, remove the parts the old sheet's relationships pointed to before deleting it.
- **Status**: pending

### [2026-09-26] — CLI arguments a count cannot catch
- **Where**: `cli/commands/` — `setrownames.go`, `show.go`, `fetch.go`, `knn.go`, `fillna.go`, `config.go`
- **What**: `cli-declared-arg-limits` made every command refuse an argument past its declared count. Auditing the 117 commands for it turned up six places where the count is right and an argument is still accepted and dropped, measured on 2026-09-26: `setrownames t r1 … r6` on a four-row table reports success and drops `r5` and `r6`; `show` on a scaler variable or on `accel.devices` ignores `<start> <end>`; `fetch yahoo AAPL calendar extra` ignores `extra`, because the count is per source and only some yahoo methods take an argument; `knn_neighbors` accepts `weighting`, which its Usage does not list and nothing reads; `fillna x mean limit 1` accepts `limit`, and `extrapolate`, for strategies that do not use them; and `config log-level` fails with the usage `config [key] [value]`, which says a key alone is allowed. `exit` being a no-op in one-shot and scripts is tracked separately in #328.
- **Suggestion**: each is a few lines in its own command: refuse the extra names, the range, the token, or the option, and make `config`'s Usage say `config [<key> <value>]` unless reading one key is wanted. They are independent, so they can go in one small change.
- **Status**: pending

### [2026-09-26] — remove the underscore reader and writer names one release after their replacements
- **Where**: `read.go` (`ReadCSV_File`, `ReadCSV_FileWithOptions`, `ReadCSV_String`, `ReadCSV_StringWithOptions`, `ReadJSON_File`), `datatable_csv.go` (`ToCSVWithOptions`), `datatable_json.go` (`ToJSON_Bytes`, `ToJSON_String`) and the two JSON methods in `IDataTable`
- **What**: `read-write-names` gave each reader and writer one name taking an optional options struct, by the owner's rulings on #213. The old names stay one release as Deprecated wrappers that keep their old meaning.
- **Suggestion**: delete them, and `CSVWriteOptions.SanitizeFormulas` (a no-op since `csv-formula-guard-by-default` made the guard the default), in the same release as `Slice2DToDataTable` and `ScalerParams.PassThrough`, with the tests that pin their old meaning (`TestDeprecatedReadWriteNamesKeepTheirMeaning`, `TestReadCSV_StringLegacyNoHeaderMatchesNoHeaderRow`) and a BREAKING changelog entry.
- **Status**: pending

### [2026-09-26] — remove `ScalerParams.PassThrough` one release after it went dead
- **Where**: `datatable_scale.go` (`ScalerParams.PassThrough`)
- **What**: `imputer-refuses-unfillable-columns` made `SimpleImputer.Fit` refuse a selected column its mean or median cannot fill, by the owner's ruling on #213, so nothing sets the field any more. It stays one release, always false and Deprecated.
- **Suggestion**: delete it in the same release as `Slice2DToDataTable`, with a BREAKING changelog entry.
- **Status**: pending

### [2026-09-25] — remove `Slice2DToDataTable` one release after `ReadSlice2D` became its one name
- **Where**: `read.go` (`Slice2DToDataTable`)
- **What**: `exported-functions-are-functions` made `ReadSlice2D` the function that turns a 2D slice into a DataTable and left `Slice2DToDataTable` as a Deprecated wrapper, by the owner's ruling on #211 that each function has one name and the `Read*` family keeps it.
- **Suggestion**: in the first release after the one that ships `exported-functions-are-functions`, delete `Slice2DToDataTable` and `TestSlice2DToDataTableMatchesReadSlice2D`, drop the deprecated-spelling note from `Docs/DataTable.md`, and add a BREAKING changelog entry. Do it in the same release as the `SetDontPanic` removal if they coincide.
- **Status**: pending

### [2026-09-27] — outside the hypothesis tests, `stats` still numbers some positions from zero, and two paths let `NaN` through
- **Where**: the `predictor %d` label in `stats/regression_shared.go`; the `(index %d)` checks in `stats/regression.go`, `stats/regression_glm.go`, `stats/regression_poisson.go` and `stats/glm_irls.go`; `row %d column %d` and `column %d` in `stats/factor_analysis.go`; `columns %d and %d` in `stats/correlation.go`; `stats/pca.go`, which also reads cells with `ToFloat64Safe` alone; `stats/knn.go` `numericVectorFromDataList`
- **What**: every hypothesis test now counts the list and the position from one (`hypothesis-tests-refuse-non-finite`). Elsewhere the old numbering is still there. Measured on 2026-09-27: `LinearRegression(y, x1, x2)` with a blank at row 3 of `x2` says `predictor 1 contains a non-numeric value at row 3: <nil>`, `LogarithmicRegression` with `x = -2` at row 2 says `(index 1)`, and `FactorAnalysis` with text at row 2 of its second column says `non-numeric value at row 1 column 1: x`. The correlation and PCA column numbers are zero-based in the code and were not measured. Separately, `PCA` and `KNNRegress`'s targets check only that a cell converts, so a `NaN` or `+Inf` reaches the result with a nil error: `PCA` returns `NaN` eigenvalues, and `KNNRegress` predicts `NaN` or `+Inf`. `Docs/stats.md` states the `PCA` and `KNNRegress` behaviour.
- **Suggestion**: number every position from one, and read `PCA` and `numericVectorFromDataList` through `numericValues`. Both change what callers see, the error text in one case and an error where a result came back in the other, so decide first whether they go on 0.3.x the way the hypothesis tests did. That change rested on the documentation already saying such a value is never used.
- **Status**: pending

### [2026-09-27] — two one-shot commands on one environment lose one command's variables
- **Where**: `cli/env/state.go` `SaveVariables` (the save every command ends with) and `RestoreVariables` (the load it starts with)
- **What**: each command reads the whole `state.json`, runs, and writes the whole file back through the same `state.json.tmp` path. Two `insyra` commands against one environment at the same time each save their own copy, so the later rename wins and the variables the other command created are gone; if both write the temporary file at once, one rename can also fail. Found by the review of `cli-env-typed-state` on 2026-09-27, by reading the code; not reproduced. It predates that change, which did not alter the read-modify-write.
- **Suggestion**: a per-environment lock file held from restore to save, or a unique temporary name per writer plus a check that the file has not changed since it was read. The lock serialises commands, which a user running commands in parallel may not expect, so decide which.
- **Status**: pending

### [2026-09-27] — an unreadable `state.json` is overwritten by the next command
- **Where**: `cli/root.go` `openEnvironment`, `cli/repl/repl.go` `Start`, `cli/repl/api.go` `NewDSLSession`, and `env open` in `cli/commands/env.go`
- **What**: when `RestoreVariables` fails, the first three start from an empty variable map without saying so, and the save after the next command writes that map over the file, so every variable in the environment is lost. Measured on 2026-09-27 with the CLI built from `dev` at 550cf94: after `newdl 1 2 3 as x`, truncating `state.json` mid-object and running `newdl 9 as y` left a `state.json` holding only `y`. `env open` fails differently, going by its code: when the opened environment's state cannot be read, it keeps the previous environment's variables, and the next save writes them into the environment just opened. `cli-env-typed-state` does not make this more likely, because a file it writes always decodes, but it does not change it.
- **Suggestion**: stop before running a command against an environment whose state could not be read, naming the file and the error, or move the unreadable file aside before saving over it. Either changes what a command does in a damaged environment, so decide which first.
- **Status**: pending

### [2026-09-27] — Auto reads an ISO-2022-KR or ISO-2022-CN file as UTF-8
- **Where**: `utils.go` `DetectEncoding`, whose UTF-8 check runs before chardet is asked; `internal/csv/decoder.go` `decoders`
- **What**: both encodings use only 7-bit bytes, so `DetectEncoding` accepts such a file as UTF-8 before chardet, which knows both, sees it. Measured on 2026-09-27: a 30-line Korean CSV written in ISO-2022-KR is detected as `utf-8`, and `csvxl.ReadCsvToString` returns its `ESC $ ) C` header and shift codes inside the cells with a nil error. x/text has no decoder for either, so naming the encoding fails with the unsupported-encoding error. `Docs/csvxl.md` states this.
- **Suggestion**: look for the ISO-2022 designator escape before the UTF-8 check and refuse the file with the unsupported-encoding error, the way IBM420 and IBM424 are refused. Decoding them would need a decoder x/text does not ship; both are rare enough in CSV files that the refusal is enough.
- **Status**: pending

### [2026-09-28] — two nested values whose strings contain the separators count as one
- **Where**: `cell_identity.go`, the string arm of `writeCellValue`
- **What**: a string inside a nested value is written without escaping the characters the encoding uses as separators, so different values can encode alike. Measured on 2026-09-28: a list holding `[]any{"a,s:b"}` and `[]any{"a", "b"}` reports `Count([]any{"a", "b"})` 2 and a `Counter` of one entry; grouping, pivoting and merging key on the same encoding. Present on `origin/0.4` before the dev merge. Separately, an array or a struct shared through interfaces is still encoded once per path, so a hand-built value of `[2]any{x, x}` nested 20 levels deep takes about 200 ms and 31 levels about a minute; slices and maps are memoized.
- **Suggestion**: write each string with a length prefix (`s<len>:`), which makes any content unambiguous, and give arrays and structs the same work bound slices have. The first changes every key a nested string produces, so group keys and `Counter` keys for such values change once.
- **Status**: pending

### [2026-09-28] — what the typed `state.json` codec still accepts badly
- **Where**: `cli/env/state.go`, `cli/env/variable_codec.go`, `cli/env/cell_codec.go`
- **What**: found by the review of the dev merge, each measured on 2026-09-28. (1) A variable, column, row or table name that is not valid UTF-8 (a Big5 `.isr` script) is written with U+FFFD in its place, so two such column names come back as `��` and `��_1`, and two such variable names collide in the file and one is lost without `SaveVariables` reporting it. Cell contents are safe, because non-UTF-8 text in a cell is base64-encoded. (2) Decoding has no depth limit while encoding refuses a cell nested more than 64 levels, so a table read from a hand-edited or imported file is dropped, with a warning, at the next save. (3) Restoring a table whose file lists thousands of columns under one name is cubic in that count (2,000 columns: 8.4 s), and every one-shot command restores the whole environment. (4) A variable that could not be decoded is kept, but its reason is dropped: `vars` prints the internal type `env.unreadableVariable` and a command using it says only that it is not a DataTable.
- **Suggestion**: (1) report a name that is not valid UTF-8 as unsaved, or encode names the way cells are; (2) apply the encoder's depth limit when decoding and keep such a variable unreadable; (3) refuse duplicate column names when decoding, since no table the library builds has them; (4) keep the decode error on the unreadable variable and print it.
- **Status**: pending

### [2026-09-28] — `db connect` history masking leaves part of some passwords
- **Where**: `cli/commands/db_conn.go` `maskKVPasswords`, `maskDSNPassword`
- **What**: measured on 2026-09-28. In a key=value DSN, an unquoted value ends at a space for libpq and pgx, but the mask stops at `;` or `"`, so `password=ab;cd port=5` is written to `history.txt` as `password=***;cd port=5`. In a URL or a MySQL native DSN the mask takes the first `@`, while the drivers take the last, so `u:p@ss@h` is written as `u:***@ss@h`. The history file is private to the user (0600), but `env export` writes it into a file anyone may be sent. The first gap came with the masking dev added (677f6c53); the second was there before.
- **Suggestion**: choose the end of a value by dialect, `;` only for the ODBC-style `sqlserver` form, and take the last `@` before the host in the URL and MySQL forms. Add each measured case to the masking tests.
- **Status**: pending

### [2026-09-28] — a fitted scaler saved by dev cannot be read on this line
- **Where**: `datatable_scale.go` `scalerRefJSON`, `decodeScalerRef`
- **What**: dev writes a scaler's column reference as a plain string (`"ref":"a"`), the selector as the caller gave it. This line writes an object (`{"name":"a"}` or `{"position":0}`) and refuses a string: `cannot unmarshal string into … scalerRefJSON`. In the CLI such a variable is kept but unusable. The scaler JSON is unreleased on both lines, so nothing breaks today; it does if a 0.3.x release ships it before 0.4.
- **Suggestion**: if a 0.3.x release ships scaler JSON first, read a plain string here the way that release resolved it, or mark the change BREAKING in 0.4's release note. Decide before the next 0.3.x release.
- **Status**: pending

### [2026-09-28] — three smaller things the dev merge's review measured
- **Where**: `internal/ccl/map_context.go` `MapContext.GetColData`; `stats/asdatalist.go`; `stats/nonparam_mwu.go` (the Hodges-Lehmann estimate)
- **What**: measured on 2026-09-28, all present before the merge. (1) `engine/ccl`'s `MapContext` hands a registered aggregate the caller's own slice; an aggregate that sorts it in place changes the caller's map and the rest of the expression (`ZZSORTFIRST(A) + A.0` gives 2 instead of 4). `dataTableContext` copies. (2) `asDataList` turns a nil `*DataList` into an empty list but not a nil value of a type that embeds one, so `TwoSampleTTest(x, (*W)(nil))` still panics. (3) `MannWhitneyU` allocates all n1·n2 pairwise differences for its estimate; two groups of six million values panic with `makeslice: cap out of range`, and smaller ones run out of memory first.
- **Suggestion**: (1) copy in the evaluator before it calls an aggregate or sequence function, so no context has to; (2) recover in `asDataList`, or look through an embedded pointer with reflection; (3) compute the median of the differences with a selection algorithm instead of materialising them.
- **Status**: pending

### [2026-09-28] — `nn` allocates from a size argument before anything checks it is sane
- **Where**: `nn/edge_sum.go` `NewEdgeTopology`; `nn` tensor constructors given a shape
- **What**: `NewEdgeTopology(nodes, nil, nil)` accepts any `nodes` up to `MaxInt32` and allocates four `int32` arrays of that length before any edge exists, about 32 GiB at the limit; `NewTensor` with a huge shape is the same kind of call. Running out of memory ends the process and cannot be recovered, against this line's rule that the library never ends the program. Found by the dev merge's review by reading the code; not run, to keep the machine up.
- **Suggestion**: decide a policy for allocations sized by an argument: a documented cap with an error above it, or allocating in proportion to the data actually given (the edges) rather than to a count. Either applies to more of `nn` than this one constructor.
- **Status**: pending

### [2026-09-27] — `AtomicDoAll` inside `AtomicDo` runs the other instances unlocked, so `AppendCols` inside a callback races
- **Where**: `internal/core/atomic.go` (the inline trust-zone path of `AtomicDoN`), reached by any method that calls `AtomicDoAll` while its receiver is held, such as `DataTable.AppendCols`
- **What**: 0.4 resolved an AB-BA deadlock (api-review IN-1) by running `AtomicDoAll` inline, without locking the other instances, when it is called inside an `AtomicDo`; `core-multilock-reentry` requires that. The cost is a data race. Measured on 2026-09-27 with `go test -race`: `dt.AtomicDo(func(t *DataTable) { t.AppendCols(col) })` in one goroutine and `col.Append(i)` in another report `DATA RACE` in three of three runs, the write at `datalist.go:108`. dev instead locks the other instances again (ddde3d45), which reopens the deadlock. The Key Conventions above tell callers to use `AtomicDoAll` from the outermost level, but a library method such as `AppendCols` calls it internally, so a caller who follows that rule and only calls `AppendCols` inside its own `AtomicDo` still runs unlocked without knowing.
- **Suggestion**: this changes a specified concurrency contract, so it needs the owner. Three ways out: keep the inline path and document that `AtomicDoAll` inside a callback does not lock (and fix the Key Conventions line); lock the others in the global order with a try-lock and fail the call with an error when one is busy; or make methods such as `AppendCols` copy the other instance under its own lock before entering the receiver's callback, so no library method nests at all.
- **Status**: pending (owner decision)

### [2026-09-27] — three ways a value on its way out still panics
- **Where**: `datatable_json.go` `jsonCell`; `parquet/ccl.go` `buildArrowRecord` and `buildArrowArray`; `plot/save_chart.go` `SaveHTML` and `SavePNG`
- **What**: `jsonCell` asks a cell for `fmt.Stringer` before it checks for a nil pointer, the defect 91e39e7c fixed for `Show`. Measured on 2026-09-27: `ToJSONString` on a table holding a nil `*T`, where `T` has a value-receiver `String`, panics with `value method … called using nil *T pointer`. Separately, `parquet.ApplyCCL` rebuilds every column with the original schema's type but builds only `INT64`, `FLOAT64`, `BOOL` and `STRING` arrays, so any other column type panics in `array.NewRecord`: measured on 2026-09-27, a file `parquet.Write` wrote from a table with a `time.Time` column panics with `arrow/array: column "c" type mismatch: got=utf8, want=timestamp[ns, tz=UTC]`, and a `[]byte` or `Date32` column fails the same way. The original file is left alone; the caller gets a panic instead of an error, against this line's rule that the library never ends the program. Third, every `plot.Create...` function returns `nil` when it cannot build a chart, and a `nil` chart still satisfies `Renderable`: measured on 2026-09-27, `plot.SaveHTML((*charts.Bar)(nil), path)` panics with a nil pointer dereference, and `SavePNG` does the same. `Docs/plot.md` tells callers to check for `nil` meanwhile.
- **Suggestion**: in `jsonCell`, check for a nil pointer before `fmt.Stringer`, as `internal/utils` now does. In `ApplyCCL`, build each untouched column from its original array, or refuse a file with a column type the builder does not handle with an error before anything is read. In `SaveHTML` and `SavePNG`, return an error for a nil chart, the way `gplot.SaveChart` already does, and change the note in `Docs/plot.md`.
- **Status**: pending

### [2026-09-27] — smaller defects the dev merge's review found on this line
- **Where**: `plot/boxplot.go` `CreateBoxPlot`; `csvxl/convert.go` `saveSheetAsCsv`; `internal/ccl/stdlib_string.go` `REPEAT`; `ccl.go` `applyCCLOnDataTable`
- **What**: measured on 2026-09-27. (1) `CreateBoxPlot` drops a series whose `Data` is empty with no warning, and given only such a series returns nil with the warning "No series provided in BoxPlotConfig.Series", although one was. (2) `saveSheetAsCsv` returns without its `cleanup()` when `GetRowVisible` fails (line 345), leaving the hidden `.<name>.*.tmp` file and its descriptor; every other error path cleans up. excelize only fails there for a row or sheet `GetRows` has just read, so it is hard to reach. (3) `REPEAT('', 100000000)` is refused with "result would exceed 67108864 bytes" although its result is empty. (4) `ExecuteCCL` evaluates a row-invariant aggregate on every row: on a 10,000-row column `NEW('C') = A / SUM(A)` took 535 ms where `AddColUsingCCL` with `A / SUM(A)` took 0.5 ms. Since the dev merge each of those evaluations also copies the column (`GetColData`), about 1.8 times slower again on 20,000 rows; folding removes both. `Docs/CCL.md` and the `ccl-performance` spec state the limit.
- **Suggestion**: (1) warn naming each dropped series and say "no series has any data"; (2) call `cleanup()` on that path; (3) return an empty string before the size check when the input is empty; (4) fold aggregates in statement mode the way expression mode does, measured before and after. Each is small and independent.
- **Status**: pending

### [2026-09-27] — CLI corners the skill audit measured
- **Where**: `cli/commands/timeseries.go` (`rolling`); `cli/commands/db.go` `db tables`; `cli/commands/run.go` against `cli/repl/repl.go`'s tokenizer; `cli/commands/hypothesis.go`; the `error:` line printer
- **What**: measured on 2026-09-27 with the CLI built from this branch. (1) `rolling … minobs` larger than the window logs an error, stores an empty DataList, prints `saved as` and exits 0, where `Docs/cli-dsl.md` says a command that cannot do what it was asked fails. (2) `db tables <conn> schema x` on SQLite accepts and ignores `schema`, against the rule that an argument a command does not use is refused. (3) `run` treats a backslash as an escape only before a quote or a backslash, while the REPL and `Session.ExecuteFile` treat it as escaping any character, so a Windows path in a `.isr` file reads differently under the two, although `Docs/cli-dsl.md` says they differ only in error handling. (4) `help chisq`'s example names its input `counts`, while `chisq gof` wants raw observations. (5) With `NO_COLOR=1` set, the `error:` line is still printed in colour.
- **Suggestion**: (1) return the error so the command fails; (2) refuse `schema` on SQLite or document that it is accepted and has no effect; (3) give `ExecuteFile` the `run` tokenizer, or say how they differ; (4) rename the example's variable; (5) route the error line through the same colour check as the rest of the output. Each is small and independent.
- **Status**: pending

### [2026-09-27] — 36 main specs fail `openspec validate --specs --strict`
- **Where**: `openspec/specs/*/spec.md`, mostly the `## Purpose` section
- **What**: on 2026-09-28 the strict run reports 36 failures of 136 specs on this branch, among them the seven `accel-*` specs, the five `ml-*` specs, `nn-inference`, `changelog`, `cli-entry`, `command-registry`, `core-multilock-reentry` and `error-philosophy`. dev recorded 26 of its 101 on 2026-09-14. Nothing in CI runs the command.
- **Suggestion**: write each Purpose from its requirements as its own change, then add the strict run to the lint workflow so the count cannot climb again.
- **Status**: pending

### [2026-09-25] — device MatMul's bit-parity rests on behaviour WGSL does not promise
- **Where**: `accel/internal/wgpu/matmul.go` (`matmulWGSL`), `accel/nn_matmul.go`, and the default-on hook in `nn/device_matmul_wiring.go`
- **What**: `ENG.md` now defines a device float32 result as the correctly rounded value of the exact operation, computed in integers, because WGSL lets an implementation contract, reassociate and flush subnormals. Device MatMul predates that rule: it accumulates `acc + a*b` in `f32` and matches the CPU only because Metal and Go on arm64 both fuse (asserted with `==` on the M3). The same measurement for `EdgeSum` on 2026-09-25 showed Metal fusing exactly like arm64, so today's parity is real, but a conforming implementation that reassociated the loop or flushed a subnormal would break it, and amd64's CPU, which does not fuse, already disagrees with the device.
- **Suggestion**: move MatMul to the exact rule — each output the correctly rounded exact dot product — on both CPU and device. That changes `nn.MatMul`'s results, which are released, so it waits for the owner; it also costs CPU time that has to be measured against M19's all-core baseline first.
- **Status**: pending

### [2026-09-20] — `TestSequentialFitMNISTConvergence` still pins numbers recorded on one machine
- **Where**: [nn/fit_mnist_test.go](nn/fit_mnist_test.go), the mean-loss and accuracy assertions
- **What**: Fit's sugar-changes-nothing proof compares against the transcribed `0.347310`, `0.165883` and `0.9547` instead of against the hand-written loop it claims to reproduce. Its sibling `TestSequentialMNISTConvergence` failed exactly that way on `ubuntu-latest` on 2026-09-20 — `0.163855` recorded on arm64 against `0.163840` measured on amd64, while the two code paths still agreed with each other — and now runs the hand loop in the same process (`mnist-proof-compares-runs`). Fit's numbers happen to hold on both platforms measured so far, so nothing is red today.
- **Suggestion**: the same treatment. Run the documented hand loop inside the test and compare the two curves, keeping only bounds any platform meets. It costs one more MNIST run, about 30 seconds in CI. Worth doing the next time `nn`'s training path is touched, or sooner if a third platform disagrees.
- **Status**: pending

### [2026-09-19] — `BartlettTest` panics when the group variances come out equal
- **Where**: `stats/ftest.go` `BartlettTest`, through `stats/distutil.go` `chiSquaredPValue`
- **What**: Bartlett's `T` is a difference of logs, so for groups whose variances agree it lands at or just below zero by rounding. `chiSquaredPValue` passes it straight to `distuv.ChiSquared{K: df}.CDF`, which panics with `cephes: parameter out of bounds` for any negative argument. Measured on this line on 2026-09-27: `chiSquaredPValue(-1e-16, 1)` panics while `chiSquaredPValue(0, 1)` returns 1. Whether two identical groups panic therefore depends on the values: `{0.1, 0.2, 0.3}` twice returns statistic 0 and p 1, while `{-0.32329881667362437, -0.319580921043235, 0.9093452052941211, 0.9788142136474671}` twice panics. `df <= 0` panics the same way (`chiSquaredPValue(1, 0)`), which `FriedmanTest`, `KruskalWallis`, `ChiSquareTest`, `PartialCorrelation` and `FactorAnalysis` also reach. `KruskalWallis` and `FriedmanTest` reach it on finite data too, when every value is equal: `KruskalWallis` on four all-zero groups of sizes 24, 4, 27 and 11, and `FriedmanTest` on 21 subjects with 7 conditions each, all zero, both panic with `cephes: parameter out of bounds`.
- **Suggestion**: clamp in `chiSquaredPValue`. A negative `chi2` is a rounding artefact of a statistic that is zero, so returning 1 for it is the right answer, and a non-positive `df` should return NaN the way `tQuantile` already does. That changes a panic into a value for every caller at once, which needs deciding before it is done: it is a behaviour change, though the old behaviour was a panic.
- **Status**: pending

### [2026-09-17] — psych 2.6.5's `faRotations` tie-break picks a start that did not tie, and nobody upstream has been told
- **Where**: upstream `psych::faRotations`; recorded on our side in [stats/testdata/crosslang_baseline.R](stats/testdata/crosslang_baseline.R) and in the comment at `factorParityTol` in [stats/factor_analysis_test.go](stats/factor_analysis_test.go)
- **What**: when several rotation starts tie on the highest hyperplane count, psych breaks the tie by applying `which()` to the tied rows alone and then uses that result as a start number. `which()` returns a position within the subset, so with starts 1 and 3 tied and start 3 the simpler one, psych selects start 2, which never tied. The fit comparison that follows is computed and never assigned, so it has no effect. Our port selects by criterion value and does not share the defect; the only thing that depends on it is which solution psych's own baselines record.
- **Suggestion**: report it to the maintainer. There is no GitHub route: psych declares no `BugReports` URL, `revelle` has no psych repository, and `cran/psych` is a read-only mirror with issues disabled, so the only channel is the maintainer address in `DESCRIPTION` (`revelle@northwestern.edu`). A report needs the reproduction (hyperplane `c(.5,.4,.5,.3)` with complexity `c(1.3,1.0,1.1,1.0)` selects 2, not 3) and the fix — index back into the tied set at both steps, and assign the fit step. Sending it goes out under the owner's name, so it waits for the owner.
- **Status**: pending

### [2026-09-14] — `stats` functions outside the input guard still crash on a nil list
- **Where**: `stats/numericinput.go` `numericSlice`, reached from goroutines in `stats/regression_shared.go` `gatherRegressionInputs`; `Correlation` and `Covariance`, which reach `numericSlice` through `requireNumericPair`; and the functions that call `AtomicDo` or `Data` on an argument without converting it: `LogisticRegression`, `ChiSquareGoodnessOfFit`, `ChiSquareIndependenceTest`, `Silhouette`'s labels, `KNNClassify`'s labels, `KNNRegress`'s targets
- **What**: `numericSlice` checks only for a nil interface, so a typed nil `*DataList` reaches `AtomicDo` and panics, and `gatherRegressionInputs` calls it from goroutines it starts itself. Measured on this line on 2026-09-27: `LinearRegression(x, typedNil)`, `PoissonRegression(typedNil, x)`, `WeightedLinearRegression` with a typed nil in any argument and `LassoRegression(typedNil, …)` end the process with a nil-pointer panic that no caller can recover, and every other function named above panics in the caller's goroutine on a typed nil. The hypothesis tests are not on this list: they convert every argument through `asDataList` before any goroutine starts, so a nil list there gets the error an empty one gets. Nor are `Skewness` and `Kurtosis` since `core-utils-cleanup`: they read through `ProcessData`, which refuses a nil list with an error, and an empty sample was already an error for them.
- **Suggestion**: convert the remaining entry points through `asDataList`, or add a typed-nil check to `numericSlice` so the goroutines in `gatherRegressionInputs` get an error back instead of panicking, then widen `stats-input-type-guard` to name them. Decide first whether a nil list is an error or an empty sample. The hypothesis tests did not need that decision, because every one of them already refused an empty list.
- **Status**: pending

### [2026-09-14] — two deeply nested values with different leaves count as one
- **Where**: `cell_identity.go` `maxCellEncodeDepth` and the interface arm of `writeCellValue`
- **What**: a `[]any` level spends two encoding levels (the slice and the interface inside it), so a value nested more than 32 `[]any` deep reaches the limit of 64. Past it an interface is written as its type with no address, so two such values that differ only in their deepest leaf encode alike. Measured on this line on 2026-09-27: a list holding a depth-40 value with leaf 1 and two with leaf 2 reports `Count` 3 for either, while the same list at depth 30 reports 1 and 2. Grouping, pivoting and merging key on the same encoding. The answer is silently wrong.
- **Suggestion**: at the limit, unwrap the interface and write the address of the value inside it, which is what the comment above `maxCellEncodeDepth` says already happens.
- **Status**: pending

### [2026-09-14] — `DataList.Shift` and `Rolling.Apply` flatten a slice cell
- **Where**: `datalist_window.go` `Shift` and `RollingDataList.Apply`
- **What**: both build their result through `NewDataList`, which flattens every slice. Measured on this line on 2026-09-27: `Append([]byte{1, 2}, 3)` then `Shift(0)` gives `[1 2 3]`, three cells for two, and on a four-row list `Rolling(RollingOptions{Window: 2}).Apply` with a function returning `[]float64{1, 2}` gives seven cells, `[<nil> 1 2 1 2 1 2]`.
- **Suggestion**: build the result with `Append`, or wrap each cell with `Cell`. The length and cells of the result change for lists holding slices, which this line can take.
- **Status**: pending

### [2026-09-12] — a bare `[]byte` still flattens in the constructors
- **Where**: `datalist.go` `flattenWithNilSupport`
- **What**: a `[]byte` cell is one value everywhere it is read — counted, matched, grouped and ordered by content — but `NewDataList([]byte{0, 255})` still produces two cells holding `0` and `255`, because the constructor flattens every slice. The owner ruled on 2026-09-12 that the flattening stays: it is what makes `NewDataList` read like constructing a pandas Series. `one-value-one-cell` gave that decision an escape hatch, `NewDataList(Cell(blob), …)`, so the remaining gap is only that a reader who writes the bare form and then searches for the blob gets 0 with nothing to explain it.
- **Suggestion**: documentation, not code. `Docs/DataList.md` now describes the flattening and `Cell` together; check that the `Count`/`Counter` examples build their list with `Append` or `Cell` so they are runnable as written, and consider whether `Count` should say something when it is handed a slice that the receiving list could not be holding.
- **Status**: pending

### [2026-09-12] — `labelKey` still merges nested values that print alike
- **Where**: `datatable_encode.go` `labelKey`, its default arm
- **What**: `identify-uncomparable-cells` gave `encodeGroupKey` and `uniqueKey` a recursive encoder, so `[]any{1}` and `[]any{"1"}` are no longer one group. `labelKey` has the same non-recursive shape (`%T:%#v`), so it still merges them — `%#v` separates an int from a string but not `[]any{1}` from `[]any{1.0}`. It was left out on purpose: its integer rule is by value where cell identity is by type, so folding it into the same encoder would change what a label means, not just fix a collision.
- **Suggestion**: decide whether an encoder label is identity or value. If identity, point it at `encodeCell` and accept that `1` and `1.0` stop sharing a label. If value, it needs its own recursion with the value rule applied at every level. Either way it is a handful of lines once the question is answered.
- **Status**: pending

### [2026-09-12] — a self-referential slice takes the process down in `NewDataList`
- **Where**: `datalist.go` `flattenWithNilSupport`
- **What**: it recurses into every `reflect.Slice` with no depth limit, so a slice containing itself exhausts the stack. Measured on 2026-09-12: `cyclic := []any{1}; cyclic[0] = cyclic; insyra.NewDataList(cyclic)` ends with `fatal error: stack overflow`. That is not a panic, `recover` cannot catch it, and the library promises never to terminate. `encodeCell` was given a depth limit of 64 for exactly this reason; the flattener was not touched because it is the constructor's hot path and the fix should be measured against it.
- **Suggestion**: the same depth bound, or a visited-pointer set. A bound is cheaper and a 64-deep slice literal is already pathological; a visited set is exact but costs an allocation per construction. Measure `NewDataList` on a large flat slice before and after, because that path runs for every table built from a slice.
- **Status**: pending

### [2026-09-12] — reading a Parquet file and writing it back still changes some column types
- **Where**: `parquet/internal.go` `inferArrowType`
- **What**: the reader handles twenty Arrow types; the writer emits eight. `parquet-binary-is-bytes` added `Binary`, so a binary column now survives a round trip, but a `Date32` column still comes back as a timestamp, a decimal as its text, and an `Int16` as an `Int64`.
- **Suggestion**: `inferArrowType` infers from Go values, so it cannot tell an `int16` that came from a `Date32` column from any other. Carrying the source schema through a read would fix it properly; inferring `time.Time` to `Date64` would guess wrong on ordinary data. Worth doing only if round-tripping is a use case someone has.
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
- **Where**: `go.mod` — `chromedp v0.12.1`, `cdproto v0.0.0-20250120090109-d38428e4d9c8` (pulled in via `go-echarts/snapshot-chromedp`)
- **What**: two blockers held the chain back. (1) `chromedp v0.15.0+` and newer `cdproto` require go >= 1.26; this line's directive is 1.26.8, so that one no longer applies here. (2) From `v0.13.0` chromedp requires `go-json-experiment/json`, whose generic variadics panic govulncheck's SSA pass ("got jsontext.Value, want variadic parameter of unnamed slice or string type"). Measured on dev on 2026-09-24: x/vuln v1.3.0 and v1.7.0 built with go1.25.14 both panic on it, and v1.7.0 built with go1.26.5 completes. This line's Vulnerability Scan job builds x/vuln v1.3.0 with Go 1.26.x, a combination nobody has measured. The chain moved to dev's `v0.12.1` when dev was merged in on 2026-09-27, the newest version below `go-json-experiment`.
- **Suggestion**: at this line's next dependency refresh, take the whole chain to its newest version and run the Vulnerability Scan job's own govulncheck (v1.3.0 under Go 1.26.x) on it; if it panics, move the job to x/vuln v1.7.0 in the same change.
- **Status**: pending
