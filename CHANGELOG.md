# Changelog

Changes that affect people using Insyra, grouped by package the same way release notes are. `## Unreleased` holds what will go into the next release.

v0.3.0 and everything before it is not repeated here — see [GitHub Releases](https://github.com/HazelnutParadise/insyra/releases).

繁體中文：[CHANGELOG_TW.md](CHANGELOG_TW.md)

## Unreleased

### Core

- Added `CSVReadOptions` together with `ReadCSV_FileWithOptions` and `ReadCSV_StringWithOptions`. Setting `RawStrings` keeps every cell as its original string and skips column-level type inference, so values like stock IDs no longer lose their leading zeros and empty cells stay `""` instead of becoming NaN. `ReadCSV_File` and `ReadCSV_String` keep their existing signatures and behavior.

### `isr`

- `CSV_inOpts` gains a `RawStrings` field, which `DT.From` passes through to the reader.

### `accel`

- `accel` now executes on real hardware. `ExecuteDataList`, `ExecuteDataTable`, and `ExecuteProjectedDataset` return the computed value per column in `ExecutionResult.Reductions`, along with measured `Transfer`, `Dispatch`, and `Readback` durations and `BytesUploaded`.
- `accel.Session` is now safe for concurrent use. Every public method is serialized behind a session lock, so several goroutines can share one session; previously concurrent `ExecuteDataList` calls raced on the cache and report state. Device submission is also serialized process-wide, because all sessions share one GPU handle.
- Added `accel.Default()`, a session shared by the process and created on first use. Discovery runs once, the resident cache is shared across operations, and `Close` on it is a no-op because no caller owns it. Importing the package still opens no device.
- Added `accel.OpSquaredDistance` with `Session.ExecuteDistances`, computing the squared Euclidean distance from every row to each query point on a GPU, and `accel.SquaredDistancesCPU` as its reference. Device results are verified bit-identical to the reference on the running platform.
- Added `accel.OpNearestQuery` with `Session.ExecuteNearestQuery` and `accel.NearestQueryCPU`, reporting the closest query point per row and its squared distance. The minimum is taken on the device, so the result grows with rows rather than with rows times queries — 13.6x over the CPU at 64 query points on an Apple M3. Ties go to the lowest query index.
- `accel` is now part of `allpkgs`, so the standard `go get .../allpkgs` install registers the GPU backend automatically. Registration is lazy — no device is probed until an accel session opens.
- GPU execution is builtin. The backend is a pure-Go WebGPU implementation ([gogpu/wgpu](https://github.com/gogpu/wgpu)) that registers itself when `accel` is initialised, so there is nothing extra to install and nothing to import for a side effect. It builds with `CGO_ENABLED=0` and reaches Metal on macOS, Vulkan on Linux and Windows, and DirectX 12 on Windows. Programs that never import `accel` compile none of it, though the gogpu modules do appear in `go list -m all`. Set `INSYRA_ACCEL_DISABLE_WGPU=1` to turn it off without changing code.
- **Breaking:** `BackendAllocator`, `RegisterBackendAllocator`, `AllocationRecord`, and `AllocatorKind` are replaced by `BackendExecutor`, `RegisterBackendExecutor`, `ExecuteRequest`/`ExecuteResponse`, and `ExecutorKind`. The old seam could not carry an operation or return a value, so no real backend could implement it.
- **Breaking:** `ExecutionResult.Allocator` and `ExecutionResult.AllocatorKind` are now `Executor` and `ExecutorKind`; `ExecutionResult.BytesMoved` is gone. It was derived from fixed per-backend constants rather than measured, so it is replaced by `BytesUploaded` and the three measured durations.
- GPU execution requires an explicit precision opt-in. WGSL has no `f64` and Apple GPUs have no double-precision hardware, so a `float64` column is not narrowed unless `WorkloadEstimate.Precision` is set to `accel.PrecisionFloat32`. Without it the workload falls back to the CPU with reason `precision-not-accepted`.
- Added fallback reasons `no-backend-executor`, `precision-not-accepted`, `dtype-not-eligible`, `shader-compile-failed`, `buffer-too-large`, `readback-timeout`, and `execution-failed`.
- A backend-reported CPU or software adapter is never treated as an acceleration device, so a host with no GPU driver falls back to the CPU instead of running a software interpreter and calling it acceleration.
- Projecting a column is much cheaper. Dataset fingerprinting no longer renders every value to text before hashing it, and `projectValues` no longer dispatches through `reflect` per element — the old path heap-allocated once per value. On a 4 Mi `float64` column, projection went from 4,194,308 allocations to 4, `ProjectDataList` from 357 ms to 43 ms, and an end-to-end GPU column sum from 354 ms to 48 ms. Fingerprints are session-local, so the different hash values are not observable across runs.
- `ExecuteDistances` and `ExecuteNearestQuery` now return the result whether or not a device ran. Previously a host with no GPU got `Accelerated: false`, the reason `no-accelerator`, and an empty slice, leaving every caller to notice and call the CPU reference itself. The same applies when a device is present but fails, times out, or exceeds a buffer limit. `Accelerated` and `FallbackReason` still report where the work ran, so nothing becomes less observable. A request refused on its own terms — `precision-not-accepted`, `dtype-not-eligible`, `workload-unsupported` — still returns nothing, because computing it on the CPU would deliver exactly what the caller declined. Strict GPU mode still returns an error rather than a CPU result.
- Added `accel.OpNearestShortlist` and `Session.ExecuteNearestExact`, which returns the M nearest query points per row as exact `float64` values while still using a GPU for most of the work. The device ranks in single precision and returns a shortlist per row plus the distance of the best candidate it discarded; the host recomputes that shortlist in `float64` and decides, falling back to every query point for the rows where single precision could not separate the candidates. The answer equals `accel.NearestExactCPU` exactly, so no precision opt-in is required. Measured on an Apple M3 over 200,000 rows against a host using all eight cores: 2.5x at 16 dimensions and 3.4x at 64, both with 1024 query points, and slower than the host below roughly 2048 distance evaluations per row — where the runtime declines the device and reports `workload-not-profitable`. `ExactNearestResult.Rechecked` reports how many rows took the full path.
- The host side of `ExecuteNearestExact` now uses every core. Both the no-device path and the verification of a device shortlist are split across `GOMAXPROCS` above a work threshold; below it they stay on one core. On 200,000 rows by 16 dimensions with 1024 query points the path taken by a machine with no GPU went from 1.575 s to 306 ms. The device shortlist is also read row-major rather than candidate-major, so verifying one row no longer strides across the whole array.
- `ExecuteNearestExact` no longer panics when asked for nine or more neighbours on a host with a GPU. The shortlist width was clamped to the device's eight slots while the decision still indexed position `m-1`; the device is now skipped for requests it cannot serve, and the host answers instead. It also no longer trusts a shortlist whose single-precision distances overflowed to infinity — the ordering carries no information in that case, and the boundary test passed for the wrong reason. The error bound used to decide whether a shortlist can be trusted was widened to account for the rounding of the difference itself and for squared terms below the smallest normal `float32`.
- **Breaking:** `OpSum`, `OpSquaredDistance` and `OpNearestQuery` are removed, together with `ExecuteDataList`, `ExecuteDataTable`, `ExecuteProjectedDataset`, `ExecuteDistances`, `ExecuteNearestQuery`, `SquaredDistancesCPU`, `NearestQueryCPU`, their WGSL kernels, and the `accel run <var>` CLI command. Each was measured against a host using all its cores and lost: the column sum runs at 0.7x because it moves one value per element and adds it once; the distance matrix reads back a result that grows with rows times query points; and the single-precision nearest query answers in `f32`, which the `float64` callers it was built for cannot use. `ExecuteNearestExact` supersedes the last of these and returns the exact `float64` answer. None of the removed surface appeared in a release. `accel devices`, `accel cache` and `accel plan` are unaffected.

### `stats`

- Added `Predict` to linear, polynomial, exponential, and logarithmic regression results. It follows the GLM prediction signature and returns response-scale point predictions for new data, with predictor-count and row-length validation. R's standard errors and prediction intervals remain outside the current API.
- Added `KMeansResult.Assign` to apply fitted centers to new observations and return the one-based center index with its squared Euclidean distance.
- `PCAResult` now returns the fitted per-column centering and scaling parameters, together with the training scores, so callers can project new observations with the same decomposition.

### CLI

- `load <file.csv>` accepts `infer true|false`, defaulting to `true`. Passing `infer false` loads every cell as a raw string. The option is rejected for JSON and Excel files.
