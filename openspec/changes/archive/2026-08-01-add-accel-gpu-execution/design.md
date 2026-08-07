## Context
The accel package is 5,300 lines that can name every GPU on the host and put nothing on one. `ExecuteProjectedDataset` walks the plan, calls `BackendAllocator.Materialize`, and records metrics; `Buffer.Values` is read at exactly two sites, both of which only take `len()` or format the values into a fingerprint hash. The numbers are never touched.

The blocker has been the toolchain, not the design. The accel package is deliberately cgo-free, which ruled out every established Go GPU binding. `github.com/gogpu/wgpu` closes that gap: a WGSL compute shader was measured running on an Apple M3 through the Metal backend with `CGO_ENABLED=0`, returning a result identical to the CPU reference.

Measurements from that run set the terms of this design. Over 16M `float32` (64 MiB) on an M3:

| Scenario | GPU | CPU | Ratio |
| --- | --- | --- | --- |
| One column sum, including upload and readback | 17.5 ms | 13.8 ms | 0.79x — GPU loses |
| Column resident on device, 20 repeated passes | 2.0 ms/pass | 4.7 ms/pass | 2.4x |
| Compute-bound kernel, 64 transcendental terms per element | 69.7 ms | 8.10 s | 116x |

The compute-bound row comes from a standalone measurement taken while evaluating the backend, not from a benchmark in this repository — this change ships one operation, and a compute-bound kernel would need a second shader.

Upload cost 10 ms and readback 7 ms against a 0.3 ms dispatch. Transfer is roughly thirty times the compute for a plain reduction, which is why `shouldFallbackForProfitability` is inverted today: it judges a 16M-row sum profitable, and the measurement says it is not.

## Goals / Non-Goals
- Goals:
  - run one operation end to end on a real device and return a number that matches the CPU path
  - give the runtime an execution seam that can carry an operation and a result
  - keep the core `insyra` module free of any GPU dependency
  - replace fabricated transfer telemetry with measured timings
- Non-Goals:
  - a kernel family; this change ships one reduction and no more
  - multi-device execution, sharding, or merge policies on real hardware
  - `float64` support on the GPU, which no WebGPU or Metal path can provide
  - performance parity or a rewritten profitability rule; this change produces the measurements the rewrite will need

## Decisions

- Decision: adopt `github.com/gogpu/wgpu` as the first real backend.
  - Rationale: one WGSL source reaches Metal on macOS, Vulkan on Windows and Linux, and DX12 on Windows, under `CGO_ENABLED=0` with no shipped native binary. It was verified working on this project's own hardware rather than accepted on documentation. The alternatives were each disqualified: `vulkan-go/vulkan`, `gorgonia/cu` and every Go OpenCL binding need cgo, are unmaintained, or both; writing the Metal Objective-C FFI plus a WGSL compiler in-house is a six-figure line count.
  - Escape hatch: the same public API switches to Rust `wgpu-native` drivers with `-tags rust`, still without cgo, at the cost of shipping a per-OS binary. Backend risk is therefore a build tag, not a rewrite.

- Decision: put the backend in its own Go module at `accel/backend/wgpu`.
  - Rationale: a subpackage of the main module would put six GPU modules into every `go get github.com/HazelnutParadise/insyra`, which breaks the standing decision to keep pure-Go default ergonomics. A build tag does not help, because a same-module import lands in `go.mod` whether or not it compiles. A separate module keeps the dependency invisible until a user asks for it, and the existing `RegisterSDKProbe` / backend registry design already expects blank-import registration.
  - Cost, accepted: a second module in CI, and a version alignment step at release.

- Decision: replace `BackendAllocator` rather than add a second seam beside it.
  - Rationale: `Materialize(dataset, plan) AllocationRecord` has no operation parameter, no context, no error return, and a return type that cannot hold a value. Keeping it alongside a real executor would leave two execution paths disagreeing, and the CLI would keep printing invented numbers from the older one. The three builtin allocators and their `bufferBytes/8` and `/2` constants are removed with it.

- Decision: reduced precision is opt-in per operation; without the opt-in a `float64` column is not accelerated, and it is never narrowed silently.
  - Rationale: WGSL has no `f64`, and Apple GPUs have no double-precision hardware at all — a `f64` shader fails at MSL compile with `'double' is not supported in Metal`. Silently narrowing would change the numbers a data-analysis library returns.
  - Revised during implementation: an earlier draft of this design refused `float64` outright. That does not survive contact with the code. `projectValues` infers `DataTypeFloat64` for any column that is not entirely integral, and Insyra's `DataList` holds `any`, so nearly every real numeric column arrives as `float64`. WGSL also has no `i64`, so `DataTypeInt64` is not eligible either. A blanket refusal would have shipped a GPU backend that never runs on real data.
  - Shape: `WorkloadEstimate.Precision` defaults to `PrecisionExact` and refuses; `PrecisionFloat32` narrows for device execution and marks the result. The safety property is unchanged — the caller, not the runtime, decides to lose precision.
  - Accuracy note: only the within-workgroup tree reduction runs in `f32`; the host folds the per-workgroup partials in `float64`. Error is therefore bounded by the depth of one workgroup reduction rather than by the whole column length. Compensated summation and double-float emulation stay available as a later change.
  - Note: `gogpu/naga` accepts `f64` and `u64` in its WGSL frontend, contrary to the WGSL spec, and defers the failure to the backend — on Metal it surfaces as an MSL `'double' is not supported` error at pipeline creation. Reported upstream as [gogpu/naga#85](https://github.com/gogpu/naga/issues/85). Eligibility must be enforced on the Insyra side regardless of how that lands, because the compiler cannot be relied on to reject a type WGSL does not have.

- Decision: one operation, a `float32` column sum.
  - Rationale: it is the smallest thing that exercises upload, shader compile, dispatch, readback, and result return at once. The measurements above say it will not be faster than the CPU, and that is acceptable — the verifiable output of this change is a correct number and an honest cost table, not a speedup.

- Decision: refuse `gogpu`'s software adapter as an acceleration device.
  - Rationale: the software backend is registered on every non-Android platform and is a SPIR-V interpreter, documented as "much slower than GPU backends (CPU-bound, interpreter, not JIT)". `core.selectAdapterIDs` prefers non-CPU adapters when one exists, but on a host with no GPU driver it still returns the software adapter and `RequestAdapter` succeeds. Accepting it would let the runtime report a workload as accelerated while running a CPU interpreter — the exact failure this change exists to remove. It is cheap to detect: the adapter reports `DeviceType: DeviceTypeCPU`, `Name: "Software Renderer"`.

- Decision: two test seams.
  - A fake executor registered through the backend registry covers the plumbing, and runs on CI machines with no GPU.
  - A real-hardware test verifies the numbers, gated behind `INSYRA_ACCEL_GPU_TESTS=1`, matching the existing `INSYRA_ACCEL_DISABLE_NATIVE_PROBES` convention.
  - Taking the registry as the high seam means these tests survive changes below it; per-collaborator tests over the buffer, pipeline, and readback helpers would go red on every internal rename.

## Measured after implementation

Apple M3, Metal, `CGO_ENABLED=0`, `float64` column narrowed to `float32`, 20 iterations. Device phases come from the runtime's own measured cost; the CPU column is a plain Go loop over `[]float64`.

Figures below are current, after `fix-accel-fingerprint-cost` and `speed-up-accel-projection`. The original run of this table measured a text-rendering fingerprint and reflection-based projection, both since removed.

| Column | GPU op, end to end | transfer | dispatch | readback | device total | CPU loop |
| --- | --- | --- | --- | --- | --- | --- |
| 64 Ki | 1.2 ms | 0.021 ms | 0.034 ms | 0.38 ms | 0.43 ms | 0.018 ms |
| 1 Mi | 12 ms | 0.47 ms | 0.061 ms | 0.93 ms | 1.46 ms | 0.29 ms |
| 4 Mi | 48 ms | 2.43 ms | 0.289 ms | 2.91 ms | 5.62 ms | 1.17 ms |

Host-side projection still costs more than the device — 43 ms of the 48 ms at 4 Mi — but the ratio is now roughly eight to one rather than the seventy to one this table first recorded. What is left splits between projection, fingerprinting, and allocation with no single dominant term.

Two consequences for the profitability rewrite that follows this change:

- Host-side projection still dominates, so a GPU-versus-CPU comparison drawn from the end-to-end figure would still be measuring the wrong thing. Use the device totals.
- Even with projection free, a plain column sum stays memory-bound and does not beat the CPU on unified memory: 8.3 ms of device work against 1.1 ms of CPU work at 4 Mi. The wins are in residency across repeated operations and in compute-bound kernels, not in single reductions.

## Flow

```text
DataList (float64/float32 column)
  -> eligibility check         float64 -> refuse, reason=dtype-not-eligible
  -> projectValues             typed Buffer, validity bitmap
  -> PlanShardableWorkload     single device only in this change
  -> BackendExecutor.Execute   [accel/backend/wgpu]
        upload    host bytes  -> storage buffer
        compile   WGSL        -> MSL / SPIR-V / DXIL  (cached per process)
        dispatch  workgroups, chunked under the 65535 limit
        readback  staging buffer -> partials
  -> CPU folds partials -> ExecutionResult.Reductions[column]
```

Trust boundary: everything from `BackendExecutor.Execute` inward may fail on hardware the runtime does not control, so every step below it returns an error rather than panicking.

## Error map

| Operation | Failure | Caught by | Observable outcome | Tested |
| --- | --- | --- | --- | --- |
| adapter request | no GPU, no driver, no loader | wgpu probe | device absent from discovery; CPU fallback as today | yes |
| shader compile | naga or backend rejects the WGSL | executor | `shader-compile-failed`; strict mode returns the error | yes |
| buffer creation | exceeds `maxStorageBufferBindingSize` | executor | chunked, or `buffer-too-large` when a single chunk still will not fit | yes |
| dispatch | workgroup count over the device limit (65535, hit in practice) | executor | chunked internally, never surfaced | yes |
| readback | map deadline exceeded | executor | `readback-timeout` | yes |
| eligibility | `float64`, or a mixed-type column | projection | `dtype-not-eligible` | yes |

`shader-compile-failed`, `buffer-too-large`, `readback-timeout`, and `dtype-not-eligible` are new fallback reason codes on the existing `FallbackReason` surface.

## Shadow paths

- Zero-row column: return 0 without dispatching. A zero-workgroup dispatch is a validation error on some backends.
- All-null column: validity bitmap is entirely clear, the sum is 0, and the result is marked as covering no values.
- Column shorter than one workgroup: threads past the end contribute the additive identity, guarded by the `idx < count` bound in the shader rather than by dispatch arithmetic.
- Discovery returned no device: unchanged from today — plan is not accelerated, existing fallback reason applies.

## Risks / Trade-offs
- Risk: `gogpu` is eight months old, one author wrote roughly 390 of its ~420 commits, and it has shipped 173 releases while staying on 0.x, so there is no API stability promise.
  - Mitigation: the separate module contains the blast radius, the `-tags rust` backend is a drop-in alternative on the same API, and nothing in core `insyra` depends on it. It is not unused, either — pkg.go.dev lists 19 importing packages, including three that use the compute path the same way this change will: `born-ml/born/internal/backend/webgpu`, `georgebuilds/anneal/backend/webgpu`, and `KEINOS/go-wgpu-mat/mat`.
- Risk: the zero-cgo FFI underneath (`go-webgpu/goffi`) reaches `runtime.cgocall` through `//go:linkname` plus assembly stubs to call `dlopen`/`dlsym`. That is the same family of technique as `purego`, but an independent reimplementation, and it depends on unexported Go runtime internals. A Go release can break it.
  - Mitigation: pin the backend module's `gogpu` version, and treat a Go upgrade as a trigger to re-run the real-hardware numeric test before release.
- Risk: `gogpu`'s public `wgpu.Instance` exposes only `RequestAdapter`; adapter enumeration lives on `core.Instance.EnumerateAdapters()`. Reaching every GPU on a multi-GPU host means depending on the `core` package rather than the stable public surface.
  - Consequence: the standing decision to support heterogeneous multi-GPU for shardable operations in v1 has no clean public API behind it on this backend. This change ships single-device execution only, and the multi-GPU decision should be revisited once there is a measurement to justify the cost.
- Risk: Metal and GLES return `ErrTimestampsNotSupported` from `CreateQuerySet`; only Vulkan and DX12 implement GPU timestamp queries.
  - Consequence: on macOS the measured cost table is host wall-clock around submit and map, not GPU-side timestamps. That is enough to compare against the CPU path, and the spec requirement is written as measured duration rather than GPU timestamp for this reason.
- Risk: `adapter.Info()` reports an Apple M3 as `DiscreteGPU`, which would mislead the shared-memory versus discrete-memory classification the runtime already relies on.
  - Mitigation: classify memory from the vendor and backend rather than trusting the reported device type, and cover it with a test.
- Risk: verified on macOS and Metal only; Windows, Linux, NVIDIA, and AMD are untested.
  - Mitigation: the change is not complete until the same numeric test passes on at least one non-Apple host; until then the backend is documented as macOS-verified.
- Trade-off: this change makes acceleration measurably slower than the CPU for the one operation it ships.
  - Accepted: the alternative is continuing to report invented numbers. A correct slow path can be made fast; a fabricated fast path cannot be made correct.
