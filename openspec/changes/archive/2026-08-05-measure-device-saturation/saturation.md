# Saturation Measurement Record

Apple M3, Metal, 2026-08-05. Best of 5 per rung, upload+dispatch+readback included, one process invocation per rung (`INSYRA_ACCEL_SATURATION_ARM` / `INSYRA_ACCEL_SATURATION_RUNG`), via `BenchmarkDeviceSaturation` with `INSYRA_ACCEL_GPU_TESTS=1`.

Environment note for future runs: a sandboxed worker sees only a software adapter and the benchmark skips — the first attempt at this measurement hit exactly that and recorded a blocker here. The curve below was produced from an unsandboxed shell on the same machine, where the backend probe returns the real Metal adapter. The `delivery-status.md` hardware-blocker note describes the same condition.

## Arm: train=100,000 × dims=32

| test_rows | best (ms) | ratio vs previous rung |
| ---: | ---: | ---: |
| 1,000 | 465.587 | — |
| 2,000 | 463.243 | 0.99 |
| 4,000 | 460.608 | 0.99 |
| 8,000 | 791.962 | 1.72 |
| 16,000 | 1,321.645 | 1.67 |
| 32,000 | 2,408.774 | **1.82 — saturation point** |
| 64,000 | 4,672.756 | 1.94 |
| 128,000 | 9,231.046 | 1.98 |

Flat through 4k (the region the prior ~467ms readings sampled), sub-proportional growth from 8k, and the pre-declared criterion (first rung ≥1.8x its predecessor) trips at **32k**, after which doubling converges to ~2.0x — the device is fully saturated.

## Arm: train=100,000 × dims=128

| test_rows | best (ms) | ratio vs previous rung |
| ---: | ---: | ---: |
| 1,000 | 1,765.353 | — |
| 2,000 | 1,810.291 | 1.03 |
| 4,000 | 2,059.117 | 1.14 |
| 8,000 | 4,284.642 | **2.08 — saturation point** |
| 16,000 | 8,065.523 | 1.88 |
| 32,000 | 20,745.600 | 2.57 — flagged, excluded from criterion |
| 64,000 | — | aborted: `device fallback: readback-timeout` |

The heavier arm saturates earlier, at **8k**, as the design predicted. The 32k rung is flagged: its run-to-run spread (20.75s–27.01s, ~30%) is inconsistent with every earlier rung (≤5%), consistent with thermal or memory pressure, so it is recorded but excluded from the criterion. The 64k rung aborted inside the device path itself — the readback timed out and the operation fell back — which is itself a finding: very large single-device submissions on this shape fail before memory does, and a shard split would shrink per-device size rather than grow it.

## Result under the pre-declared criterion

- dims=32: saturation begins at **32,000 test rows**; below it, wall time is dominated by fixed costs a second device cannot share.
- dims=128: saturation begins at **8,000 test rows**; the arm stops being runnable at 64k on a single device.

Saturating shapes exist and are reachable — both are within the KNN bridge's eligibility gates (per-row work ≥2048, test rows ≥2048). Whether a real workload occupies that region is a separate question the executor work should wait for; the seam remains `executionDevice` → per-assignment dispatch.
