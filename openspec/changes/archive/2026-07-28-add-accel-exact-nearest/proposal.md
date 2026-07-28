# Change: Exact nearest neighbours through a device shortlist

## Why
`ExecuteNearestQuery` is fast and unusable by the callers it was built for. It answers in `f32`, and every caller in `stats` computes in `float64` — `stats/internal/clustering` alone mentions `float64` 88 times and `float32` never. Handing those callers an `f32` answer would change their results, which the acceleration roadmap forbids.

Measurement says the gap closes. Measured on an Apple M3 over 200,000 rows by 16 dimensions, asking for the two nearest:

| query points | `float64` throughout | device shortlist plus `float64` decision | |
| --- | --- | --- | --- |
| 16 | 40.7 ms | 40.4 ms | not worth it |
| 64 | 125.4 ms | 39.7 ms | 3.2x |
| 256 | 402.3 ms | 56.3 ms | 7.1x |

Rows needing a full recompute: 0 of 200,000. Assignments differing from the `float64` reference: 0.

The device leg barely moves across the sweep while the host grows with the query count, because the device is bound by moving the columns across and the shortlist back rather than by the arithmetic. That is also why there is a floor: below roughly twenty query points the round trip costs more than the work it removes, so the runtime declines and says so.

The device does not have to be right, only close. It ranks in `f32` and keeps a handful of candidates; the CPU recomputes those in `float64` and decides. The final arithmetic never leaves `float64`, so the numbers cannot move, and the device still removes most of the work.

This is the first operation to deliver the arrangement the contract in `AGENTS.md` describes, and the one `stats.KMeans` needs.

## What Changes
- Add `OpNearestShortlist`, a kernel that keeps the *k* smallest `f32` distances per row in registers and returns them plus the distance of the best rejected candidate
- Add `Session.ExecuteNearestExact`, which asks the device for a shortlist, recomputes it in `float64`, and returns the *m* nearest query points per row with `float64` distances
- Recompute a row against every query point when the shortlist boundary falls inside the `f32` error bound, so a shortlist that dropped the true winner cannot produce a wrong answer
- Derive that bound from the dimension count rather than fixing it, and take the conservative unfused form so more rows are rechecked rather than fewer
- Report how many rows were rechecked, so the shortlist width can be judged on real data instead of assumed
- Decline the device below the measured crossover, and report `workload-not-profitable` rather than running it anyway
- Ship `NearestExactCPU` as the reference, and assert equality with it rather than closeness

## Impact
- Affected specs: `accel-gpu-execution`
- Affected code: `accel/types.go`, `accel/executor.go`, `accel/distance.go`, `accel/reference.go`, `accel/backend_wgpu.go`, `accel/internal/wgpu/wgpu.go`
- No existing signature changes; `ExecuteNearestQuery` stays as the `f32` operation for callers that want it
