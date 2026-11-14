# 0029 — A resident recurrent step pays only when its edge sums dominate

**Status:** superseded by 0030 (its measurements; the decision stands)
**Date:** 2026-11-14

## Context
Record 0019 left open whether a resident recurrent step beats the all-core CPU on a GPU stronger than the GT 1030, and no further operation was to move to the device until a measurement answered it. `TestRecurrentStepTiming` times `tanh(EdgeSum(W, s) + input)` unrolled four times, the reverse pass and SGD, on the default tape, the rounded tape on the host and the rounded tape on the device. Every operation of that step already runs on the device.

## Decision
The device path stays opt-in and no further operation moves to the device for speed. The measurements:

- **Apple M3** (8 cores, Metal; load average 4 to 5 before the run, rising to about 14 with the run's own CPU columns; 2026-11-14), best of three, rounded host against device: 10,000 nodes, in-degree 10, batch 1: 4.1 ms against 22.9 ms; 10,000 × 10 × 16: 31.0 ms against 134 ms; 100,000 × 10 × 1: 43.4 ms against 150 ms; 100,000 × 10 × 16: 420 ms against 1.34 s; **100,000 × 100 × 16: 5.91 s against 3.59 s** (the default tape: 4.57 s); 1,000,000 × 10 × 16: 17.6 s against 20.2 s (the default tape: 13.3 s).
- **GeForce GT 1030** (Vulkan, against an i7-13700 with 24 threads; 2026-11-07): the device is 8 to 14 times slower than the rounded host at every size, and 1,000,000 × 10 × 16 does not fit its 2 GB.

`EdgeSum` alone is faster on the M3 GPU from 100,000 nodes on (`TestRoundedEdgeSumTiming`, same session, load average about 10 at the start): 100,000 × 100 × 16 forward 1.10 s on 8 cores against 338 ms; 1,000,000 × 10 × 16 forward 1.82 s against 572 ms, and its weight gradient 3.14 s against 458 ms. What the whole step loses is the device `tanh`: the 128-bit fixed-point kernel takes 2.85 s for 2^24 elements on the M3, about 0.17 µs each, so the 6.4 million forward `tanh` values of a 100,000 × 16 step unrolled four times cost about 1.1 s, nearly all of the 1.34 s the device needs at in-degree 10. Only at in-degree 100 do the edge sums outweigh it.

## Options rejected
- Moving more operations to the device. Every operation of the measured step is already there; what would change the answer is a cheaper exact `tanh`, not another kernel.
- Defaulting a recurrent tape to the device when the in-degree is high. The win is one row on one GPU, and the GT 1030 loses at every size; a default needs a threshold measured on more than one card.

## Consequences
- A caller with a high in-degree (around 100) on a GPU like the M3 can choose the device and gain about 1.6 times over the rounded host and 1.3 times over the default tape; below that the host is faster.
- A faster exact device `tanh` is the lever, recorded as a follow-up in `/AGENTS.md`. The M36 kernel-time ladder in the archived `accel-device-edge-sum` design is superseded by the `EdgeSum` numbers above.
