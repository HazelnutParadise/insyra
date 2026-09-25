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
- Anything held back — its newest version needs a newer Go, is inside an advisory, or breaks a tool CI depends on — goes into the Follow-ups below with the reason, the way the modules held back by the Go 1.25 directive already are.

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
- `AtomicDo` serializes access to ONE instance (same-instance nesting is safe, e.g. `Stdev`→`Var`). To read/operate on MULTIPLE instances atomically, use `insyra.AtomicDoAll(func(){...}, a, b, ...)` — it locks all given DataList/DataTable instances together in a deadlock-free order. Do NOT nest `AtomicDo` on a *different* instance inside a callback: that inner call runs WITHOUT locking the other instance and can race a concurrent mutation. (`engine/atomic.AtomicDoN([]*Actor, f)` is the same primitive for arbitrary user structs holding an `*atomic.Actor`.)
- Error handling uses an instance-level `Err()` pattern rather than returning errors from every method (check `.Err()` after chained calls; `PopErr()` reads and clears it). Wrapper packages such as `isr` record their failures through the exported `SetErr(packageName, funcName, msg, args...)`.
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
- Archive every OpenSpec change whose work is in the release on `dev` before the release branch is cut. `openspec/changes/` on the release branch holds only work that is not in it.
- Release only when every CI check is green, on the release PR and on the `main` merge commit it produces. A red check blocks the release until it is fixed or excluded by its own change, including one that fails on every branch.
- A release is always named with both the series name and the version number — "Huashan v0.3.1", never a bare "v0.3.1" — in the GitHub Release title and anywhere else the release is announced. The series name comes from `VersionName` in [version.go](version.go) and changes only when a new series starts.

Keep the English ([README.md](README.md), [CHANGELOG.md](CHANGELOG.md), `Docs/`) and Traditional-Chinese ([README_TW.md](README_TW.md), [CHANGELOG_TW.md](CHANGELOG_TW.md)) docs in lockstep — never update one side without the other.

## Agent Skills

[skills/insyra/](skills/insyra/) — for AI agents writing Go code using Insyra APIs.  
[skills/use-insyra-cli/](skills/use-insyra-cli/) — for AI agents operating via the CLI/REPL or `.isr` scripts.

## Follow-ups

Out-of-scope issues discovered during development, waiting for a decision. Delete an entry once it is resolved.

### [2026-09-20] — `TestSequentialFitMNISTConvergence` still pins numbers recorded on one machine
- **Where**: [nn/fit_mnist_test.go](nn/fit_mnist_test.go), the mean-loss and accuracy assertions
- **What**: Fit's sugar-changes-nothing proof compares against the transcribed `0.347310`, `0.165883` and `0.9547` instead of against the hand-written loop it claims to reproduce. Its sibling `TestSequentialMNISTConvergence` failed exactly that way on `ubuntu-latest` on 2026-09-20 — `0.163855` recorded on arm64 against `0.163840` measured on amd64, while the two code paths still agreed with each other — and now runs the hand loop in the same process (`mnist-proof-compares-runs`). Fit's numbers happen to hold on both platforms measured so far, so nothing is red today.
- **Suggestion**: the same treatment. Run the documented hand loop inside the test and compare the two curves, keeping only bounds any platform meets. It costs one more MNIST run, about 30 seconds in CI. Worth doing the next time `nn`'s training path is touched, or sooner if a third platform disagrees.
- **Status**: pending

### [2026-09-20] — `oblimin` ignores its starting point, so `Restarts` costs it N identical runs
- **Where**: `stats/internal/fa/psych_faRotations.go`, the `"oblimin"` arm of the switch in `FaRotations`
- **What**: every other method rotates from the start it was handed; oblimin builds its own identity matrix and rotates from that, ignoring the start entirely. The comment says the identity start is deliberate, "better SPSS compatibility than random starts". Measured on 2026-09-20 after `orthogonal-rotation-starts` landed here: over the 20 datasets of `stats/factor_analysis_test.go` and all four extractions, oblimin returns the `Restarts: 1` answer in all 240 combinations at `Restarts` 2, 5 and 20, while every other GPA method differs from its single-start answer in more than half of them. On the `noisyStructure` fixture, best of 5, `Restarts: 20` takes 74.6 ms to return the 2.6 ms answer. It matters more now than before: the starts oblimin is refusing used to include two oblique matrices and are now all legitimate.
- **Suggestion**: the decision is SPSS parity against an honest `Restarts`, not how to write it. Either let oblimin use the starts like everything else, which moves oblimin results for `Restarts > 1` and may move them away from SPSS, or keep the identity start and reject `Restarts > 1` for oblimin so the parameter stops claiming a search it does not run. `0.4` took the first (6c4fce1, which also pins that at `Delta` 0 oblimin then agrees with quartimin bit for bit); it is not on this line because it moves results with nothing in v0.3's documentation describing the move. `TestRestartsParameter` in `stats/verify_more_test.go` already covers oblimin and will pin whichever is chosen. `Docs/stats.md` and `skills/insyra/references/stats.md` state the current behaviour meanwhile.
- **Status**: pending

### [2026-09-19] — `gplot.SaveChart` still ends the program on a path it cannot write
- **Where**: `gplot/save_chart.go:17`
- **What**: it calls `insyra.LogFatal`, which under the default configuration is `log.Fatalf`, so a bad path ends the host program. Measured on 2026-09-19: a program calling `gplot.SaveChart(plt, "/no/such/dir/chart.png")` printed `<{[insyra - FATAL!]}> gplot.SaveChart: Failed to save chart: open /no/such/dir/chart.png: no such file or directory` and exited with status 1; the statement after the call never ran. `insyra.Config.SetDontPanic(true)` downgrades it to a log line, which is what `skills/insyra/references/plotting.md` already tells readers. The `error-philosophy` spec on this line covers `gplot` and `plot` chart *construction* and `plot.SavePNG`'s path check; it does not cover `gplot`'s saving, correctly, and no doc, skill or spec claims otherwise.
- **Suggestion**: the fix is the error-returning `SaveChart`, which changes the signature and therefore stays on `0.4`. Nothing to do on the 0.3.x line but keep the documentation honest. When the two lines merge, this entry goes away with it.
- **Status**: pending (0.4 carries the fix; recorded here so the gap is not mistaken for closed)

### [2026-09-19] — `BartlettTest` panics when the group variances come out equal
- **Where**: `stats/ftest.go` `BartlettTest`, through `stats/distutil.go` `chiSquaredPValue`
- **What**: Bartlett's `T` is a difference of logs, so for groups whose variances agree it lands at or just below zero by rounding. `chiSquaredPValue` passes it straight to `distuv.ChiSquared{K: df}.CDF`, which panics with `cephes: parameter out of bounds` for any negative argument — measured on 2026-09-19, `chiSquaredPValue(-1e-16, 1)` panics while `chiSquaredPValue(0, 1)` returns 1. Whether two identical groups panic therefore depends on the values: `{0.1, 0.2, 0.3}` twice returns statistic 0 and p 1, while `{-0.32329881667362437, -0.319580921043235, 0.9093452052941211, 0.9788142136474671}` twice panics. A random sweep over k=2..4 and n=2..40 hit it 5 times in 135 tries. Pre-existing: the same input panics identically on e6fbb53, so this is not a backport regression. `df <= 0` panics the same way (`chiSquaredPValue(1, 0)`), which `FriedmanTest`, `KruskalWallis`, `ChiSquareTest`, `PartialCorrelation` and `FactorAnalysis` also reach.
- **Suggestion**: clamp in `chiSquaredPValue` — a negative `chi2` is a rounding artefact of a statistic that is zero, so returning 1 for it is the right answer, and a non-positive `df` should return NaN the way `tQuantile` already does. That changes a panic into a value for every caller at once, which needs deciding before it is done: it is a behaviour change, though the old behaviour was a panic.
- **Status**: pending

### [2026-09-19] — govulncheck fails on every branch: GO-2026-6452 wrongly lists excelize v2.11.0 as unfixed
- **Where**: `go.mod` — `github.com/xuri/excelize/v2 v2.11.0`; the traces are `read.go` `ReadExcelSheet`, `csvxl` `replaceSheet` and `paddedSheetCells`
- **What**: GO-2026-6452, "Panic via negative shared-string index", lists `Fixed in: N/A`, so `govulncheck ./...` exits 3 on every branch and the Vulnerability Scan job is red. The entry is wrong. Its source, GHSA-fx5j-qcqg-grpf, gives v2.11.0 as the patched version; the fix, commit 93f0b3c (qax-os/excelize#2331), is an ancestor of the v2.11.0 tag; and v2.11.0's `getValueFrom` checks `xlsxSI < 0`. Checked 2026-09-24. golang/vulndb#6510 reports it, with several duplicates, and is still open. GitHub's advisory database lists no advisory covering v2.11.0.
- **Suggestion**: releases need a green `main`. The owner ruled on 2026-09-25 to wait for golang/vulndb to correct the entry rather than exclude it in the workflow, so the next release waits on golang/vulndb#6510. When the entry is fixed, rerun the Vulnerability Scan job on `dev` and delete this entry.
- **Status**: pending (waiting on golang/vulndb#6510)

### [2026-09-17] — psych 2.6.5's `faRotations` tie-break picks a start that did not tie, and nobody upstream has been told
- **Where**: upstream `psych::faRotations`; recorded on our side in [stats/testdata/crosslang_baseline.R](stats/testdata/crosslang_baseline.R) and in the comment at `factorParityTol` in [stats/factor_analysis_test.go](stats/factor_analysis_test.go)
- **What**: when several rotation starts tie on the highest hyperplane count, psych breaks the tie by applying `which()` to the tied rows alone and then uses that result as a start number. `which()` returns a position within the subset, so with starts 1 and 3 tied and start 3 the simpler one, psych selects start 2, which never tied. The fit comparison that follows is computed and never assigned, so it has no effect. Our port selects by criterion value and does not share the defect; the only thing that depends on it is which solution psych's own baselines record.
- **Suggestion**: report it to the maintainer. There is no GitHub route: psych declares no `BugReports` URL, `revelle` has no psych repository, and `cran/psych` is a read-only mirror with issues disabled, so the only channel is the maintainer address in `DESCRIPTION` (`revelle@northwestern.edu`). A report needs the reproduction (hyperplane `c(.5,.4,.5,.3)` with complexity `c(1.3,1.0,1.1,1.0)` selects 2, not 3) and the fix — index back into the tied set at both steps, and assign the fit step. Sending it goes out under the owner's name, so it waits for the owner.
- **Status**: pending

### [2026-09-14] — `stats` functions outside the input guard still crash on a nil list
- **Where**: `stats/anova.go` `OneWayANOVA` (and `KruskalWallis`, `FriedmanTest`), `TwoWayANOVA`, and `stats/numericinput.go` `numericSlice`
- **What**: `stats-input-type-guard` covers only the functions that read their arguments through `asDataList` or `numericSlice`. Reported by the 2026-09-14 backport review: a nil group given to `OneWayANOVA`, `KruskalWallis` or `FriedmanTest` is dereferenced inside a goroutine (`groups[i].AtomicDo` in `OneWayANOVA`), which ends the process because no caller can recover it; `TwoWayANOVA` and other functions calling `AtomicDo` on the argument directly panic. `numericSlice` checks only for a nil interface, so a typed nil `*DataList` still reaches `AtomicDo` and panics; the t, z, F, Bartlett and Levene tests and `CalculateMoment` avoid that by converting through `asDataList` first.
- **Suggestion**: route these entry points through `asDataList`, which already turns a typed nil into an empty list, or check for nil before any goroutine starts, then widen the spec to name them. Decide first whether a nil list is an error or an empty sample.
- **Status**: pending

### [2026-09-14] — two deeply nested values with different leaves count as one
- **Where**: `cell_identity.go` `maxCellEncodeDepth` and the interface arm of `writeCellValue`
- **What**: a `[]any` level spends two encoding levels (the slice and the interface inside it), so a value nested more than 32 `[]any` deep reaches the limit of 64. Past it an interface is written as `p:interface {}`, its type with no address, so two such values that differ only in their deepest leaf encode alike: measured on 2026-09-14, a list holding a depth-40 value with leaf 1 and two with leaf 2 reports `Count` 3 for either. On v0.3.2 the same `Count` panicked, so nothing that worked changed, but the answer is silently wrong.
- **Suggestion**: at the limit, unwrap the interface and write the address of the value inside it, which is what the comment above `maxCellEncodeDepth` says already happens.
- **Status**: pending

### [2026-09-14] — `DataList.Shift` flattens a slice cell
- **Where**: `datalist_window.go` `Shift`
- **What**: `Shift` builds its result through `NewDataList`, which flattens every slice, so a list holding `[]byte{1, 2}` and `3` comes back three cells long: measured on 2026-09-14, `Append([]byte{1, 2}, 3)` then `Shift(0)` gives `[1 2 3]`. Present on v0.3.2; found while backporting `one-value-one-cell`.
- **Suggestion**: build the result with `Append`, or wrap each cell with `Cell`. The length and cells of the result change for lists holding slices, so decide which line takes it.
- **Status**: pending

### [2026-09-14] — 26 main specs fail `openspec validate --specs --strict`
- **Where**: `openspec/specs/*/spec.md`, mostly the `## Purpose` section
- **What**: on 2026-09-14 the strict run reports 26 failures of 101 specs: the seven `accel-*` specs, `changelog`, `cli-entry`, `command-registry`, `core-preprocessing`, `dsl-commands`, `env-management`, the five `ml-*` specs, `nn-inference`, `nn-training`, `repl-engine`, `script-runner`, `stats-clustering`, `stats-decomposition`, `stats-knn` and `stats-regression`. dev at e6fbb53 had 27 of 50; the 0.4 backport fixed `verification-integrity` and gave every spec it added a real Purpose. Nothing in CI runs the command.
- **Suggestion**: write each Purpose from its requirements as its own change, then add the strict run to the lint workflow so the count cannot climb again.
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

### [2026-09-12] — a self-referential slice takes the process down in `NewDataList`
- **Where**: `datalist.go` `flattenWithNilSupport`
- **What**: it recurses into every `reflect.Slice` with no depth limit, so a slice containing itself exhausts the stack. Measured on 2026-09-13: `cyclic := []any{1}; cyclic[0] = cyclic; insyra.NewDataList(cyclic)` ends with `fatal error: stack overflow`. That is not a panic: `recover` cannot catch it, and the whole program ends. `encodeCell` was given a depth limit of 64 for exactly this reason; the flattener was not touched because it is the constructor's hot path and the fix should be measured against it.
- **Suggestion**: the same depth bound, or a visited-pointer set. A bound is cheaper and a 64-deep slice literal is already pathological; a visited set is exact but costs an allocation per construction. Measure `NewDataList` on a large flat slice before and after, because that path runs for every table built from a slice.
- **Status**: pending

### [2026-09-12] — a decimal column is exact but is not a number to the rest of the library
- **Where**: `internal/utils` `IsNumeric` / `ToFloat64Safe`, and every numeric path behind them
- **Note (2026-09-14)**: measured on this line, a decimal column reaches `stats.SingleSampleTTest`, which returns a `NaN` result with a nil error, and `stats.Skewness`, which fails with "zero variance" because `SliceToF64` reads every decimal as 0; `stats.Correlation` refuses it. `ToJSON` writes a decimal cell as `{}`. `0.4` fixes these through changes that stayed there.
- **What**: `parquet-foreign-column-types` made `Decimal128` and `Decimal256` read as a go-decimal `decimal.Decimal`, exact and sorting by value. It is a struct, so `IsNumeric` says no and `ToFloat64Safe` cannot read it, which means the numeric path treats a decimal cell the way it treats a `time.Time` cell: as not a number. That follows the `time.Time` precedent exactly, and it is the reason the change did not go further on its own. But a money column is far likelier to want arithmetic than a date column is.
- **Suggestion**: the decision is whether a decimal is a number in insyra. If it is, `ToFloat64Safe` needs an arm for it, which today means going through `String()` and `strconv.ParseFloat` because go-decimal exposes no `Float64()`; adding one upstream would be cleaner and it is the same author's library. Weigh that against making `internal/utils`, which every value in the library passes through, depend on a decimal package.
- **Status**: pending

### [2026-09-12] — reading a Parquet file and writing it back still changes column types
- **Where**: `parquet/internal.go` `inferArrowType`
- **What**: the reader handles more Arrow types than the writer's seven, so a read-then-write round trip downgrades: a `Date32` column comes back as a timestamp, a decimal or a `Binary` column as a string column, an `Int16` as an `Int64`. A `Binary` column loses more than its type: `appendValue` writes each `[]byte` through `conv.ToString`, so `A-01` goes out as the text `[65 45 48 49]` (measured 2026-09-14). On 0.4 `parquet-binary-is-bytes` also writes a column of `[]byte` cells as Arrow `Binary`; that changes file output, so it stayed off the 0.3.x line. Nothing is silently wrong, but the file is not the file that went in. Found on 2026-09-12 while fixing #371, which only concerned the read half.
- **Suggestion**: `inferArrowType` infers from Go values, so it cannot tell an `int16` that came from a `Date32` column from any other. Carrying the source schema through a read would fix it properly; inferring `time.Time` to `Date64` would not, and would guess wrong on ordinary data. Worth doing only if round-tripping is a use case someone has.
- **Status**: pending

### [2026-09-12] — `lp.SolveFromFile` returns a nil result table and the documented example dereferences it
- **Where**: `lp/lp.go` `SolveFromFile` and `SolveModel`; the example in `Docs/lp.md`
- **What**: on a timeout or a solver error both functions return `nil` as the first DataTable, and `SolveFromFile` returns `nil, nil` for more than one `timeoutSeconds` argument. `Docs/lp.md` says only "returns the result as two DataTable" and its example calls `result.Show()` straight away. A nil `*DataTable` panics on `Show()`; measured on 2026-09-12. Found while writing the parser tests in `test-unpinned-behaviour`, which cover the failure path of `parseGLPKOutputFromFile` (also nil on an unreadable file).
- **Suggestion**: returning an empty table instead of nil changes a returned value, so on the 0.3.x line the nil stays (the never-nil version lives on 0.4 as `lp-never-returns-a-nil-table`). `Docs/lp.md` was corrected on 2026-09-18: both Returns sections say the solution is nil when the solve produces none, and both examples check it. What is left is the code decision for a future release.
- **Status**: pending (documentation corrected; the nil return itself is undecided)

### [2026-09-10] — how insyra turns a number into text is decided nowhere
- **Where**: every path that writes a number as text — `internal/ccl/stdlib_string.go` `toString` (behind CCL's `CONCAT`, `TOSTR` without a format, `LEN`, `UPPER`, …), the `&` operator in `internal/ccl/ccl_evaluator.go`, `ToCSV`, `ToJSON`, and `Show`.
- **What**: each path uses Go's default formatting, so a `float64` outside roughly 1e-5..1e21 comes out in exponent form. Measured on one column `[0.0000001, 1e6, 1e21, 12345.678]`: CCL `'x' & A` gives `x1e-07`, `x1e+06`, `x1e+21`; `ToCSV` writes `1e-07`, `1e+06`, `1e+21`; `ToJSON` writes `1e-07`; `Show` prints `1.0000e-07`, `1000000`, `1.0000e+21`, `1.2346e+04`, a display format of its own. Every numeric literal in CCL is a `float64`, so `LEN(1000000)` is `5` while the same value read from an integer column gives `7`.
- **Why it is not a CCL-only fix**: CCL is not the odd one out — `ToCSV` and `ToJSON` write the same exponent form. Changing CCL alone would make `'x' & A` say `x0.0000001` while `ToCSV` of the same column still says `1e-07`. That split may well be the right answer: a serialised number and a number embedded in text do different jobs, and `1e-07` in a CSV or JSON number cell reads back as the same value, whereas `id-1e-07` in a string reads back as nothing useful. But it should be chosen, not fall out of a patch to one file. It is also a different question from the `<nil>` that CCL's `&` still writes for a nil operand on this line: CCL is the only path that writes that — `ToCSV` writes an empty cell and `ToJSON` writes `null`.
- **Suggestion**: decide per path. The strongest case for a change is text built inside CCL, where `strconv.FormatFloat(f, 'f', -1, 64)` gives the shortest exact decimal with no exponent (`"0.0000001"`). The cost is that a genuinely huge value gets long (`1e300` is 301 characters), so a magnitude threshold may be wanted. Any change is **breaking** for expressions that build text from a float.
- **Status**: pending

### [2026-08-01] — multi-GPU planning and execution coverage
- **Where**: `accel/planner.go` (`PlanShardable`, weighted per-device `ShardAssignment`s), `accel/exact.go` (per-assignment dispatch)
- **What**: the planner retains capability-weighted heterogeneous assignments and its existing `MergePolicy`. `ExecuteNearestExact` now dispatches one worker per assignment, uses the bounded chunk seam, merges by input range, and falls back per assignment without changing the exact CPU decision.
- **Suggestion**: run `INSYRA_ACCEL_GPU_TESTS=1 go test ./accel -run TestMultiDeviceParityConcurrentAndSequentialOnHardware` on a multi-GPU host, then record concurrent-versus-sequential wall clock for the 32k/8k saturation classes. The single-device host verifies correctness only; it cannot supply a multi-GPU speedup number.
- **Status**: single-device correctness verified; multi-GPU wall clock and non-Apple parity remain pending

### [2026-08-01] — `ToF64Slice` still fabricates zeros for 54 callers outside `stats`
- **Where**: `datalist.go` `ToF64Slice`, and its callers in `plot/`, `gplot/`, `cli/`, `datalist_interpolation.go`
- **What**: it routes every value through `insyra.ToFloat64`, which has no failure channel and yields `0` for anything it cannot parse, then returns a full-length slice — so a caller cannot tell a real zero from a value that was never read. `stats` was moved off it on 2026-08-01 after a blank among six observations was measured moving a Pearson coefficient from 0.9992 to 0.9879. `quant` followed on 2026-09-05 (`fix-quant-legacy-numeric-input`): its last five call sites — `SharpeRatio`, `MaxDrawdown`, `AnnualizedReturn`, `DeflatedSharpeRatio`, and `PBO`'s column loop — now read through `numericSeries`, so the whole package refuses an unreadable cell instead of zeroing it. The remaining callers were left deliberately: they are display and reporting paths, where a fabricated zero shows up as a point on a chart rather than inside a coefficient.
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

### [2026-07-11] — dependencies held back by the Go 1.25 directive
- **Where**: `go.mod`
- **What**: the 2026-09-24 refresh left three groups below their newest versions. (1) The newest version of each of these declares `go 1.26`, so taking it would raise insyra's directive: `golang.org/x/crypto`, `exp`, `image`, `mod`, `net`, `oauth2`, `sync`, `sys`, `telemetry`, `term`, `text`, `time` and `tools`; `google.golang.org/api`; `google.golang.org/genproto` with its `googleapis/api` and `googleapis/rpc`; `github.com/googleapis/gax-go/v2`; `github.com/quic-go/quic-go`; `modernc.org/libc`; and `chromedp` from `v0.15.0` with the `cdproto` it needs. (2) `chromedp` stops at `v0.12.1`, below the newest Go-1.25 version `v0.14.2`, because from `v0.13.0` it requires `go-json-experiment/json`, and govulncheck panics on that package's generic variadics ("got jsontext.Value, want variadic parameter of unnamed slice or string type") when govulncheck itself is built with Go 1.25. Measured 2026-09-24: x/vuln v1.3.0 and v1.7.0 built with go1.25.14 both panic on it, and v1.7.0 built with go1.26.5 completes. The Vulnerability Scan job builds govulncheck with Go 1.25, so taking `v0.14.2` would break it. (3) `google.golang.org/grpc` stays at v1.83.2: v1.84.0 is inside GHSA-2v4p-qf9q-27wj, which is fixed on the newer line only in a `1.85.0-dev` pseudo-version.
- **Suggestion**: when the minimum Go rises to 1.26, take group (1) and the whole chromedp chain together, and move the Vulnerability Scan job to Go 1.26 in the same change. Take grpc past v1.83.2 at the first release outside every range of GHSA-2v4p-qf9q-27wj.
- **Status**: pending
