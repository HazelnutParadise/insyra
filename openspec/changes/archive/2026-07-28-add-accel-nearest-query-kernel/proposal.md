# Change: Collapse the distance matrix on the device

## Why
The squared-distance kernel returns one value per (row, query) pair, and measurement showed that is the wrong shape: at 100,000 rows and 64 query points the device spends 16 of its 18 ms moving the answer back. Compute is not the constraint.

Every consumer of that matrix immediately reduces it. `stats.KMeans` wants the nearest centroid per row; `stats.KNNClassify` and `KNNRegress` want the smallest few. Doing the reduction on the device turns the output from rows-times-queries into rows, which is where the remaining win is.

## What Changes
- Add `OpNearestQuery`, returning the index of the closest query point and its squared distance, one pair per row
- Add `Session.ExecuteNearestQuery` and a CPU reference, held to the same bit-parity gate
- Break ties by lowest query index on both sides, so the answer is deterministic

## Impact
- Affected specs: `accel-gpu-execution`
- Affected code: `accel/types.go`, `accel/executor.go`, `accel/distance.go`, `accel/reference.go`, `accel/backend_wgpu.go`, `accel/internal/wgpu/wgpu.go`
