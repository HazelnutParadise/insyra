# Delivery Plan

## Current Phase
Phase 1 - Backend Discovery Kickoff

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
| M2 | Backend discovery and scoring frozen in spec | planning | in_progress | discoverer registry, auto-discovery, normalized scoring, and selection tests land; concrete backend adapters still pending |
| M3 | Columnar layout and cache model frozen in spec | planning | not_started | `add-accel-columnar-layout-cache` proposal/design/spec delta validated |
| M4 | Scheduler and observable fallback frozen in spec | planning | not_started | `add-accel-scheduler-multi-gpu` and `add-accel-observability-fallback` validated |
| M5 | CLI/DSL accel surface frozen in spec | planning | not_started | `add-accel-cli-dsl-surface` validated |

## Current Blockers
None.

## Next Verifiable Output
Concrete CUDA, Metal, and WebGPU discoverer stubs merged behind the new discovery contract, with report output still staying backend-agnostic.

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
- `add-accel-backend-discovery` is now partially implemented in code. The remaining work is concrete CUDA, Metal, and WebGPU discoverers plus richer device budget normalization.
- Do not move to CLI/DSL or scheduler work until the discovery change has real adapter stubs and stable report semantics.
