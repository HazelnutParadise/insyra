# Change: Use every core on the host side of exact nearest

## Why
The host half of `ExecuteNearestExact` runs on one core, and it is the half that decides whether the operation is worth anything.

Two consequences, both measured on an Apple M3 with 8 cores.

**Machines with no GPU are paying five times over.** The no-device path is `exactNearestAll`, a single-threaded loop over rows. Splitting it across cores makes it 5.2x faster on 200,000 rows by 16 dimensions with 1024 query points: 1.575 s becomes 301.6 ms. Nothing about that work resists parallelism — the rows are independent. Every other hot loop in this repository is already parallel; `stats/internal/clustering/cluster.go:88` splits its distance matrix across cores above a work threshold.

**The measured speedups were against the wrong baseline.** Comparing a device against one core, when the alternative is eight, overstates the device. Redone honestly, on 24 shapes across three dimension counts:

| rows | dims | queries | 1 core | 8 cores | device | vs 1 core | vs 8 cores |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 200,000 | 16 | 32 | 66.7 ms | 14.3 ms | 39.8 ms | 1.68x | **0.36x** |
| 200,000 | 16 | 1024 | 1.575 s | 301.6 ms | 171.6 ms | 9.18x | 1.76x |
| 200,000 | 64 | 1024 | 9.330 s | 1.870 s | 614.6 ms | 15.18x | **3.04x** |
| 10,000 | 4 | 32 | 2.03 ms | 0.43 ms | 2.76 ms | 0.74x | **0.15x** |

The device beats eight cores in 13 of the 24 shapes and never by more than 3.04x. The floor that decides when to use it was calibrated against one core, so it is wrong in both directions — and it reads only the query count, when the measurement says the dimension count matters just as much: four dimensions need 256 query points before the device is worth it, sixty-four dimensions need eight.

The host side also holds the device path back. At 200,000 rows the host verification costs about 25 ms whatever the query count, which is 79.6% of the total when there are eight query points. Some of that is the shortlist arriving candidate-major, so reading one row's candidates strides across the whole array.

## What Changes
- Split both host loops — the no-device path and the verification of a device shortlist — across cores, above a work threshold, matching how the rest of the repository parallelises
- Read the shortlist row by row rather than striding across a candidate-major array
- Replace the query-count floor with one that reads the work per row, recalibrated against the parallel host
- Re-measure the whole shape map against the parallel host and record it

## Impact
- Affected specs: `accel-gpu-execution`
- Affected code: `accel/exact.go`, `accel/internal/wgpu/wgpu.go`, `accel/backend_wgpu.go`
- No API change; results are unchanged and still asserted equal to the reference
