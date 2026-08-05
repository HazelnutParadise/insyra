# Proposal: add-accel-multi-device-dispatch

## Why

The v1 architecture default — "heterogeneous multi-GPU only for shardable columnar operations" — has always described the planner, not the executor: `PlanShardable` builds capability-weighted multi-device assignments and `executionDevice` then runs everything on one card. Two things changed. The saturation measurement put numbers on when a second device would have real work (the ≥1.8x criterion trips at 32k test rows on the light arm, 8k on the heavy one, and a 64k heavy-arm submission cannot run on one device at all). And the earlier "wait until a workload shows up" parking was application logic misapplied to a library: insyra cannot observe its users' workloads, and users with multi-GPU hosts exist whether or not the maintainer's machine has two cards. This change supersedes the parked decision.

## What Changes

- `executionDevice` becomes per-assignment dispatch: a shard plan's assignments execute concurrently across the eligible devices (from `add-accel-device-selection`), each assignment bounded by the chunk seam (from `add-accel-chunked-submission`), merged under the existing deterministic `MergePolicy`.
- A shard strategy axis with three values: `single` (today's behavior), `auto` (shard only when every resulting assignment stays at or above the recorded saturation point — splitting below it pays fixed costs for nothing, by measurement), and `forced` (the caller asked; comply and report).
- Per-assignment observability: the execution result names which device ran which share.
- Correctness is fully verified on this single-device host: multi-assignment plans execute their assignments and merge, asserted bit-identical to single-device brute force; stub probes exercise multi-device planning. What cannot be produced here is a multi-GPU wall-clock number — the docs say so plainly, and the standing hardware-coverage follow-up extends to cover it. No speedup is claimed unmeasured.
- Docs and both changelogs.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `accel-scheduler`: shard plans SHALL execute across devices under an explicit strategy, gated by the measured saturation floor in `auto`, bit-identical always, with per-assignment placement observable.

## Impact

- **Code**: `accel/executor.go` (dispatch), session config (strategy), execution result surface (per-assignment placement), planner consumption unchanged.
- **Behavior**: default (`auto`) changes nothing below the saturation floor and on single-device hosts.
- **Docs**: `Docs/accel.md` including the honest multi-GPU measurement gap; `CHANGELOG.md` / `CHANGELOG_TW.md`.
- **Dependencies**: none; blocked on `add-accel-chunked-submission` and `add-accel-device-selection`.
