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
   must be archived, not left sitting in `openspec/changes/`.** Archiving a
   change that creates a capability leaves `TBD - created by archiving change …`
   as its Purpose; replace it in the same commit with one or two sentences on
   what the capability is for.

Every main spec must pass `openspec validate --specs --strict`, and the
GolangCI-Lint workflow runs it on every push and pull request: a Purpose of at
least 50 characters, every requirement's text before its first scenario at most
500 characters, a SHALL or MUST in each, and a scenario under each. To shorten a
requirement, split it into requirements that each state one behaviour, keep
the original header on the first so a later `MODIFIED` delta still finds it,
and keep every rule and scenario.

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
- **A bump that changes what insyra returns is not part of the refresh.** If updating a module changes a result or an output insyra produces, stop at the newest version that changes nothing and record the difference. The update then lands as its own change, with its changelog entry, so the behaviour change is reviewed and announced rather than carried in with a version bump.
- Anything held back — its newest version needs a newer Go, is inside an advisory, breaks the build or a tool CI depends on, or changes what insyra returns — goes into the Follow-ups below with the reason, the way `chromedp` is.

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
- Thread safety is on by default via the actor model. `Config.Dangerously_TurnOffThreadSafety()` exists but is explicitly discouraged; `Config.TurnOnThreadSafety()` turns it back on without resetting other settings.
- **Names that depart from Go convention on purpose.** Ruled by the owner on 2026-10-12 (#212). Do not raise them again in a review.
  - **Getters keep the `Get` prefix** and pair with a `Set…` setter (`GetName`/`SetName`, `GetLogLevel`/`SetLogLevel`). Effective Go drops `Get` from field getters, but the module has more than a hundred exported `Get…` names. Dropping the prefix would break almost every caller for no change in behaviour. A new getter follows the library: `Get` plus the setter's noun (`GetUseColoredOutput` for `SetUseColoredOutput`).
  - **`Dangerously_TurnOffThreadSafety` keeps its underscore.** A name that looks wrong is the warning. Naming linters report only the declaration, never a caller.
  - **`isr` keeps both `Of` and `From`** on `DL` and `DT`, an exception to the one-name rule, after Java's `List.of` (list the elements) and Rust's `From::from` (convert what you have).
  - **`isr`'s `Push` is not a second name.** `dl.Push` is `Append` returning the isr list so the chain continues, which `Append` cannot do without the isr list leaving `IDataList`. `dt.Push` adds rows or columns depending on the type it is given.
  - Column conversions use the library's two nouns: an index is the Excel-style letters (`"AA"`), and a number is the 0-based position (`ColIndexToNumber`, `ColNumberToIndex`, `GetColIndexByNumber`).
- **Log at Info only what the return value does not say.** Ruled by the owner on 2026-10-13 (#215). An Info line is for a guess the library made about the data (an encoding it detected), a change made outside the call's own result (a column `ToSQL` added to an existing table), the progress of a long operation (`nn.Sequential.Fit`'s epochs, preparing the Python environment), or a switch changing state (acceleration, thread safety). Never log a line that only confirms success: a nil error already says the file was saved. The default level stays `LogLevelInfo`, because the lines that remain are the ones a user needs to see; a program that wants only warnings calls `SetLogLevel(LogLevelWarning)`, and one with its own logging calls `Config.SetLogger`.
- **Every enum whose underlying type is an integer has `String()`.** A known value prints its constant's name, an unknown one prints `TypeName(n)`, and an error that reports an enum value formats it with `%v`, never `%d`.
- `AtomicDo` serializes access to ONE instance (same-instance nesting is safe, e.g. `Stdev`→`Var`). To read/operate on MULTIPLE instances atomically, use `insyra.AtomicDoAll(func(){...}, a, b, ...)` — it locks all given DataList/DataTable instances together in a deadlock-free order. Do NOT nest `AtomicDo` on a *different* instance inside a callback: that inner call runs WITHOUT locking the other instance and can race a concurrent mutation. (`engine/atomic.AtomicDoN([]*Actor, f)` is the same primitive for arbitrary user structs holding an `*atomic.Actor`.)
- Error handling uses an instance-level `Err()` pattern rather than returning errors from every method (check `.Err()` after chained calls; `PopErr()` reads and clears it). Wrapper packages such as `isr` record their failures through the exported `SetErr(packageName, funcName, msg, args...)`.
- Use the `isr` package for convenient syntax and the root `insyra` package where performance matters: some `isr` constructors convert or copy their input (building a table from lists measured about 9x slower than `insyra.NewDataTable`), while methods called through the wrapper cost almost nothing extra.
- **One JSON library: `github.com/goccy/go-json`.** Every non-test file reads and writes JSON through it, imported as `json "github.com/goccy/go-json"`; `golangci-lint` (depguard) fails on `encoding/json` or another JSON library outside tests. Tests may import `encoding/json`, and should where it is the independent reference insyra's output is checked against. The library is chosen for speed and must behave like `encoding/json`, which the `json-codec-conformance` spec pins.
  - **When a faster library turns up, switch everything to it.** Measure it against the current one with `go test -run '^$' -bench BenchmarkJSONPaths -count 5 .` (best run of each). If it is faster and passes every `json-codec-conformance` test without settings that depart from `encoding/json` (unsorted map keys, no HTML escaping and the like), move every non-test file to it in one change, name it here and in `.golangci.yml`, and record both libraries' results in that change.

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

### [2026-10-21] — device tensors live on one card per process
- **Where**: `accel/internal/wgpu/wgpu.go` `open`, `accel/resident.go` `ResidentDevice`, `nn/device_tensor.go` `To`
- **What**: the WebGPU backend opens the adapter WebGPU names as its default and has no way to be told which one. `accel.DefaultDevice` returns the first device of the session's list, which on a host with several cards may not be that adapter, and `Tensor.To(device)` uses the `Device` only as a label. Operands on different devices are refused, so nothing runs wrongly, but a program cannot choose a card. `Config.Devices`, `PreferredDevices` and `INSYRA_ACCEL_DEVICES` narrow what is discovered, not where tensors are placed. Only a single-card host has been available to test.
- **Suggestion**: one WebGPU handle per adapter, buffers and queues per handle, `To(cards[i])` choosing the card, and a gated multi-card test on a host that has two. Data-parallel training across cards (gradient synchronisation) is a separate, larger change.
- **Status**: pending

### [2026-10-20] — `excelize` pinned to an untagged commit for two advisories
- **Where**: `go.mod`, `github.com/xuri/excelize/v2`, reached through `ReadExcel`, `ReadExcelSheet`, `DataTable.WriteExcel` and `csvxl.AppendCsvToExcel`
- **What**: v2.11.0, the newest tag, is inside GO-2026-6689 (a panic and memory exhaustion from an unbounded CFB stream allocation) and GO-2026-6688 (mutually referencing array formulas recurse until the stack overflows); govulncheck traces both to these four functions, and the Vulnerability Scan job went red on every branch. The fixes are on the module's `master` and in no tag, so `go.mod` names the commit `v2.11.1-0.20261020033641-611b222fd8f7`. `govulncheck ./...` reports 0 affected vulnerabilities with it, `go.mod`'s `go` directive is unchanged, and the root and `csvxl` tests pass.
- **Suggestion**: move to the next tagged release (v2.11.1 or later) as soon as it exists and drop this entry. Until then the dependency refresh before a release must not step back to v2.11.0.
- **Status**: pending (a pseudo-version is a decision for the owner: it trades a tagged release for a fix)

### [2026-10-17] — remove `py.RunCodeWithTimeout` in 0.4
- **Where**: `py/py.go`
- **What**: `RunCodeWithTimeout` is marked Deprecated on this line (`RunCodeContext` with `context.WithTimeout` is the same call). `0.4` has no such function.
- **Suggestion**: delete it when this line is merged into `0.4`, together with its test `TestRunCodeWithTimeoutKeepsItsMeaning`.
- **Status**: pending

### [2026-10-10] — `chromedp` held at v0.16.0
- **Where**: `go.mod`, `github.com/chromedp/chromedp` and the `cdproto` it requires, reached through `github.com/go-echarts/snapshot-chromedp` v0.0.5, which `plot.SavePNG` uses
- **What**: v0.16.0 is the newest `chromedp` insyra can take. v0.17.0 through v0.19.1 declare `go 1.27`. v0.20.0 and v0.20.1 declare `go 1.25` but change `chromedp.Run`, `FullScreenshot` and the query options to a generic API, so `snapshot-chromedp` v0.0.5, its newest release, no longer compiles (`too many arguments in call to chromedp.Run`, `undefined: chromedp.ByQuery`). Measured on 2026-10-10: a bar chart saved through `plot.SavePNG` with `chromedp` v0.12.1 and with v0.16.0 produced byte-identical PNGs.
- **Suggestion**: take `chromedp` v0.20 or later once `snapshot-chromedp` releases a version that compiles against it, and compare `plot.SavePNG` output before and after, as above.
- **Status**: pending

### [2026-10-13] — a guess below some confidence is still returned as the encoding
- **Where**: `utils.go` `detectEncodingOfSample`
- **What**: the answer for one input is now stable (`encoding-detection-is-deterministic`), but a 24-byte Big5 CSV (`名稱,數量` / `蘋果,3` / `香蕉,5`) can still be read as another encoding when the recognizers tie, and the read returns a nil error with a garbled header. The only sign is the Info line naming the guess.
- **Suggestion**: decide whether a guess below some confidence should be a warning or an error that asks for the encoding. That changes what a read does with such a file.
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

### [2026-09-19] — `gplot.SaveChart` still ends the program on a path it cannot write
- **Where**: `gplot/save_chart.go:17`
- **What**: it calls `insyra.LogFatal`, which under the default configuration is `log.Fatalf`, so a bad path ends the host program. Measured on 2026-09-19: a program calling `gplot.SaveChart(plt, "/no/such/dir/chart.png")` printed `<{[insyra - FATAL!]}> gplot.SaveChart: Failed to save chart: open /no/such/dir/chart.png: no such file or directory` and exited with status 1; the statement after the call never ran. `insyra.Config.SetDontPanic(true)` downgrades it to a log line, which is what `Docs/gplot.md` and `Docs/Configuration.md` tell readers. The `error-philosophy` spec on this line covers `gplot` and `plot` chart *construction* and `plot.SavePNG`'s path check; it does not cover `gplot`'s saving, correctly, and no doc, skill or spec claims otherwise.
- **Suggestion**: the fix is the error-returning `SaveChart`, which changes the signature and therefore stays on `0.4`. Nothing to do on the 0.3.x line but keep the documentation honest. When the two lines merge, this entry goes away with it.
- **Status**: pending (0.4 carries the fix; recorded here so the gap is not mistaken for closed)

### [2026-09-17] — psych 2.6.5's `faRotations` tie-break picks a start that did not tie, and nobody upstream has been told
- **Where**: upstream `psych::faRotations`; recorded on our side in [stats/testdata/crosslang_baseline.R](stats/testdata/crosslang_baseline.R) and in the comment at `factorParityTol` in [stats/factor_analysis_test.go](stats/factor_analysis_test.go)
- **What**: when several rotation starts tie on the highest hyperplane count, psych breaks the tie by applying `which()` to the tied rows alone and then uses that result as a start number. `which()` returns a position within the subset, so with starts 1 and 3 tied and start 3 the simpler one, psych selects start 2, which never tied. The fit comparison that follows is computed and never assigned, so it has no effect. Our port selects by criterion value and does not share the defect; the only thing that depends on it is which solution psych's own baselines record.
- **Suggestion**: report it to the maintainer. There is no GitHub route: psych declares no `BugReports` URL, `revelle` has no psych repository, and `cran/psych` is a read-only mirror with issues disabled, so the only channel is the maintainer address in `DESCRIPTION` (`revelle@northwestern.edu`). A report needs the reproduction (hyperplane `c(.5,.4,.5,.3)` with complexity `c(1.3,1.0,1.1,1.0)` selects 2, not 3) and the fix — index back into the tied set at both steps, and assign the fit step. Sending it goes out under the owner's name, so it waits for the owner.
- **Status**: pending

### [2026-09-12] — grouping, pivoting, merging and `Describe` still merge nested values that print alike
- **Where**: `datatable_groupby.go` `encodeGroupKey` and `uniqueKey`, `datatable_encode.go` `labelKey`, their default arms
- **What**: `encodeGroupKey` and `uniqueKey` fall back to `%T:%v`, which does not descend, so `[]any{1}` and `[]any{"1"}` produce the same key: `GroupBy`, `Pivot` and `Merge` put the integer and the string in one group, and `Describe`'s unique count counts them once. `labelKey` uses `%T:%#v`, which separates an int from a string but not `[]any{1}` from `[]any{1.0}`. Measured on 2026-09-13. `identify-uncomparable-cells` gave cell lookups a recursive encoder, `encodeCell`; on 0.4 `encodeGroupKey` and `uniqueKey` moved to it too, but that changes the keys existing data groups by, so it stayed off the 0.3.x line.
- **Suggestion**: moving `encodeGroupKey` and `uniqueKey` to `encodeCell` is a change of result, so it belongs to 0.4. `labelKey` needs a decision first: its integer rule is by value where cell identity is by type, so decide whether an encoder label is identity or value before pointing it at any encoder.
- **Status**: pending

### [2026-09-12] — a bare `[]byte` still flattens in the constructors
- **Where**: `datalist.go` `flattenWithNilSupport`
- **What**: a `[]byte` cell is one value everywhere it is looked up — counted and matched by content — but `NewDataList([]byte{0, 255})` still produces two cells holding `0` and `255`, because the constructor flattens every slice (pinned by `TestAnUnmarkedSliceStillFlattens`). The owner ruled on 2026-09-12 that the flattening stays: it is what makes `NewDataList` read like constructing a pandas Series. `one-value-one-cell` gave that decision an escape hatch, `NewDataList(Cell(blob), …)`, so the remaining gap is only that a reader who writes the bare form and then searches for the blob gets 0 with nothing to explain it.
- **Suggestion**: documentation, not code. `Docs/DataList.md` describes the flattening and `Cell` together; keep the `Count`/`Counter` examples building their list with `Append` or `Cell` so they are runnable as written, and consider whether `Count` should say something when it is handed a slice that the receiving list could not be holding.
- **Status**: pending

### [2026-09-12] — a decimal column is exact but is not a number to the rest of the library
- **Where**: `internal/utils` `IsNumeric` / `ToFloat64Safe`, and every numeric path behind them
- **Note (2026-09-14)**: measured on this line, a decimal column reaches `stats.SingleSampleTTest`, which returns a `NaN` result with a nil error, and `stats.Skewness`, which fails with "zero variance" because `SliceToF64` reads every decimal as 0; `stats.Correlation` refuses it. `ToJSON` writes a decimal cell as `{}`. `0.4` fixes these through changes that stayed there.
- **What**: `parquet-foreign-column-types` made `Decimal128` and `Decimal256` read as a go-decimal `decimal.Decimal`, exact and sorting by value. It is a struct, so `IsNumeric` says no and `ToFloat64Safe` cannot read it, which means the numeric path treats a decimal cell the way it treats a `time.Time` cell: as not a number. That follows the `time.Time` precedent exactly, and it is the reason the change did not go further on its own. But a money column is far likelier to want arithmetic than a date column is.
- **Suggestion**: the decision is whether a decimal is a number in insyra. If it is, `ToFloat64Safe` needs an arm for it, which today means going through `String()` and `strconv.ParseFloat` because go-decimal exposes no `Float64()`; adding one upstream would be cleaner and it is the same author's library. Weigh that against making `internal/utils`, which every value in the library passes through, depend on a decimal package.
- **Status**: pending

### [2026-09-12] — reading a Parquet file and writing it back still changes column types
- **Where**: `parquet/internal.go` `inferArrowType`
- **What**: the reader handles more Arrow types than the writer's seven, so a read-then-write round trip downgrades: a `Date32` column comes back as a timestamp, a decimal as a string column, an `Int16` as an `Int64`. A column of `[]byte` cells now survives, written as `Binary` (`parquet-write-bytes-as-binary`, 2026-10-18); before that it was written as the text `[65 45 48 49]`. Nothing is silently wrong, but the file is not the file that went in. Found on 2026-09-12 while fixing #371, which only concerned the read half.
- **Suggestion**: `inferArrowType` infers from Go values, so it cannot tell an `int16` that came from a `Date32` column from any other. Carrying the source schema through a read would fix it properly; inferring `time.Time` to `Date64` would not, and would guess wrong on ordinary data. Worth doing only if round-tripping is a use case someone has.
- **Status**: pending

### [2026-09-12] — `lp.SolveFromFile` returns a nil result table and the documented example dereferences it
- **Where**: `lp/lp.go` `SolveFromFile` and `SolveModel`; the example in `Docs/lp.md`
- **What**: on a timeout or a solver error both functions return `nil` as the first DataTable, and `SolveFromFile` returns `nil, nil` for more than one `timeoutSeconds` argument. `Docs/lp.md` says only "returns the result as two DataTable" and its example calls `result.Show()` straight away. A nil `*DataTable` panics on `Show()`; measured on 2026-09-12. Found while writing the parser tests in `test-unpinned-behaviour`, which cover the failure path of `parseGLPKOutputFromFile` (also nil on an unreadable file).
- **Suggestion**: returning an empty table instead of nil changes a returned value, so on the 0.3.x line the nil stays (the never-nil version lives on 0.4 as `lp-never-returns-a-nil-table`). `Docs/lp.md` was corrected on 2026-09-18: both Returns sections say the solution is nil when the solve produces none, and both examples check it. What is left is the code decision for a future release.
- **Status**: won't fix (2026-10-18). Both functions are Deprecated and removed in 0.4, where `Solve` and `SolveFile` return an error. Changing a returned value on a function that is about to go would break `result == nil` checks for little gain, and the documentation already tells callers to check. Delete this entry with the functions.

### [2026-08-01] — multi-GPU planning and execution coverage
- **Where**: `accel/planner.go` (`PlanShardable`, weighted per-device `ShardAssignment`s), `accel/exact.go` (per-assignment dispatch)
- **What**: the planner retains capability-weighted heterogeneous assignments and its existing `MergePolicy`. `ExecuteNearestExact` now dispatches one worker per assignment, uses the bounded chunk seam, merges by input range, and falls back per assignment without changing the exact CPU decision.
- **Suggestion**: run `INSYRA_ACCEL_GPU_TESTS=1 go test ./accel -run TestMultiDeviceParityConcurrentAndSequentialOnHardware` on a multi-GPU host, then record concurrent-versus-sequential wall clock for the 32k/8k saturation classes. The single-device host verifies correctness only; it cannot supply a multi-GPU speedup number.
- **Status**: single-device correctness verified; multi-GPU wall clock and non-Apple parity remain pending

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
