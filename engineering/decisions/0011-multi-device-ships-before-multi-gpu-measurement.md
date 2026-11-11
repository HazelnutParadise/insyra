# 0011 — Multi-device execution ships before a multi-GPU host has measured it

**Status:** accepted
**Date:** 2026-08-05

## Context
The single-device saturation sweep on the M3 (exact-nearest, best of 5, upload, dispatch and readback included, one process per rung) is flat only up to 4k test rows at 100k×32, then bends: the 1.8x-per-doubling criterion trips at 32k rows (1.82x, converging to 1.98x by 128k). The heavier 100k×128 arm trips at 8k rows (2.08x) and stops being runnable at 64k, where one submission dies in `readback-timeout` and falls back. The curve is in [saturation.md](/openspec/changes/archive/2026-08-05-measure-device-saturation/saturation.md). The first reading of it parked multi-device work until a real workload was seen in the saturated region.

## Decision
Multi-device execution was built the same day as three changes: chunked submission, device selection, then dispatch. Dispatch defaults to `auto`, which shards only when every assignment stays at or above the measured floors of 32k rows (32 dimensions) and 8k rows (128 dimensions); `single` and `forced` are explicit choices. One submission is bounded at 16,000 rows, and larger inputs run in sequential chunks.

## Options rejected
- Park it until a workload is observed. A library cannot observe its users' workloads, users with multi-GPU hosts exist regardless of the maintainer's hardware, and the measured floors and the 64k failure show real shapes it serves. Decided by the project owner on 2026-08-05.
- Bound a submission at 32k rows. Its worst recorded sample reached 27.01 s against the 30 s readback timeout. At 16k the slowest arm took 8.07 s, a 73.1% margin.

## Consequences
Correctness is verifiable on a single-device host: sequential, concurrent and brute-force results are bit-identical. Multi-GPU wall clock stays unmeasured until a host with several real devices runs the gated test, tracked in the `AGENTS.md` Follow-ups.
