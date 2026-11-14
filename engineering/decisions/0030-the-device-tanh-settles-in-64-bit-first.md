# 0030 — The device tanh settles in 64-bit arithmetic first

**Status:** accepted
**Date:** 2026-11-14

## Context
Record 0029 found the 128-bit fixed-point device `tanh` to be nearly all of a resident recurrent step: about 0.17 µs per element on the M3, so the step beat the host only at in-degree 100. The owner chose on 2026-11-14 to look for a cheaper exact scheme rather than stop there.

## Decision
`tanh_rounded` first evaluates a 64-bit fixed-point approximation, as the host's `tanhFloat32` first evaluates float64: `2|x| = n ln2/64 + r`, a 64-entry table for `2^(-(n&63)/64)`, nine series terms for `e^-r`, and two Newton steps on the reciprocal of `1 + e^-2|x|`. Its largest error is 2.9 units of 2^-58 on 3,600 inputs across its range against a 600-bit oracle, from reciprocal starts moved by 2^-18 either way. An input within 64 units of a rounding midpoint takes the 128-bit path, which on the host model of the arithmetic is 243 of the 135,790,592 positive inputs in range. The device equals `nn.Tanh` on all 2^32 inputs on the M3 and on a GeForce GT 1030 over Vulkan.

On the M3, `tanh` over 2^24 elements falls from 2.85 s to 31 ms. `TestRecurrentStepTiming` on the M3 (load average 8 to 9, 2026-11-14), rounded host against device: 10,000 nodes, in-degree 10, batch 1: 4.6 ms against 19.0 ms; 10,000 × 10 × 16: 33.1 ms against 48.9 ms; 100,000 × 10 × 1: 40.5 ms against 98.7 ms; 100,000 × 10 × 16: 320 ms against 493 ms; **100,000 × 100 × 16: 4.29 s against 2.81 s** (the default tape: 4.11 s); **1,000,000 × 10 × 16: 12.1 s against 9.36 s** (the default tape: 11.5 s).

## Options rejected
- f32 arithmetic with error-free transformations. WGSL lets a compiler contract, reassociate and flush subnormals, so an error bound on f32 code holds only for the code a given driver produced. The two f32 estimates the integer path does use (the reduction's `n` and the reciprocal's start) are corrected by exact integer steps after them, so a sloppy estimate costs nothing.
- Dropping the 128-bit path. The few hundred inputs near a midpoint need more than 64 bits to round correctly.

## Consequences
- Record 0029's numbers for the resident step are superseded by these; its decision stands: the device path stays opt-in, and a default needs a threshold measured on more than one card.
- On the M3 the resident recurrent step now beats the host at about ten million edges with a batch of 16 (both rows above); at the smaller sizes measured the host is faster. The GT 1030 timing is not re-measured yet.
