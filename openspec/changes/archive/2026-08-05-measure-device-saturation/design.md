# Design: measure-device-saturation

## Context

`accel/planner.go` produces capability-weighted multi-device shard assignments; `accel/executor.go:136` (`executionDevice`) executes the whole operation on the single device with the largest share. The recorded rationale for not building per-assignment dispatch is that no workload has been shown to saturate one device — and the only measurement touching the question (KNN true-direction, 100k×32) showed device wall time flat at ~467ms across 1k/2k/4k test rows. Flat wall time means the device is dominated by fixed costs (upload, dispatch, readback, underutilized occupancy), and splitting a fixed cost across two cards buys nothing.

Saturation has a measurable signature: past it, doubling the parallel axis doubles wall time. The transition from flat to proportional is the number the multi-device decision needs.

## Goals / Non-Goals

**Goals:**

- Locate the flat→scaling transition of the exact-nearest device operation on the M3, on the same shape as the prior measurement plus one heavier per-row shape.
- Record the curve and the decision it implies, in the change and in `delivery-status.md`.

**Non-Goals:**

- No executor changes, no per-assignment dispatch, no new kernels, no threshold retuning in `accel/exact.go`.
- No claim about other hardware — the curve is a property of the device it ran on, recorded as such.
- No CPU-vs-device profitability re-measurement; this is about the device's own scaling, not about who wins.

## Decisions

1. **Sweep the axis the kernel parallelizes over.** Test rows double per rung — 1k, 2k, 4k, 8k, 16k, 32k, 64k, 128k — on fixed 100k×32 training data, because that is the axis a shard plan would split and the shape the prior flat readings came from. A second arm at a heavier per-row cost (100k×128) checks whether saturation arrives earlier when each row carries more arithmetic; if even that stays flat to the top rung, the parked decision is safe everywhere measured.
2. **Saturation criterion declared before running.** A rung is "scaling" when its wall time is ≥1.8x the previous rung's (doubling the work bought ≥90% of proportional cost). The transition point is the first scaling rung; everything below it is fixed-cost territory where a second device has nothing to do. The criterion is written down here so the measurement cannot be read to taste afterward.
3. **Measure like every prior accel number.** Best of 5, upload+dispatch+readback included, `INSYRA_ACCEL_GPU_TESTS=1` gate, software adapters skip — consistent with the M17/KNN measurement discipline so the numbers compose.
4. **The output is a decision, not just a table.** The recorded delta must say one of two things: "saturation begins at N test rows on this shape — a shard split pays above N, and the executor seam is `executionDevice` → per-assignment dispatch", or "no measured shape saturates — multi-device stays parked, reopen when a workload reaches N rows". Either way the `AGENTS.md` follow-up stops saying the measurement doesn't exist.

## Risks / Trade-offs

- [Memory ceiling before saturation: 128k test rows × 100k train candidates may exhaust device or host memory before the curve bends] → the harness stops at the largest rung that fits, records the abort reason, and the decision is stated for the measured range only.
- [Thermal throttling on a laptop-class M3 fakes a bend in the curve] → best-of-5 with the existing measurement discipline; if late rungs show run-to-run spread inconsistent with earlier rungs, the spread is recorded and the rung excluded from the criterion.
- [One machine] → the curve is recorded as Apple/Metal-specific, same as every other accel number; the standing non-Apple follow-up already covers the generalization gap.

## Open Questions

- None. The criterion, shapes, and outputs are fixed above; the only unknown is the number the hardware returns.
