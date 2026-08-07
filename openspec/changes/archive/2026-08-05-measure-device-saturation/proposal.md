# Proposal: measure-device-saturation

## Why

The planner already produces heterogeneous multi-device shard plans, but the executor runs every operation on the single device carrying the largest share — and the standing `AGENTS.md` follow-up says the first honest step toward multi-device execution is a saturation measurement, because the KNN true-direction numbers point the wrong way: device wall time was nearly flat in test rows (~467ms at 1k/2k/4k on 100k×32), meaning the single M3 was not saturated and a second card would have had nothing to do. Nobody has measured where saturation actually begins, so the multi-device question currently has no number to be decided on.

## What Changes

- A gated benchmark sweeps the one shardable device operation (exact-nearest) up the test-row axis until device wall time stops being flat and starts scaling, on a fixed per-row workload (100k×32, matching the prior measurement) and one heavier per-row shape. Upload, dispatch, and readback are included, best-of-5, as all accel measurements are.
- The measured curve and the flat→scaling transition point are recorded in the change and in `delivery-status.md` as a decision delta: either a workload shape exists where a second device would have work (with the number), or multi-device execution stays parked (with the number that would reopen it).
- The `AGENTS.md` multi-GPU follow-up is updated from "no saturation measurement exists" to the measured answer.
- No wiring, no kernels, no executor changes — measurement only. One change, one verifiable output: the recorded saturation curve.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `accel-scheduler`: multi-device execution SHALL be gated on a measured single-device saturation point, not assumed from the existence of shard plans.

## Impact

- **Code**: one gated benchmark under `accel/` (runs with `INSYRA_ACCEL_GPU_TESTS=1` on device hardware); no production paths touched.
- **Docs/records**: `delivery-status.md` decision log, `AGENTS.md` follow-up entry. No changelog — nothing user-visible changes.
- **Dependencies**: none.
