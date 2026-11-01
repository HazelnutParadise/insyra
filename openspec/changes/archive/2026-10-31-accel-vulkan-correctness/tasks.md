# Tasks

- [x] `ieee_decode` uses `31 - firstLeadingBit`; the exp, log, division and square root device tests pass on an NVIDIA GT 1030 over Vulkan.
- [x] Resident uploads and writes settle the queue off Metal; `Zero` then read passed 60 of 60 runs (3 of 20 failed before).
- [x] Hardware tests scale their queries off Apple and skip when the backend has no executor.
- [x] Shortlist smoke test compares to a few ulps.
- [x] `Docs/accel.md` warns against `countLeadingZeros` in a user kernel.
- [x] Entries under `## Unreleased` / `### `accel`` in both CHANGELOGs.
- [x] `openspec validate accel-vulkan-correctness --strict` passes (CI, OpenSpec strict validation on the PR).
