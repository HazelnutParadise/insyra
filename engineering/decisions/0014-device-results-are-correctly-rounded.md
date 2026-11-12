# 0014 — A float32 result that may come from a device is the correctly rounded exact value

**Status:** accepted
**Date:** 2026-09-25

## Context
WGSL guarantees correct rounding only for `+ - *`, and still lets an implementation flush subnormals, ignore the sign of zero, reassociate and contract. `/`, `exp`, `sqrt` and `tanh` are bounded loosely. The device `EdgeSum` prototype matched the CPU's order on the M3 because an xor kept Metal from fusing, which worked by observation, not by any guarantee. Integer operations are exact under the specification. The owner ruled on 2026-09-25 for production device kernels and device-resident tensors and delegated the technical design.

## Decision
A float32 result that may come from a device is defined as the correctly rounded value of the exact operation, and a device kernel computes it in integer arithmetic. `EdgeSum` and its gradients are the exact sum of their products rounded once. `nn.Tanh` is correctly rounded. A result that is exactly zero is `+0`.

## Options rejected
- Keep `EdgeSum`'s fixed ascending order and reproduce it on the device. A fixed order can only be reproduced by one thread per node, and on a conforming implementation only by luck.
- Accept a tolerance between device and CPU. The device rule requires the device to change no numbers.

## Consequences
- The CPU and the device agree by definition rather than by one imitating the other. This is IEEE 754 §9.2's recommendation for elementary functions, the approach of CORE-MATH and RLIBM for binary32, and for sums the Kulisch long accumulator of ReproBLAS and ExBLAS.
- An exact sum does not depend on evaluation order, so any split of the work across cores or devices returns the same bits.
- Exactness has a CPU price, recorded rather than avoided (M33, 2026-09-25, `BenchmarkEdgeSum` on all 8 cores of the M3, best of 5). The exact 640-bit register alone was 3.1x to 23.6x slower than the fixed-order version. A float64 fast path that accepts `float32(S)` only when Higham's bound keeps `[S−B, S+B]` clear of every float32 midpoint (Ziv's method) brings it to 1.05x to 2.28x: 1.35x forward at 10k nodes, 10 edges, batch 1 (0.09 to 0.12 ms); 1.10x at 100k nodes, 100 edges, batch 16 (488 to 538 ms); 1.79x reverse at 1M nodes, 10 edges, batch 16 (1,143 to 2,041 ms). On random data with a quarter of the sums built to cancel, the fast path settled 75% of sums.
- Device MatMul predates this rule (record 0009); its move to the exact rule waits for the owner, as an `AGENTS.md` follow-up.
