## Decisions

- Moving a topology is explicit (`To` and `Release`, as for tensors), not a lazy cache freed by the garbage collector: a cache would hold its memory budget until a collection, upload inside an operation, and re-upload silently after a lost device. Reviewed with Fable before implementing.
- The functions are `Rounded*`, like every resident operation; `Exact` already names `PrecisionExact` and the propose-then-verify path in accel.
- A resident EdgeSum needs `PrecisionFloat32Rounded`, as every resident operand does, although the sum is exact in every precision: a `PrecisionExact` tape that ran EdgeSum and refused the next MatMul would be the more surprising rule.
- One invocation per output: per (batch, target) for the sum, per (batch, source) for the value gradient, per edge for the weight gradient. A node with a very high degree is summed by one invocation.

## Measurement (M3, 8 cores, best of 5, inputs already on the device)

| nodes | degree | batch | CPU forward | GPU forward | CPU backward | GPU dW | GPU dV |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 10,000 | 10 | 1 | 0.54 ms | 1.54 ms | 0.93 ms | 12.7 ms | 0.85 ms |
| 10,000 | 10 | 16 | 6.6 ms | 3.9 ms | 46.9 ms | 2.8 ms | 4.4 ms |
| 10,000 | 100 | 1 | 7.5 ms | 2.9 ms | 12.0 ms | 17.2 ms | 1.6 ms |
| 10,000 | 100 | 16 | 125 ms | 24 ms | 262 ms | 14 ms | 22 ms |
| 10,000 | 1000 | 1 | 162 ms | 17 ms | 203 ms | 147 ms | 24 ms |
| 10,000 | 1000 | 16 | 3.14 s | 0.32 s | 1.89 s | 0.25 s | 0.31 s |
| 100,000 | 10 | 1 | 8.5 ms | 3.8 ms | 32.9 ms | 13.8 ms | 3.2 ms |
| 100,000 | 10 | 16 | 140 ms | 35 ms | 184 ms | 26 ms | 33 ms |
| 100,000 | 100 | 1 | 121 ms | 22 ms | 149 ms | 142 ms | 33 ms |
| 100,000 | 100 | 16 | 2.21 s | 0.37 s | 6.51 s | 0.33 s | 0.60 s |
| 1,000,000 | 10 | 1 | 240 ms | 32 ms | 526 ms | 122 ms | 51 ms |
| 1,000,000 | 10 | 16 | 4.59 s | 0.64 s | 9.11 s | 0.60 s | 0.96 s |

The weight gradient at batch 1 rounds one product per edge, and setting up and rounding a 27-digit register per edge costs more than the CPU's float64 fast path; at batch 16 the device wins every row. Recorded by `TestRoundedEdgeSumTiming` (`INSYRA_ACCEL_GPU_TESTS=1 INSYRA_ACCEL_TIMING=1`).
