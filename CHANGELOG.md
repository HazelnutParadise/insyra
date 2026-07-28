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
- `accel` is now part of `allpkgs`, so the standard `go get .../allpkgs` install registers the GPU backend automatically. Registration is lazy — no device is probed until an accel session opens.
- GPU execution is builtin. The backend is a pure-Go WebGPU implementation ([gogpu/wgpu](https://github.com/gogpu/wgpu)) that registers itself when `accel` is initialised, so there is nothing extra to install and nothing to import for a side effect. It builds with `CGO_ENABLED=0` and reaches Metal on macOS, Vulkan on Linux and Windows, and DirectX 12 on Windows. Programs that never import `accel` compile none of it, though the gogpu modules do appear in `go list -m all`. Set `INSYRA_ACCEL_DISABLE_WGPU=1` to turn it off without changing code.
- **Breaking:** `BackendAllocator`, `RegisterBackendAllocator`, `AllocationRecord`, and `AllocatorKind` are replaced by `BackendExecutor`, `RegisterBackendExecutor`, `ExecuteRequest`/`ExecuteResponse`, and `ExecutorKind`. The old seam could not carry an operation or return a value, so no real backend could implement it.
- **Breaking:** `ExecutionResult.Allocator` and `ExecutionResult.AllocatorKind` are now `Executor` and `ExecutorKind`; `ExecutionResult.BytesMoved` is gone. It was derived from fixed per-backend constants rather than measured, so it is replaced by `BytesUploaded` and the three measured durations.
- GPU execution requires an explicit precision opt-in. WGSL has no `f64` and Apple GPUs have no double-precision hardware, so a `float64` column is not narrowed unless `WorkloadEstimate.Precision` is set to `accel.PrecisionFloat32`. Without it the workload falls back to the CPU with reason `precision-not-accepted`.
- Added fallback reasons `no-backend-executor`, `precision-not-accepted`, `dtype-not-eligible`, `shader-compile-failed`, `buffer-too-large`, `readback-timeout`, and `execution-failed`.
- A backend-reported CPU or software adapter is never treated as an acceleration device, so a host with no GPU driver falls back to the CPU instead of running a software interpreter and calling it acceleration.
- Projecting a column is much cheaper. Dataset fingerprinting no longer renders every value to text before hashing it, and `projectValues` no longer dispatches through `reflect` per element — the old path heap-allocated once per value. On a 4 Mi `float64` column, projection went from 4,194,308 allocations to 4, `ProjectDataList` from 357 ms to 43 ms, and an end-to-end GPU column sum from 354 ms to 48 ms. Fingerprints are session-local, so the different hash values are not observable across runs.

### CLI

- `load <file.csv>` accepts `infer true|false`, defaulting to `true`. Passing `infer false` loads every cell as a raw string. The option is rejected for JSON and Excel files.
