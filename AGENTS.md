# AGENTS.md

This file is the project operating contract for Insyra acceleration planning and execution.

## Required Entry Sequence
- Read `delivery-plan.md` before doing any accel-related work.
- Use `delivery-plan.md` as the source of truth for current phase, blockers, next verifiable output, and next OpenSpec change.
- Read the named OpenSpec change before proposing implementation or writing code.

## Required Artifacts
- `delivery-plan.md` is the shared progress and handoff surface.
- `openspec/changes/` holds the executable units of work.
- `CLAUDE.md` is only a bootstrap pointer back to this file.

## Planning Discipline
- The accel phase may not use umbrella proposals. One change must produce one verifiable result.
- Do not start implementation for uncovered accel scope. Missing proposal coverage means the work is out of bounds.
- Keep Phase 1 and Phase 2 separate. Full GPU string kernels remain a Phase 2 track unless the delivery plan explicitly changes.
- Preserve the fixed architecture defaults unless a new decision is logged in `delivery-plan.md`:
  - optional `insyra/accel` package family
  - `CUDA + Metal + WebGPU native`
  - heterogeneous multi-GPU only for shardable columnar operations in v1
  - observable CPU fallback by default, strict GPU-only as opt-in

## Update Discipline
- Update `delivery-plan.md` after every milestone, blocker, or handoff.
- Change the named next OpenSpec change when the recommended pickup point changes.
- Update this file only when operating rules change.
- Keep `CLAUDE.md` minimal unless its bootstrap pointer becomes wrong.

## OpenSpec Rules
- Every active accel stage item must map to one OpenSpec change.
- Every OpenSpec change must map to one milestone and one verifiable output.
- Validate changed proposals with `openspec validate <change-id> --strict` before handoff.
- Do not merge unrelated capability slices into one change.

## Handoff Requirements
Every accel handoff must include:
- current phase
- blocker status
- next verifiable output
- next OpenSpec change
- decision delta since previous handoff
- source links for critical context
- whether `delivery-plan.md` changed
- whether `AGENTS.md` changed

## Implementation Constraints
- Do not silently reinterpret existing `DataList.Map(func...)` or `DataTable.Map(func...)` as GPU kernels.
- Keep accel runtime opt-in and package-scoped until the relevant OpenSpec changes are implemented and approved.
- Treat Apple shared-memory residency separately from discrete VRAM in specs and docs.
- Keep CLI/DSL exposure aligned with the named accel change; do not implement commands outside validated proposal scope.
