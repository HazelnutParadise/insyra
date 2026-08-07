# Tasks: measure-device-saturation

## 1. Harness

- [x] 1.1 Add a gated saturation benchmark under `accel/` for the exact-nearest operation: test-row rungs 1k → 128k doubling, fixed 100k×32 training arm plus a 100k×128 heavier arm, best of 5, upload+dispatch+readback included, gated on `INSYRA_ACCEL_GPU_TESTS=1`, skipping cleanly on software adapters and recording the abort reason if a rung exceeds device or host memory.

## 2. Measurement

- [x] 2.1 Run both arms on the M3 and record the full curve per rung in this change (a `saturation.md` next to this file), flagging any rung whose run-to-run spread is inconsistent with earlier rungs.
- [x] 2.2 Apply the pre-declared criterion (first rung ≥1.8x its predecessor) and state the result: the saturation point per arm, or its absence across the measured range.

## 3. Decision on record

- [x] 3.1 Add the decision delta to `delivery-status.md`: the curve summary and either "shard split pays above N rows; seam is `executionDevice` → per-assignment dispatch" or "multi-device stays parked; reopen at the shape that saturates".
- [x] 3.2 Update the `AGENTS.md` multi-GPU follow-up entry so it carries the measured answer instead of pointing at a measurement that did not exist.

## 4. Bookkeeping

- [x] 4.1 No changelog entry (nothing user-visible); confirm no production code path changed — the diff is one gated benchmark plus records.
- [x] 4.2 `openspec validate measure-device-saturation --strict` passes before handoff.
