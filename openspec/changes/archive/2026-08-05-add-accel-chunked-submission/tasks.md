# Tasks: add-accel-chunked-submission

## 1. The bound

- [x] 1.1 Derive the chunk bound from the recorded saturation curve (both arms), with a stated margin under the readback timeout on the slowest measured arm; record the number and its derivation in this change. The bound is 16,000 rows: 8.065523s on the 100k×128 arm leaves 21.934477s, 73.1% of the 30s timeout; the 32k rung is flagged and reaches 27.01s in its worst recorded sample.

## 2. Mechanism

- [x] 2.1 Implement chunk split, sequential submission, and in-order merge at the exact-nearest execution seam, active only above the bound; below it the path is unchanged.
- [x] 2.2 Add the chunk count to the execution result surface without disturbing existing fields.

## 3. Verification

- [x] 3.1 Ungated unit tests: chunk boundary math (edges, remainders, bound±1), merge order, forced-chunking equality against the unchunked path with a stubbed executor.
- [x] 3.2 Gated hardware test: the 64k×128 rung that aborted in `measure-device-saturation` completes on the device and matches brute force exactly; record its wall time. Unsandboxed M3/Metal run 2026-08-05: completes in 32,188.225 ms across 4 chunks, `Accelerated`, indices and distances `DeepEqual` to `NearestExactCPU`.
- [x] 3.3 Gated overhead record: at a shape both paths can run, record chunked vs single-submission wall time so the per-chunk fixed cost is a number. Unsandboxed M3/Metal run 2026-08-05 at 32k×32: production chunked 2,671.404 ms vs single submission 2,425.096 ms — fixed-cost delta 193.089 ms (~8%) for one extra chunk.
- [x] 3.4 Full `go test ./accel/...` (sandboxed run may skip gated tests; run the gated ones from an unsandboxed shell per the recorded environment note). The sandboxed suite passed; gated hardware tests skipped because only a software adapter is visible.

Hardware note: the sandboxed shell exposes only a software GPU adapter, so the gated tests skip there; both were run and passed from an unsandboxed shell on the same M3 (numbers above).

## 4. Docs, changelog, bookkeeping

- [x] 4.1 `Docs/accel.md`: note that oversized submissions chunk instead of falling back. Changelog entries under `## Unreleased` in both `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] 4.2 `delivery-status.md` decision delta with the bound and the previously-aborting shape's new wall time. The wall time remains explicitly pending the required unsandboxed hardware run.
- [x] 4.3 `openspec validate add-accel-chunked-submission --strict` passes.
