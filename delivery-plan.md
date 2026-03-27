# Delivery Plan

## Current Phase
Phase 1 - Backend Capability Normalization Kickoff

## Stage Objective
Build backend discovery and device selection on top of the now-frozen `insyra/accel` runtime surface without expanding scope beyond the named change.

## Active Workstreams
- `M0`: convergence surface and handoff contract
- `M1`: runtime API and package boundary freeze for `insyra/accel`
- `M2-M4`: backend discovery, memory/cache, scheduler, and observability proposal freeze
- `M5`: CLI/DSL accel surface proposal freeze

## Milestones
| id | target | owner | status | verification_signal |
| --- | --- | --- | --- | --- |
| M0 | Control surface established | planning | done | `delivery-plan.md`, `AGENTS.md`, `CLAUDE.md`, and full accel proposal inventory exist |
| M1 | Accel runtime API frozen in spec | planning | done | `accel` package compiles, `go test ./accel` passes, docs surface added |
| M2 | Backend discovery and scoring frozen in spec | planning | in_progress | discoverer registry, builtin CUDA/Metal/WebGPU stubs, native probe seams, discovery timeout handling, cross-backend dedupe, shared-memory budget fallback, budget normalization, and selection tests land; true SDK-backed probes and deeper capability shaping still pending |
| M3 | Columnar layout and cache model frozen in spec | planning | done | typed projection now emits validity bitmaps, encoded string transport, lineage-aware session-local cache keys, and aggregate budget enforcement; true device residency is still pending allocator-backed work |
| M4 | Scheduler and observable fallback frozen in spec | planning | in_progress | strict-mode fallback reason codes, core accel metrics, weighted shard assignments, merge-policy selection, and profitability-aware planning land; true execution merge behavior still pending |
| M5 | CLI/DSL accel surface frozen in spec | planning | in_progress | `accel devices|cache|run`, `config accel.mode`, and `show accel.devices|accel.cache` land; full cache/runtime execution semantics still pending |

## Current Blockers
None.

## Next Verifiable Output
True SDK-backed backend probes layered onto the new native probe seam, so env-driven stubs become optional rather than the primary discovery path.

## Next OpenSpec Change
`add-accel-backend-discovery`

## Decision Log
- decision: Keep acceleration in optional `insyra/accel` packages rather than core `insyra`.
  rationale: Preserve pure-Go default ergonomics and isolate native/runtime dependencies behind explicit opt-in.
  timestamp: 2026-03-27
  impacted_change_ids: `add-accel-runtime-capability`, `add-accel-backend-discovery`, `add-accel-cli-dsl-surface`
- decision: Use `CUDA + Metal + WebGPU native` as the backend strategy and do not use ROCm as the AMD iGPU v1 primary path.
  rationale: This covers NVIDIA, Apple, Intel, and AMD integrated/shared-memory devices with a portable fallback route.
  timestamp: 2026-03-27
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-columnar-layout-cache`, `add-accel-scheduler-multi-gpu`
- decision: Support heterogeneous multi-GPU only for shardable columnar operations in v1.
  rationale: It keeps v1 implementable while preserving a path toward more transparent fusion later.
  timestamp: 2026-03-27
  impacted_change_ids: `add-accel-scheduler-multi-gpu`, `add-accel-observability-fallback`
- decision: Default to observable CPU fallback, with strict GPU-only mode as an explicit opt-in.
  rationale: This balances usability with debuggability and makes backend selection visible to users.
  timestamp: 2026-03-27
  impacted_change_ids: `add-accel-scheduler-multi-gpu`, `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Treat full GPU string kernels as a Phase 2 slice.
  rationale: Phase 1 needs typed columnar transport and encoded-string eligibility, but full string-kernel parity should not block runtime convergence.
  timestamp: 2026-03-27
  impacted_change_ids: `add-accel-columnar-layout-cache`, `add-accel-string-kernels-phase-2`
- decision: Start implementation by freezing the public accel runtime and typed CPU-side projection before backend work.
  rationale: Backend discovery, cache, scheduler, and CLI all depend on one stable runtime contract.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-runtime-capability`, `add-accel-backend-discovery`
- decision: Use a discoverer registry plus `Open()` auto-discovery as the first backend-discovery implementation slice.
  rationale: This keeps real adapters pluggable while making session behavior testable before native bindings exist.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-backend-discovery`
- decision: Ship env-driven builtin CUDA, Metal, and WebGPU adapter stubs before native SDK probing.
  rationale: This creates stable backend boundaries and lets report, scheduler, and CLI work be verified on machines without GPU SDKs.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Land the first accel CLI/DSL surface as probe/report commands before true cache or workload execution exists.
  rationale: This gives users and future agents a stable inspection path without pretending the runtime is already complete.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-cli-dsl-surface`, `add-accel-observability-fallback`
- decision: Return a session alongside strict-gpu and discovery errors so report surfaces remain inspectable on failure.
  rationale: Observable fallback is part of the contract; callers and CLI should still be able to inspect reason codes and metrics when acceleration cannot proceed.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Introduce a shardable multi-device planning surface before true execution scheduling.
  rationale: This establishes the contract for heterogeneous device aggregation and total budget reporting without claiming weighted partitioning or merge execution already exists.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-scheduler-multi-gpu`, `add-accel-cli-dsl-surface`
- decision: Start cache implementation as a session-local resident index fed by typed projection, before adding true device allocators or eviction.
  rationale: This gives the CLI and report surface truthful cache state now, while preserving a clean seam for later VRAM/shared-memory backends.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-columnar-layout-cache`, `add-accel-cli-dsl-surface`
- decision: Enforce cache budgets at the session-local cache layer before introducing backend-native allocators.
  rationale: This makes cache state actionable now and proves eviction/report semantics before native VRAM/shared-memory plumbing is added.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-columnar-layout-cache`, `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Complete the columnar/cache change by adding validity bitmaps and encoded string transport before returning to backend work.
  rationale: Cache/accounting alone was not enough to claim the memory/layout change complete; string/null transport needed to be allocator-ready first.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-columnar-layout-cache`
- decision: Add native probe seams and normalized capability maps before attempting real SDK bindings.
  rationale: Binding to CUDA/Metal/WebGPU without a stable seam would couple probe failures, report semantics, and CLI output too tightly to backend-specific code.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Honor discovery timeout and shared-memory budget fallback inside the runtime before treating backend discovery as converged.
  rationale: A public timeout field and shared-memory budget policy are not credible if they only exist in config shape but not in behavior.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-columnar-layout-cache`, `add-accel-observability-fallback`
- decision: Repair accel CLI/DSL spec text and Cobra flag parsing before further backend work.
  rationale: Broken spec text and a non-functional `--mode` path would make the control surface look complete while failing in actual use.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-cli-dsl-surface`, `add-accel-backend-discovery`
- decision: Move multi-device scheduling from aggregate-only planning to workload-aware weighted partition planning before attempting allocator-backed execution.
  rationale: A multi-GPU surface that only sums budget is not enough to validate heterogenous scheduling semantics or strict/auto profitability behavior.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-scheduler-multi-gpu`, `add-accel-cli-dsl-surface`, `add-accel-observability-fallback`

## Source Links
- `delivery-plan.md`
- `AGENTS.md`
- `CLAUDE.md`
- `openspec/changes/add-accel-convergence-surface/`
- `openspec/changes/add-accel-runtime-capability/`
- `openspec/changes/add-accel-backend-discovery/`
- `openspec/changes/add-accel-columnar-layout-cache/`
- `openspec/changes/add-accel-scheduler-multi-gpu/`
- `openspec/changes/add-accel-cli-dsl-surface/`
- `openspec/changes/add-accel-observability-fallback/`
- `openspec/changes/add-accel-string-kernels-phase-2/`
- `Docs/accel.md`
- `README.md`
- `Docs/README.md`
- `go.mod`
- `datalist.go`
- `datatable.go`
- `interfaces.go`
- `Docs/CCL.md`
- `openspec/specs/cli-entry/spec.md`
- `openspec/specs/command-registry/spec.md`
- `openspec/specs/dsl-commands/spec.md`

## Handoff Notes
- The convergence surface and runtime capability are both in place. `accel` now exists as a compilable opt-in package with `Open/NewSession`, typed projection helpers, and report/device/dataset/buffer surface.
- Use a fresh `GOCACHE` when running Go validation in this environment. The default cache path hit a local toolchain/cache issue after `go clean -cache`, but tests pass with a clean alternate cache directory.
- `add-accel-backend-discovery` is now materially deeper in code. Builtin stubs, native probe seams, normalized capability flags, budget normalization, probe-source reporting, and CLI/report capability visibility are in place. The remaining gap is true SDK-backed probing and any backend-specific capability enrichment that comes with it.
- `add-accel-backend-discovery` now also honors `DiscoveryTimeout`, supports host-memory-derived shared-memory budgets when native budget data is missing, and avoids the earlier gap where native probe tests and config fields existed without working code behind them.
- `add-accel-observability-fallback` now has code behind it: stable fallback reason codes, strict-gpu failure reports, discovery-error reporting, and core metrics are wired into `accel.Report` and CLI output.
- `add-accel-scheduler-multi-gpu` now has a real planning contract in code: `PlanShardable()` / `PlanShardableWorkload(...)` produce weighted per-device assignments, deterministic merge-policy selection, and profitability-aware fallback for auto mode. True execution merge paths and allocator-backed dispatch still do not exist.
- `add-accel-columnar-layout-cache` is now complete enough to close the current slice: typed projection emits validity bitmaps, string offsets/data transport, lineage-aware session-local cache identity, aggregate budget enforcement, eviction metrics, and truthful cache output that does not pretend projection buffers are already resident on every shardable device.
- `add-accel-cli-dsl-surface` remains partially implemented. `accel cache` now shows truthful session-local resident state, the Cobra `--mode` path now works end-to-end, and the broken change-local spec text was repaired; there is still no device allocator, eviction policy, or true workload execution surface.
- The next change to pick up is `add-accel-backend-discovery`, focusing on richer capability normalization and eventually replacing env-driven stubs with native probe seams.
