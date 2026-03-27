## Context
The runtime must handle single-device, multi-device, and CPU fallback paths without surprising users or forcing every operation onto a GPU.

## Goals / Non-Goals
- Goals:
  - define which workload classes are shardable in v1
  - define throughput-aware partitioning
  - define deterministic merge behavior
  - define strict GPU-only and auto fallback policy boundaries
- Non-Goals:
  - transparent fusion for every existing API
  - peer-merge optimization across every backend pair in v1

## Decisions
- Decision: v1 multi-device support applies only to shardable columnar workloads
  - Rationale: this keeps the execution model tractable while allowing real parallel speedup.
- Decision: partition by effective throughput and usable memory, not equal row count
  - Rationale: heterogeneous devices should not be treated as symmetric.
- Decision: default cross-backend merge happens on CPU
  - Rationale: CPU merge is the most portable deterministic baseline for v1.
- Decision: fallback is automatic but observable unless strict GPU mode is selected
  - Rationale: this preserves usability while keeping acceleration behavior inspectable.

## Risks / Trade-offs
- Risk: scheduler heuristics may choose GPU when CPU would be faster for small inputs.
  - Mitigation: require the planner to consider transfer and setup overhead.
- Risk: CPU merge could reduce gains for some heterogeneous workloads.
  - Mitigation: keep the merge contract explicit and optimize it later without changing semantics.
