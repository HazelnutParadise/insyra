# Change: Wire dl's large MatMul to the device behind an opt-in bridge

## Why

The measurement change closed positively: on the M3 against the all-core CPU
baseline, single-dispatch 2-D matmul wins 8.9x at [4096,256]×[256,256], 11.7x
and 13.2x at the FFN shapes, and 52x at [4096,4096]² — with maxULP=0 on every
shape, because the prototype's per-output thread accumulates along k in the
CPU's serial order. The same measurement refused the attention-shaped batched
products: driven as 128 separate dispatches they lose 1.08x–2.08x, so they are
out of scope until a single-dispatch batched kernel is measured.

`dl` must stay dependency-free by default — a direct `accel` import would drag
the wgpu toolchain into every consumer's build. The KNN wiring already solved
this shape of problem: a hook in the host package, nil by default, filled by
an opt-in blank-import bridge (`accel/knnbridge` → `stats`). This change
repeats that pattern for `dl`.

## What Changes

- `dl` gains a package-level device-matmul hook, nil by default. When nil,
  nothing changes — the parallel CPU path runs as today. The hook is consulted
  only for 2-D (non-batched) f32 matmuls at or above a measured MAC floor.
- A new `accel/dlbridge` package registers the device implementation in its
  `init`, mirroring `accel/knnbridge`. Blank-importing it is the only way the
  device path activates.
- The prototype WGSL matmul is promoted from the accel test surface to a
  production path in `accel`'s internal wgpu layer, preserving the k-serial
  per-output accumulation order that produced maxULP=0.
- The MAC floor is measured, not guessed: the crossover lies between the
  measured per-dispatch loss (~1M MACs, loses up to 2.08x) and the measured
  win (268M MACs, 8.9x). The ticket bisects with the existing harness and
  records the number as a commented, measured constant.
- Correctness on the established terms: the bridge test asserts device output
  EXACTLY EQUAL (==, bit-for-bit) to the CPU result on hardware. A device
  error, absence, or a platform failing the parity assertion falls back to the
  CPU path — a performance event, never a correctness one — and the fallback
  is observable following accel's existing Accelerated/FallbackReason
  convention where the bridge surface allows. Strict GPU mode fails instead of
  falling back, per the standing architecture default.
- End-to-end proof: with the bridge blank-imported and a device present, the
  measured encoder layer's large matmuls run on the device and the whole-layer
  wall time is recorded against the M19 CPU number; all dl parity suites pass
  unchanged with the bridge active.

## Non-Goals

- No batched device kernel (refused by measurement; future change).
- No Conv device path (unmeasured; the same gate applies before proposing it).
- No change to dl's public API beyond the hook registration function.
- No multi-device execution.

## Impact

- Affected specs: `dl-inference`
- Affected code: `dl` (hook + dispatch sites), `accel` internal wgpu
  (production matmul), new `accel/dlbridge`, docs, changelogs, skills.
- New package ⇒ the AGENTS.md new-package contract applies: Docs page, both
  README package tables, docs index, allpkgs registration, changelog both
  languages, skills — all in this change.
