# Tasks: add-accel-multi-device-dispatch

Blocked on `add-accel-chunked-submission` (chunk seam) and `add-accel-device-selection` (eligible set). Do not start before both are archived or at least implemented and verified.

## 1. Strategy surface

- [x] 1.1 Add the shard strategy (`single` / `auto` / `forced`) to session configuration with `auto` as default; `auto`'s shard count = min(eligible devices, floor(total rows / recorded saturation point)), collapsing to `single` when that is 1.

## 2. Dispatch

- [x] 2.1 Replace single-device execution with per-assignment dispatch: one worker per assignment on its assigned eligible device, each internally using the chunk seam; merge by assignment input range under the existing `MergePolicy`, independent of completion order.
- [x] 2.2 Per-assignment failure: a failed assignment's rows re-run on the existing CPU path; other assignments stand; the result reports per-assignment placement, wall time, and any fallback.

## 3. Verification (all on this host)

- [x] 3.1 Stub-probe tests: multi-device plans form under fabricated device lists; `auto` respects the floor (below → single, above → sharded with every assignment ≥ floor); `forced` shards regardless.
- [x] 3.2 Parity: the same multi-assignment plan run concurrently and sequentially on the real single device produces byte-identical output equal to brute force (gated, unsandboxed shell per the recorded environment note). Unsandboxed M3/Metal run 2026-08-05: `TestMultiDeviceParityConcurrentAndSequentialOnHardware` PASS — sequential, concurrent, and `NearestExactCPU` all agree; note this needed only the single real device, not a multi-GPU host.
  Acceptance note: the test is written, but this sandbox has no real adapter and skips. Run it from the unsandboxed acceptance shell with `INSYRA_ACCEL_GPU_TESTS=1`; keep this checkbox unchecked until that run executes and passes.
- [x] 3.3 Failure-path unit tests: one assignment fails, its rows come back correct from the CPU, the report shows the per-assignment fallback.
- [x] 3.4 Full `go test ./accel/...` plus the race detector on the new concurrent path.

## 4. Docs, changelog, bookkeeping

- [x] 4.1 `Docs/accel.md`: strategies, placement reporting, and the honest statement that multi-GPU wall clock is unmeasured pending multi-GPU hardware (extend the standing follow-up in `AGENTS.md` accordingly). Changelog entries in both files stating correctness-verified / wall-clock-unmeasured.
- [x] 4.2 `delivery-status.md`: decision delta superseding the 2026-08-05 "stays parked" decision, with the library-consumer rationale and what remains unmeasured.
- [x] 4.3 `openspec validate add-accel-multi-device-dispatch --strict` passes.
