# Change: Return an answer when no device is available

## Why
Fallback is announced but not performed. On a machine with no GPU, `ExecuteDistances` and `ExecuteNearestQuery` return no error, `Accelerated: false`, the reason `no-accelerator` — and an empty result. Measured on this host with the device disabled:

```
err=<nil> accelerated=false reason="no-accelerator" Index=[] Distance=[]
```

The CPU reference that would have produced the right answer already exists and is exported, so every caller has to notice `Accelerated` is false and call it themselves. Nothing stops a caller from reading the empty slice as a result, and the roadmap's goal is acceleration that needs no code change from the caller — which cannot hold while the caller is responsible for the case where acceleration did not happen.

The distinction that matters is *why* the device did not run. If the device is missing, refused the work, or failed, the CPU can still answer and should. If the caller's own terms are what excluded the device — asking for exact precision when the only kernel is `f32`, or handing over columns no kernel accepts — then computing on the CPU would deliver exactly what the caller refused, and returning nothing is correct.

## What Changes
- Compute the result on the CPU and return it whenever the device is unavailable or fails: no accelerator, no registered executor, CPU-only mode, unprofitable workload, discovery error, shader compile failure, buffer too large, readback timeout, execution failure
- Keep returning no result when the request itself is ineligible: `precision-not-accepted`, `dtype-not-eligible`, `workload-unsupported`
- Leave strict GPU mode alone — it exists to fail rather than fall back
- Report the reason and `Accelerated: false` exactly as now, so where the work ran stays observable

## Impact
- Affected specs: `accel-gpu-execution`
- Affected code: `accel/distance.go`
- No behavioural change for callers that already branch on `Accelerated`; the runtime has no in-repo production callers yet, only tests
