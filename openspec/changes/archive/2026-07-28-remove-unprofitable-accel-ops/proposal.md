# Change: Remove the accelerated operations measurement says lose

## Why
Three of the four device operations are known not to pay, and each one is a standing claim that acceleration helps where it does not.

| operation | measured | |
| --- | --- | --- |
| `OpSum` | 0.7x | Memory-bound. `delivery-plan.md` lists column sums on the permanent CPU side, and the plan has said since it landed that this kernel was scaffolding to replace once a profitable one existed. One now exists. |
| `OpSquaredDistance` | superseded | Returns the whole rows-by-queries matrix, so readback grows with the answer. That is what `OpNearestQuery` was built to fix, and it has no caller. |
| `OpNearestQuery` | unusable | Answers in `f32`. Every caller in `stats` computes in `float64`, which is precisely why `ExecuteNearestExact` was written. It is superseded by its own successor. |

Keeping them is not free. Each is a kernel to keep compiling against a moving WebGPU backend, a public method to document, and an entry in the changelog claiming a speedup — and several of those claims were measured against a single CPU core, which the project has since established overstates them by up to the core count.

What is kept is the part that works: the shortlist kernel and `ExecuteNearestExact`, which returns exactly the `float64` answer while a device narrows the field, and the runtime around it — discovery, the resident cache, the scheduler, and the observable fallback that makes a missing device a performance event rather than a correctness one. That runtime is the seam a future kernel lands in, and rebuilding it would cost far more than keeping it.

None of this has shipped. `ExecuteDistances` and `ExecuteNearestQuery` do not exist in `v0.3.0`, so no released API changes.

## What Changes
- Remove `OpSum`, `OpSquaredDistance` and `OpNearestQuery`, their WGSL kernels, their pipeline caches and their backend dispatch
- Remove `ExecuteDistances`, `ExecuteNearestQuery`, `SquaredDistancesCPU`, `NearestQueryCPU` and their references
- Remove the device path from `ExecuteDataList`, `ExecuteDataTable` and `ExecuteProjectedDataset`, and the methods themselves — with no profitable reduction there is nothing for them to do
- Remove the `accel run <var>` CLI command, which existed to invoke them
- Keep `OpNearestShortlist`, `ExecuteNearestExact`, `NearestExactCPU`, the whole runtime, and the measurement harness
- Correct the speedup figures still published in `Docs/accel.md` and both changelogs, which were measured against one core

## Impact
- Affected specs: `accel-gpu-execution`
- Affected code: `accel/types.go`, `accel/executor.go`, `accel/distance.go`, `accel/reference.go`, `accel/exact.go`, `accel/backend_wgpu.go`, `accel/internal/wgpu/wgpu.go`, `cli/commands/accel.go`
- No released API changes; the removed surface has never appeared in a release
