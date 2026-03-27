# Delivery Plan

## Current Phase
Phase 0 - Acceleration Convergence Kickoff

## Stage Objective
Establish the shared control surface and full OpenSpec proposal coverage for Insyra GPGPU acceleration so the next agent can pick up one accel change without re-planning the whole phase.

## Active Workstreams
- `M0`: convergence surface and handoff contract
- `M1`: runtime API and package boundary freeze for `insyra/accel`
- `M2-M4`: backend discovery, memory/cache, scheduler, and observability proposal freeze
- `M5`: CLI/DSL accel surface proposal freeze

## Milestones
| id | target | owner | status | verification_signal |
| --- | --- | --- | --- | --- |
| M0 | Control surface established | planning | done | `delivery-plan.md`, `AGENTS.md`, `CLAUDE.md`, and full accel proposal inventory exist |
| M1 | Accel runtime API frozen in spec | planning | not_started | `add-accel-runtime-capability` proposal/design/spec delta validated |
| M2 | Backend discovery and scoring frozen in spec | planning | not_started | `add-accel-backend-discovery` proposal/design/spec delta validated |
| M3 | Columnar layout and cache model frozen in spec | planning | not_started | `add-accel-columnar-layout-cache` proposal/design/spec delta validated |
| M4 | Scheduler and observable fallback frozen in spec | planning | not_started | `add-accel-scheduler-multi-gpu` and `add-accel-observability-fallback` validated |
| M5 | CLI/DSL accel surface frozen in spec | planning | not_started | `add-accel-cli-dsl-surface` validated |

## Current Blockers
None.

## Next Verifiable Output
Reviewed and approved `add-accel-runtime-capability` proposal package: `proposal.md`, `tasks.md`, `design.md`, and `specs/accel-runtime/spec.md`.

## Next OpenSpec Change
`add-accel-runtime-capability`

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
- `go.mod`
- `datalist.go`
- `datatable.go`
- `interfaces.go`
- `Docs/CCL.md`
- `openspec/specs/cli-entry/spec.md`
- `openspec/specs/command-registry/spec.md`
- `openspec/specs/dsl-commands/spec.md`

## Handoff Notes
- Phase 0 convergence setup is in place and the proposal inventory is complete enough for the next agent to start with one named change.
- Do not start runtime implementation outside the named next change. Proposal coverage is complete for the phase, but no accel runtime code exists yet.
- The next agent should review `add-accel-runtime-capability` first, because it freezes public API shape and package boundaries used by the rest of the phase.
