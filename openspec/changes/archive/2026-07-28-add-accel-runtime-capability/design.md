## Context
Insyra currently exposes CPU-first `DataList` and `DataTable` APIs backed by `[]any` and actor-serialized mutation. Acceleration needs a package boundary that preserves source compatibility while making room for typed, backend-specific execution.

## Goals / Non-Goals
- Goals:
  - keep accel opt-in under `insyra/accel`
  - define a runtime surface that later backend, cache, scheduler, and CLI changes can share
  - avoid implicit GPU compilation of arbitrary Go closures
- Non-Goals:
  - implementing any backend
  - changing core `insyra` data structures in this change
  - promising universal GPU eligibility for all existing APIs

## Decisions
- Decision: expose session-scoped execution under `accel.Session`
  - Rationale: device discovery, cache ownership, execution reports, and fallback policy need session-level state.
- Decision: represent GPU-eligible inputs as `accel.Dataset` and `accel.Buffer`
  - Rationale: these types separate typed columnar execution from existing `[]any` containers.
- Decision: include `accel.Report` in the public surface
  - Rationale: observable fallback and backend choice are part of the v1 contract, not debug-only internals.

## Risks / Trade-offs
- Risk: too much API detail too early could block implementation experiments.
  - Mitigation: freeze only shape and ownership boundaries, not backend-specific tuning knobs.
- Risk: users may expect `DataList.Map` and `DataTable.Map` to become GPU-transparent.
  - Mitigation: state explicitly that generic Go closures remain CPU unless re-expressed through eligible accel surfaces such as CCL and typed built-ins.
