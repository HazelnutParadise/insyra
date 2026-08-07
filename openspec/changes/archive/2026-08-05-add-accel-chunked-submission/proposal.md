# Proposal: add-accel-chunked-submission

## Why

The saturation measurement found a shape the device loses outright: at 100k×128 with 64k test rows, the single submission dies in `readback-timeout` (`FallbackReasonReadbackTimeout`) and the whole call falls back to CPU — the largest shapes lose the device exactly where it pays most. Bounding submission size fixes that, and the chunk seam it creates is the substrate the multi-device dispatch change will reuse.

## What Changes

- Exact-nearest device submissions larger than a measured bound are split along the parallel axis into sequential bounded chunks on the same device; per-chunk results merge into the same answer, and the CPU decision half is untouched.
- The bound is derived from the recorded saturation curve (wall time per rung), chosen with margin against the readback timeout, and recorded — not guessed.
- Verifiable output: the 64k×128 rung that aborted in `measure-device-saturation` completes on the device with results bit-identical to brute force.
- Observability gains the chunk count; everything else in the result surface is unchanged.
- Changelog entries in both languages (large shapes now stay on the device instead of silently falling back).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `accel-gpu-execution`: submissions exceeding a measured size bound SHALL be chunked and merged bit-identically instead of timing out and losing the device.

## Impact

- **Code**: the exact-nearest execution path in `accel/` (submission construction and readback); no public API change.
- **Behavior**: results identical; shapes that previously fell back now complete on the device.
- **Docs**: `Docs/accel.md` note, `CHANGELOG.md` / `CHANGELOG_TW.md`.
- **Dependencies**: none.
