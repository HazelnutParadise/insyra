# 0016 — Device kernels share one exact register and work on resident inputs

**Status:** accepted
**Date:** 2026-09-26

## Context
Record 0014 requires device kernels to compute in integers. M35 built a WGSL library that decodes float32 bit patterns, forms exact products, accumulates them in integer limbs and rounds once. Parity on the M3 covered 3,420 rows and 289,766 products: random data, every float32 bit pattern, cancellation, ties, the subnormal and overflow boundaries, NaN and infinities, 5,000-product rows, and borrows through every digit. Every output equalled the `math/big` oracle and `nn.EdgeSum` bit for bit.

Measured on the M3 over 10M products, best of 5 after a warm-up, all 8 cores for the CPU: the whole device call took 27.6 to 39.1 ms against 3.4 to 6.0 ms for `nn.EdgeSum`. The same call with every row emptied, uploading the same 80 MB, took 25.0 to 33.1 ms. About 85% of the device's time is the transfer; the kernel itself runs at 1.7 to 3.9 G products per second, the CPU's speed or better.

## Decision
Every device kernel does its exact arithmetic with the register in [`accel/internal/wgpu/exact_sum.go`](/accel/internal/wgpu/exact_sum.go): 27 digits of 24 bits in `u32`, counting in units of 2^-298, carries deferred for up to 64 additions, unsigned operations only, and no `select()`. A kernel that needs more exact arithmetic extends this library instead of writing its own. Device `EdgeSum` and the other `nn` kernels take device buffers, `nn` reaches them through device tensors (record 0019), and nothing uploads per call.

## Options rejected
- Upload per call. Even an upload several times faster would at best break even.

## Consequences
- The device `EdgeSum` (M36) is bit-identical to the host on the M3 and on a GeForce GT 1030 over Vulkan, and stays opt-in through `To`. It is not faster: on an idle Windows host (i7-13700, 24 threads; GT 1030, Vulkan 1.4.312; load 2–5%, 2026-11-07) every row of the ladder was slower on the device, forward 4 to 26 times (1M nodes, in-degree 10, batch 16: 566 ms against 2.22 s). The M3 ladder in the archived change was taken under a load average of 20 to 40 and is not quoted. A batch-1 weight gradient rounds one product per edge and is slower on the device regardless of load.
- The earlier f32 prototype (`measure-edge-sum-device`, 2026-09-25) beat the all-core CPU only on the largest graphs, by at most 1.8x (1M nodes, 10 edges, batch 16: 637 ms against 355 ms with the topology resident). On unified memory the operation is bound by memory both devices share.
- Each device call is split into launches of about 2^24 products, because one dispatch past Windows' two-second reset lost the device.
- gogpu/naga bugs found by the parity tests are reported upstream (gogpu/naga#95, fixed in v0.19.1, and #133, fixed in v0.20.4).
