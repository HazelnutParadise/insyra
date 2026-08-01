# Change: Add a squared-distance kernel with verified bit parity

## Why
`sum` is the only operation the runtime can execute, and measurement puts it on the permanent CPU list — a column sum is memory-bound and the device loses. The roadmap's stage B needs the first kernel where the device actually wins, and the arithmetic-intensity spike named it: pairwise distance measured 62x, because work grows with the number of point pairs while the data grows with the number of points.

Distance is also the shared primitive under the operations users feel as slow. `stats.KMeans`, `stats.KNNClassify`, `stats.KNNRegress` and `stats.Silhouette` all reduce to "how far is every row from each of these query points".

The parity question that gates default-on execution is already settled by measurement, so this kernel can be held to it from the start.

## What Changes
- Add `OpSquaredDistance`, computing the squared Euclidean distance from every row to each of a set of query points
- Extend the execution seam with query operands and a distance result, since the existing request describes one dataset and the existing response holds one number per column
- Ship a CPU reference in the natural fused form, and a test asserting the GPU result is bit-identical to it
- Report the platform's parity result, so an operation that cannot reach parity on a host stays on the CPU there

## Impact
- Affected specs: `accel-gpu-execution`
- Affected code: `accel/types.go`, `accel/executor.go`, `accel/reference.go` (new), `accel/backend_wgpu.go`, `accel/internal/wgpu/wgpu.go`
