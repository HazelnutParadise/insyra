## Context
Current `DataList` and `DataTable` store values as `any`, which is convenient for CPU workflows but unsuitable for GPU kernels and device caches. The accel layer needs a typed representation that can be derived from core structures without mutating them.

## Goals / Non-Goals
- Goals:
  - define typed numeric and boolean buffers
  - define nullability via validity bitmaps
  - define string transport via offsets and optional dictionary encoding
  - define cache budgeting for discrete VRAM and shared-memory devices
- Non-Goals:
  - implementing final string-kernel parity in Phase 1
  - changing the public shape of core `DataTable` or `DataList`

## Decisions
- Decision: project core data into accel-owned typed columns
  - Rationale: this keeps core containers stable and gives the accel layer full control over layout.
- Decision: use validity bitmaps for nullable columns
  - Rationale: null semantics must stay consistent across CPU and GPU execution.
- Decision: treat Apple and APU memory as working-set-managed shared residency, not fake discrete VRAM
  - Rationale: docs and budgets must match the actual memory model.
- Decision: support encoded-string transport in Phase 1
  - Rationale: equality, filtering, and key-based operations need string eligibility even before full string function kernels land.

## Risks / Trade-offs
- Risk: typed projection adds conversion cost for small workloads.
  - Mitigation: later scheduler policy can keep small jobs on CPU.
- Risk: aggressive cache reuse may hide stale data.
  - Mitigation: cache keys must include dataset fingerprint, layout, and operation lineage.
