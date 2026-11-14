# 0031 — The GT 1030 still loses the resident recurrent step

**Status:** accepted
**Date:** 2026-11-14

## Context
Record 0030 made the device `tanh` settle in 64-bit arithmetic first and re-measured the resident recurrent step on the M3, where the device now wins at about ten million edges with a batch of 16. The GT 1030 had not been re-measured.

## Decision
No default changes: the device path stays opt-in on every card. `TestRecurrentStepTiming` on `dev` at 39131a6a, on an idle Windows host (i7-13700, 24 threads; GeForce GT 1030, Vulkan; CPU load 4.4%), best of three, rounded host against device, with the 2026-11-07 device time before the faster `tanh` in parentheses:

| nodes × in-degree × batch | rounded host | device | device before |
| --- | --- | --- | --- |
| 10,000 × 10 × 1 | 5.0 ms | 19.2 ms | 31.6 ms |
| 10,000 × 10 × 16 | 23.7 ms | 120 ms | 272 ms |
| 100,000 × 10 × 1 | 40.1 ms | 203 ms | 252 ms |
| 100,000 × 10 × 16 | 238 ms | 1.83 s | 3.33 s |
| 100,000 × 100 × 16 | 1.93 s | 17.3 s | 18.1 s |
| 1,000,000 × 10 × 16 | 5.20 s | does not fit 2 GB | does not fit 2 GB |

The faster `tanh` halves the device time where `tanh` dominated, but the device is still 4 to 9 times slower than the rounded host at every size. At in-degree 100 the step is bound by the exact edge sums, which this card computes far slower than the M3 does.

## Options rejected
- Choosing the device by size. The M3 wins where the GT 1030 loses by nine times, so a size threshold would have to be measured per card.

## Consequences
- The M3 numbers of record 0030 and these are the two cards measured; a resident step pays only on a GPU whose exact integer arithmetic outruns all the host's cores, which the GT 1030's does not.
