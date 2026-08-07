# Design: add-accel-multi-device-dispatch

## Context

`PlanShardable` already produces weighted `ShardAssignment`s across shardable devices with a deterministic `MergePolicy`; the exact-nearest verification half merges per-row results on the CPU without caring which device proposed them, so cross-device correctness is structural. The chunk seam bounds any single submission; device selection defines the eligible set. What remains is dispatch and honesty about what this host can and cannot verify.

## Goals / Non-Goals

**Goals:**

- Assignments run concurrently, one worker per eligible device, each internally chunked; results merge deterministically regardless of device count, order, or timing.
- `auto` never shards into the fixed-cost region: every assignment stays at or above the recorded saturation point, or the plan collapses to `single`.
- Everything except multi-GPU wall clock is verified on this host; the wall-clock gap is documented, not papered over.

**Non-Goals:**

- No speedup claim for multi-GPU execution — no number exists until a multi-GPU host runs the recorded benchmark.
- No changes to `PlanShardable`'s weighting or the `MergePolicy` semantics.
- No new operations; exact-nearest remains the only shardable device operation.

## Decisions

1. **Dispatch is a worker per assignment, failure degrades per assignment.** An assignment whose device fails re-runs on the CPU path for its rows only (the code that already exists for untrustworthy rows), so a flaky second card cannot cost more than its own share — "a missing device is a performance event" survives sharding.
2. **`auto` shards by the measured floor.** Shard count = min(eligible devices, floor(total rows / saturation point)), so no assignment lands in the flat region where fixed costs dominate. Saturation points come from the recorded curve; if a future measurement moves them, the constant moves with its record.
3. **`forced` exists because the caller may know better.** A user benchmarking their own multi-GPU host needs to force sharding below the floor; the report shows the placement so they can see what they asked for.
4. **Determinism is by merge order, not execution order.** Assignments carry their input ranges; merge concatenates by range regardless of completion order. The sequential-parity test (same plan, assignments run one by one on this host's single device) must produce byte-identical output to concurrent execution.
5. **The verification split is stated in the spec.** Correctness scenarios run here (stub multi-device planning, forced multi-assignment parity, per-assignment fallback); the multi-GPU wall-clock scenario is written as requiring multi-GPU hardware, extending the standing Apple/Metal-only follow-up rather than inventing a new caveat mechanism.

## Risks / Trade-offs

- [Concurrent submissions contend on one queue when devices share a backend] → per-assignment wall time is part of the report; contention shows up as numbers, and `single` remains the default outcome below the floor.
- [Unverifiable-here speedup invites optimistic docs] → the doc text states the measured facts (saturation points, single-device curve) and the gap; the changelog entry says "correctness verified, multi-GPU wall clock unmeasured".
- [Partial failure complexity] → per-assignment CPU re-run reuses the existing fallback code path; the new surface is only the bookkeeping of which rows belonged to the failed assignment, covered by unit tests.

## Open Questions

- None blocking; blocked-on edges (chunking, selection) are recorded in the proposal.
