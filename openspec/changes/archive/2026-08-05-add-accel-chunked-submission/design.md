# Design: add-accel-chunked-submission

## Context

A device submission's wall time grows with the parallel axis once past the flat region — the recorded curve reaches ~9.2s at 128k rows on the light arm, and the heavy arm's 64k rung exceeded the readback timeout entirely and fell back. The rows are independent by construction (the CPU verification half already merges per-row results), so a submission can be split along that axis without touching the precision contract.

## Goals / Non-Goals

**Goals:**

- No eligible shape loses the device to `readback-timeout`: oversized submissions run as several bounded ones.
- Bit-identical results, chunked or not, asserted against brute force on hardware.
- A chunk seam (bounded per-submission work, sequential submit, deterministic merge) that the multi-device dispatch change can reuse unchanged.

**Non-Goals:**

- No concurrency across chunks (one device, sequential) — concurrency arrives with multi-device dispatch.
- No change to eligibility gates, profitability floors, or the CPU decision half.
- No public API change.

## Decisions

1. **Chunk along the axis the kernel parallelizes over.** Rows on that axis are independent; the merge is concatenation in input order, so determinism is structural rather than argued.
2. **The bound comes from the curve, not from a guess.** The recorded saturation measurements give wall time per rung on both arms; the bound is set so a chunk's expected wall time keeps a comfortable margin under the readback timeout on the slowest measured arm, and the chosen number plus its derivation is recorded in the change. If implementation finds the timeout is configurable instead, the bound still stands — a smaller submission also bounds retry cost and memory.
3. **Chunking activates only above the bound.** Below it, the path is byte-for-byte today's single submission — no new overhead where nothing was wrong.
4. **Fixed costs are re-paid per chunk, and that is accepted and measured.** The flat region of the curve (~465ms light arm) is the per-submission fixed cost; the verification task records total chunked wall time against the single-submission time at a shape both can run, so the overhead is a number, not a hope.

### Recorded bound derivation

The backend readback timeout is 30 seconds (`accel/internal/wgpu/wgpu.go`). The archived saturation curve reaches 8.065523 seconds at 16,000 rows on the slowest runnable arm, 100,000 training rows × 128 dimensions. The next 32,000-row rung is flagged for a 20.745600-second best run and a 27.01-second observed run, leaving only about 10% of the timeout in the worst recorded sample. The 16,000-row bound therefore preserves 30 - 8.065523 = **21.934477 seconds**, or about **73.1%**, of the timeout as margin on that arm. The 32-dimension arm is 1.321645 seconds at the same rung, so it does not determine the bound. A 64,000-row submission consequently runs as four sequential chunks.

## Risks / Trade-offs

- [A chunk still times out] → the bound is chosen with margin from the slowest measured arm; a chunk that fails falls back exactly as a whole submission does today — the failure mode shrinks, it does not grow.
- [Per-chunk fixed cost erodes the win on shapes just above the bound] → measured and recorded; the bound placement trades a few fixed costs against losing the device entirely, and the previously-aborting shape is the proof either way.
- [Merge bugs] → ungated unit tests drive the chunk math and merge with a stubbed executor; the gated hardware test asserts equality with brute force.

## Open Questions

- None blocking; the bound's exact value is deliberately left to the measurement task.
