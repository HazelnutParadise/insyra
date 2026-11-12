# 0009 — `nn`'s large MatMul runs on the device by default

**Status:** accepted
**Date:** 2026-08-03

## Context
Measured on the M3 against the all-core CPU (record 0002), best of 5, with upload, dispatch and readback included (2026-08-02): `[4096,256]×[256,256]` 9.3 ms against 82.4 ms (8.9x), FFN up 58.5 ms against 683 ms (11.7x), FFN down 55.4 ms against 730 ms (13.2x), `[4096,4096]²` 1.87 s against 97.2 s (52x). Every shape returned maxULP 0, because the kernel's per-output thread accumulates along k in the CPU's serial order. Attention-shaped batched products driven as 128 separate dispatches lost by 1.08x to 2.08x.

## Decision
Large 2-D MatMul in `nn` runs on the device by default, from 16Mi multiply-accumulates up, with CPU fallback. `nn` imports `accel` directly; the opt-in bridge package was removed. `INSYRA_ACCEL_DISABLE_WGPU=1` disables the backend and `nn.RegisterDeviceMatMul(nil)` clears the hook. Batched products stay on the CPU.

## Options rejected
- Keep the opt-in bridge. The dependency-cycle argument held only for the bridge: `nn → accel → insyra` is acyclic, and the compile cost is about 1.9 s of cold build and 200 KB.
- Place the floor at the crossover near 4M MACs. 4M to 8M sit in the noise band (device/CPU 0.90 and 0.96, flipping run to run); 16Mi is the first rung that wins dependably (0.74, improving to 0.135 at 268M).

## Consequences
Bit parity here is an observation about Apple and Metal, not a property of the kernel: it holds because Metal and Go on arm64 both fuse multiply-add, which WGSL does not promise. MatMul predates record 0014, and moving it to the exact rule is an open follow-up in `AGENTS.md`.
