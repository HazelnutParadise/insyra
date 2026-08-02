# Change: Measure whether a device wins dl's MatMul before proposing a kernel

## Why

M19 made the CPU baseline honest: MatMul saturates all eight cores and still
dominates every dl workload (≈90% of an encoder layer at ~0.9 s, reproducible).
The operating contract forbids writing a production device kernel for an
operation until measurement says the device wins — and forbids scoping that
measurement against anything but the all-core baseline. M8's measurements do
not transfer: they covered memory-bound columnar scans, and matmul is
compute-bound, a different class with a different verdict plausible in either
direction once dispatch and readback are priced in.

There is also a precision question measurement must answer before any wiring
ticket can be scoped. dl tensors are natively f32, but a tiled device matmul
reassociates each output's accumulation, so results will generally not be
bit-identical to the CPU's serial-order sums. The result-shape table's
"types the device holds exactly" row assumed bit-identity outright; whether
device matmul can default on, must be opt-in like new-`float64` values, or can
be gated per platform, depends on the parity delta actually observed.

## What Changes

- A prototype tiled f32 matmul kernel and benchmark harness inside `accel`'s
  test/benchmark surface — explicitly NOT wired into `dl`, NOT exported, NOT a
  production path.
- The harness measures device wall time (including transfer, dispatch,
  readback) against the M19 all-core CPU baseline at dl's measured hot shapes:
  the encoder projections/FFN (4096×256·256×256, 4096×256·256×1024,
  4096×1024·1024×256), the batched attention products (128 batches of
  128×64·64×128 and 128×128·128×64), and at least one larger LLM-adjacent
  shape to locate where the win, if any, begins.
- The harness records the maximum absolute and ULP deviation between device
  and CPU results per shape, so the precision decision is made on observed
  numbers rather than assumption.
- The verdict — win/lose per shape, parity delta, and the resulting go/no-go
  with its precision consequence — is recorded in `delivery-status.md` as a
  decision entry, and the milestone row updated. If the device loses, M17
  closes negatively the way M8 did, and that is a complete, successful outcome
  of this change.

## Non-Goals

- No production kernel, no `dl` wiring, no API change, no new dependency.
- No multi-device execution (the standing follow-up's saturation rule applies).
- No f64 emulation and no quantized types.

## Impact

- Affected specs: `dl-inference` (one requirement recording the measurement
  gate for device matmul).
- Affected code: `accel` test/benchmark files only.
