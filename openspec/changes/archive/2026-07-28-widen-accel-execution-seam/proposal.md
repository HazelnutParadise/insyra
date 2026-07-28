# Change: Carry a whole dataset through the execution seam, not one column at a time

## Why
`ExecuteRequest` names a single column and a single `[]float32`, so `ExecuteProjectedDataset` calls the backend once per column. A DataTable with eight columns pays eight full round trips — eight uploads, eight dispatches, eight readback waits — where the device could have taken all eight in one submission. Readback is the dominant device cost at roughly 3 ms per call on an Apple M3, so the waste is proportional to column count.

It also blocks the kernels the roadmap targets. Pairwise correlation, distance matrices, KMeans and PCA all read several columns at once and produce a result that is not per-column. A seam that can only describe one column cannot express them, so the profitable half of the roadmap has nowhere to plug in.

## What Changes
- `ExecuteRequest` carries a slice of named columns instead of one column
- `ExecuteResponse` returns a map of per-column reductions instead of one value
- `ExecuteProjectedDataset` submits one request per dataset rather than one per column
- Cost accounting stays per execution, so a multi-column submission reports one measured transfer, dispatch, and readback

## Impact
- Affected specs: `accel-gpu-execution`
- Affected code: `accel/executor.go`, `accel/backend_wgpu.go`, `accel/internal/wgpu/wgpu.go`
