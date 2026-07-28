# Delivery Plan

## Current Phase
Phase 1 - First Real GPU Execution Path (achieved; hardening)

## Stage Objective
Run one operation end to end on a real device and return a number that matches the CPU path. Every accel surface built so far — discovery, projection, cache, scheduler, CLI — has been verified against stubs only, so the first correct GPU number is what turns the existing scaffolding from assumption into fact.

## Active Workstreams
- `M6`: real device execution through an opt-in `gogpu/wgpu` backend module

`M0` through `M5` are closed. They established the convergence surface, the frozen `insyra/accel` runtime API, backend discovery, the columnar layout and cache model, the scheduler and observable fallback, and the CLI/DSL surface. All of it was verified against stubs, which is what `M6` exists to correct.

## Milestones
| id | target | owner | status | verification_signal |
| --- | --- | --- | --- | --- |
| M0 | Control surface established | planning | done | `delivery-plan.md`, `AGENTS.md`, `CLAUDE.md`, and full accel proposal inventory exist |
| M1 | Accel runtime API frozen in spec | planning | done | `accel` package compiles, `go test ./accel` passes, docs surface added |
| M2 | Backend discovery and scoring frozen in spec | planning | done | discoverer registry, builtin CUDA/Metal/WebGPU stubs, native probe seams, Windows/Linux portable probe fallback chains, discovery timeout handling, cross-backend dedupe, shared-memory budget fallback, budget normalization, selection tests, the SDK probe seam, a Windows NVML SDK probe via stdlib `syscall` (no cgo), and driver/compute/PCI metadata on `Device` land; all 10 tasks are closed, and further SDK probes are deprioritized under M6 |
| M3 | Columnar layout and cache model frozen in spec | planning | done | typed projection now emits validity bitmaps, encoded string transport, lineage-aware session-local cache keys, and aggregate budget enforcement; true device residency is still pending allocator-backed work |
| M4 | Scheduler and observable fallback frozen in spec | planning | done | strict-mode fallback reason codes, core accel metrics, weighted shard assignments, merge-policy selection, profitability-aware planning, execution-ledger residency metrics, builtin homogeneous allocator stubs, and deterministic cache eviction land; all 4 tasks are closed, and the remaining execution gap is owned by M6 |
| M5 | CLI/DSL accel surface frozen in spec | planning | done | `accel devices|cache|plan|run <var>`, `config accel.mode`, and `show accel.devices|accel.cache` land; all 8 tasks are closed, and backend-native execution semantics move to M6 |
| M6 | One operation executes on a real device | planning | done | `go test ./accel/... -run GPU` with `INSYRA_ACCEL_GPU_TESTS=1` reduces a `float32` column on a discovered device and the returned value equals the CPU result, with measured transfer/dispatch/readback timings replacing the fabricated per-backend constants |

## Current Blockers
None. `add-accel-gpu-execution` cannot be archived until its numeric test passes on a non-Apple host (task 1.13); that is a coverage gap, not a blocker on further work.

## Next Verifiable Output
`add-accel-gpu-execution` task 1.13: the GPU numeric test passes on a non-Apple host. Everything else in the accel path is now measured and green on macOS, and the host-cost work has reached the point of diminishing returns — no single dominant term is left. Cross-platform verification is the one claim still unbacked.

## Next OpenSpec Change
`add-accel-gpu-execution`, task 1.13 only. It needs a Windows or Linux machine with an NVIDIA or AMD GPU; it cannot be done on this host.

## Decision Log
- decision: Keep acceleration in optional `insyra/accel` packages rather than core `insyra`.
  rationale: Preserve pure-Go default ergonomics and isolate native/runtime dependencies behind explicit opt-in.
  timestamp: 2026-03-27
  impacted_change_ids: `add-accel-runtime-capability`, `add-accel-backend-discovery`, `add-accel-cli-dsl-surface`
- decision: Use `CUDA + Metal + WebGPU native` as the backend strategy and do not use ROCm as the AMD iGPU v1 primary path.
  rationale: This covers NVIDIA, Apple, Intel, and AMD integrated/shared-memory devices with a portable fallback route.
  timestamp: 2026-03-27
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-columnar-layout-cache`, `add-accel-scheduler-multi-gpu`
- decision: Support heterogeneous multi-GPU only for shardable columnar operations in v1.
  rationale: It keeps v1 implementable while preserving a path toward more transparent fusion later.
  timestamp: 2026-03-27
  impacted_change_ids: `add-accel-scheduler-multi-gpu`, `add-accel-observability-fallback`
- decision: Default to observable CPU fallback, with strict GPU-only mode as an explicit opt-in.
  rationale: This balances usability with debuggability and makes backend selection visible to users.
  timestamp: 2026-03-27
  impacted_change_ids: `add-accel-scheduler-multi-gpu`, `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Treat full GPU string kernels as a Phase 2 slice.
  rationale: Phase 1 needs typed columnar transport and encoded-string eligibility, but full string-kernel parity should not block runtime convergence.
  timestamp: 2026-03-27
  impacted_change_ids: `add-accel-columnar-layout-cache`, `add-accel-string-kernels-phase-2`
- decision: Start implementation by freezing the public accel runtime and typed CPU-side projection before backend work.
  rationale: Backend discovery, cache, scheduler, and CLI all depend on one stable runtime contract.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-runtime-capability`, `add-accel-backend-discovery`
- decision: Use a discoverer registry plus `Open()` auto-discovery as the first backend-discovery implementation slice.
  rationale: This keeps real adapters pluggable while making session behavior testable before native bindings exist.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-backend-discovery`
- decision: Ship env-driven builtin CUDA, Metal, and WebGPU adapter stubs before native SDK probing.
  rationale: This creates stable backend boundaries and lets report, scheduler, and CLI work be verified on machines without GPU SDKs.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Land the first accel CLI/DSL surface as probe/report commands before true cache or workload execution exists.
  rationale: This gives users and future agents a stable inspection path without pretending the runtime is already complete.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-cli-dsl-surface`, `add-accel-observability-fallback`
- decision: Return a session alongside strict-gpu and discovery errors so report surfaces remain inspectable on failure.
  rationale: Observable fallback is part of the contract; callers and CLI should still be able to inspect reason codes and metrics when acceleration cannot proceed.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Introduce a shardable multi-device planning surface before true execution scheduling.
  rationale: This establishes the contract for heterogeneous device aggregation and total budget reporting without claiming weighted partitioning or merge execution already exists.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-scheduler-multi-gpu`, `add-accel-cli-dsl-surface`
- decision: Start cache implementation as a session-local resident index fed by typed projection, before adding true device allocators or eviction.
  rationale: This gives the CLI and report surface truthful cache state now, while preserving a clean seam for later VRAM/shared-memory backends.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-columnar-layout-cache`, `add-accel-cli-dsl-surface`
- decision: Enforce cache budgets at the session-local cache layer before introducing backend-native allocators.
  rationale: This makes cache state actionable now and proves eviction/report semantics before native VRAM/shared-memory plumbing is added.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-columnar-layout-cache`, `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Complete the columnar/cache change by adding validity bitmaps and encoded string transport before returning to backend work.
  rationale: Cache/accounting alone was not enough to claim the memory/layout change complete; string/null transport needed to be allocator-ready first.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-columnar-layout-cache`
- decision: Add native probe seams and normalized capability maps before attempting real SDK bindings.
  rationale: Binding to CUDA/Metal/WebGPU without a stable seam would couple probe failures, report semantics, and CLI output too tightly to backend-specific code.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Honor discovery timeout and shared-memory budget fallback inside the runtime before treating backend discovery as converged.
  rationale: A public timeout field and shared-memory budget policy are not credible if they only exist in config shape but not in behavior.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-columnar-layout-cache`, `add-accel-observability-fallback`
- decision: Repair accel CLI/DSL spec text and Cobra flag parsing before further backend work.
  rationale: Broken spec text and a non-functional `--mode` path would make the control surface look complete while failing in actual use.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-cli-dsl-surface`, `add-accel-backend-discovery`
- decision: Move multi-device scheduling from aggregate-only planning to workload-aware weighted partition planning before attempting allocator-backed execution.
  rationale: A multi-GPU surface that only sums budget is not enough to validate heterogenous scheduling semantics or strict/auto profitability behavior.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-scheduler-multi-gpu`, `add-accel-cli-dsl-surface`, `add-accel-observability-fallback`
- decision: Introduce an internal execution ledger allocator before backend-native allocators.
  rationale: Per-device residency and execution metrics need truthful runtime events before CUDA/Metal/WebGPU allocators exist.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-runtime-capability`, `add-accel-columnar-layout-cache`, `add-accel-observability-fallback`
- decision: Split accel CLI into explicit planning and execution surfaces.
  rationale: `accel plan` and `accel run <var>` should not share the same semantics once the runtime has an execution ledger.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-cli-dsl-surface`, `add-accel-runtime-capability`, `add-accel-observability-fallback`
- decision: Add a backend allocator registry before implementing backend-native allocators.
  rationale: The runtime needs a stable seam for CUDA/Metal/WebGPU allocators while preserving a ledger fallback for heterogeneous or unimplemented backends.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-runtime-capability`, `add-accel-observability-fallback`, `add-accel-backend-discovery`
- decision: Seed builtin allocator stubs for CUDA, Metal, and WebGPU before SDK-backed allocators exist.
  rationale: The runtime should exercise backend-specific execution paths for homogeneous plans now, instead of routing every plan through the generic ledger allocator and calling that convergence.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-runtime-capability`, `add-accel-observability-fallback`, `add-accel-backend-discovery`
- decision: Add multi-command fallback chains for portable native probes on Windows and Linux before introducing backend SDK bindings.
  rationale: Auto-detection should survive missing host utilities on real machines; a single optimistic command path would keep env stubs as the practical default.
  timestamp: 2026-03-28
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-observability-fallback`
- decision: Layer SDK probes on top of the native probe seam with a fixed priority of SDK > native command > env stub, instead of replacing the native command probe seam.
  rationale: Native command probes already work on hosts that ship `nvidia-smi`/`system_profiler`/`lspci`; keeping them as a documented fallback means SDK probe outages do not regress discovery, and tests can exercise the priority order without mocking driver libraries.
  timestamp: 2026-05-09
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-observability-fallback`, `add-accel-cli-dsl-surface`
- decision: Ship the first SDK-backed probe (Windows NVML) via stdlib `syscall.LoadLibrary`/`SyscallN` instead of cgo or third-party bindings.
  rationale: The accel package has been deliberately cgo-free; using stdlib lets the SDK probe seam land on Windows immediately without forcing a cgo toolchain on every consumer of `insyra/accel`. Linux/macOS NVML, Metal, and wgpu-native probes can plug into the same seam later.
  timestamp: 2026-05-09
  impacted_change_ids: `add-accel-backend-discovery`

- decision: Adopt `github.com/gogpu/wgpu` as the first real accel execution backend.
  rationale: One WGSL compute source reaches Metal, Vulkan, and DX12 under `CGO_ENABLED=0` with no shipped native binary, which is the only option found that satisfies the standing cgo-free constraint. It was verified on this project's own hardware rather than accepted on documentation: a WGSL reduction ran on an Apple M3 through Metal and returned a value identical to the CPU reference. Every cgo-free alternative surveyed either needs cgo, is unmaintained, or does not exist yet.
  timestamp: 2026-07-28
  impacted_change_ids: `add-accel-gpu-execution`, `add-accel-backend-discovery`, `add-accel-observability-fallback`
- decision: Ship the GPU backend as a separate Go module at `accel/backend/wgpu` rather than a subpackage of the core module.
  rationale: A same-module subpackage would add six GPU modules to every `go get github.com/HazelnutParadise/insyra`, and a build tag does not prevent that because the requirement lands in `go.mod` regardless of whether the file compiles. A separate module keeps the core module's dependency set unchanged and matches the existing blank-import registration design.
  timestamp: 2026-07-28
  impacted_change_ids: `add-accel-gpu-execution`
  superseded_by: the fold-gpu-backend-into-core decision below
- decision: Reverse the above and fold the GPU backend into the core module.
  rationale: Two of the three grounds for separating it did not survive measurement. A `require` on a module with a higher `go` directive has no effect on consumers who never import it — a module requiring a `go 1.29` dependency it does not import builds fine on Go 1.26 — so the minimum-Go argument was wrong. Go compiles only imported packages, so the 442,188 lines of gogpu were never entering anyone's build either; a consumer importing only the root package compiles zero gogpu packages, verified. What remains is that insyra's own build and CI now depend on a 0.x project that shipped 175 releases in seven and a half months, and that six gogpu modules appear in every consumer's `go list -m all`. Both were accepted in exchange for a one-step install with no registration import.
  timestamp: 2026-07-28
  impacted_change_ids: `fold-gpu-backend-into-core`, `add-accel-gpu-execution`
- decision: Restrict GPU execution to `float32` and refuse `float64` columns instead of narrowing them.
  rationale: WGSL has no `f64` and Apple GPUs have no double-precision hardware — a `f64` shader fails at MSL compile with `'double' is not supported in Metal`. Silently narrowing would change the numbers a data-analysis library returns. Refusal is observable through the existing fallback-reason surface and leaves compensated summation or double-float emulation available as a later change. `gogpu/naga` accepts `f64` in its WGSL frontend contrary to the spec, so eligibility has to be enforced on the Insyra side.
  timestamp: 2026-07-28
  impacted_change_ids: `add-accel-gpu-execution`, `add-accel-columnar-layout-cache`
- decision: Replace the `BackendAllocator` seam with an execution seam, and delete the fabricated per-backend transfer constants along with it.
  rationale: `Materialize(dataset, plan) AllocationRecord` has no operation parameter, no context, no error return, and a return type that cannot hold a value, so no real backend can plug into it. The three builtin allocators return `bufferBytes/8` and `bufferBytes/2` as transfer cost and the CLI prints those invented numbers to users as `execution.bytes_moved`; keeping them beside a real executor would leave two execution paths disagreeing.
  timestamp: 2026-07-28
  impacted_change_ids: `add-accel-gpu-execution`, `add-accel-runtime-capability`, `add-accel-cli-dsl-surface`
- decision: Deprioritize further SDK probes (Linux/macOS NVML, Metal, WebGPU) and hold the backend strategy at WebGPU alone until real execution exists.
  rationale: More discovery does not move the runtime closer to computing anything, and `gogpu`'s adapter enumeration already covers the same hardware for the backends it supports. Opening three backend tracks before one of them can run a kernel is cost without evidence. CUDA is reconsidered only when a measurement shows the WebGPU path is the limit.
  timestamp: 2026-07-28
  impacted_change_ids: `add-accel-backend-discovery`, `add-accel-gpu-execution`

## Source Links
- `delivery-plan.md`
- `AGENTS.md`
- `CLAUDE.md`
- `openspec/changes/add-accel-convergence-surface/`
- `openspec/changes/add-accel-runtime-capability/`
- `openspec/changes/add-accel-backend-discovery/`
- `openspec/changes/add-accel-columnar-layout-cache/`
- `openspec/changes/add-accel-scheduler-multi-gpu/`
- `openspec/changes/add-accel-cli-dsl-surface/`
- `openspec/changes/add-accel-observability-fallback/`
- `openspec/changes/add-accel-string-kernels-phase-2/`
- `openspec/changes/add-accel-gpu-execution/`
- `Docs/accel.md`
- `README.md`
- `Docs/README.md`
- `go.mod`
- `datalist.go`
- `datatable.go`
- `interfaces.go`
- `Docs/CCL.md`
- `openspec/specs/cli-entry/spec.md`
- `openspec/specs/command-registry/spec.md`
- `openspec/specs/dsl-commands/spec.md`

## Handoff Notes
- The convergence surface and runtime capability are both in place. `accel` now exists as a compilable opt-in package with `Open/NewSession`, typed projection helpers, and report/device/dataset/buffer surface.
- Use a fresh `GOCACHE` when running Go validation in this environment. The default cache path hit a local toolchain/cache issue after `go clean -cache`, but tests pass with a clean alternate cache directory.
- `add-accel-backend-discovery` is now materially deeper in code. Builtin stubs, native probe seams, normalized capability flags, budget normalization, probe-source reporting, and CLI/report capability visibility are in place. The remaining gap is true SDK-backed probing and any backend-specific capability enrichment that comes with it.
- `add-accel-backend-discovery` now also has portable native-probe fallback chains: Windows no longer depends only on `Get-CimInstance`, and Linux no longer depends only on `lspci`; this makes Intel/AMD iGPU and portable GPU detection more resilient before backend SDK bindings exist.
- `add-accel-backend-discovery` now also honors `DiscoveryTimeout`, supports host-memory-derived shared-memory budgets when native budget data is missing, and avoids the earlier gap where native probe tests and config fields existed without working code behind them.
- `add-accel-observability-fallback` now has code behind it: stable fallback reason codes, strict-gpu failure reports, discovery-error reporting, and core metrics are wired into `accel.Report` and CLI output.
- `add-accel-scheduler-multi-gpu` now has a real planning contract in code: `PlanShardable()` / `PlanShardableWorkload(...)` produce weighted per-device assignments, deterministic merge-policy selection, and profitability-aware fallback for auto mode. True execution merge paths still do not exist.
- `accel` now also has an execution seam in code: `ExecuteProjectedDataset(...)`, `ExecuteDataList(...)`, and `ExecuteDataTable(...)` materialize truthful per-device residency through builtin `CUDA` / `Metal` / `WebGPU` allocator stubs plus ledger fallback and update execution metrics/report state without pretending real GPU kernels already run.
- `add-accel-columnar-layout-cache` is now complete enough to close the current slice: typed projection emits validity bitmaps, string offsets/data transport, lineage-aware session-local cache identity, aggregate budget enforcement, eviction metrics, and truthful cache output that does not pretend projection buffers are already resident on every shardable device.
- `add-accel-cli-dsl-surface` remains partially implemented. `accel cache` now shows truthful session-local resident state, the Cobra `--mode` path now works end-to-end, `accel plan` stays planning-only, and `accel run <var>` now drives the execution ledger; there is still no device allocator or true backend-native workload execution surface.
- `add-accel-backend-discovery` now also has an SDK probe seam in code: `RegisterSDKProbe` / `SDKProbe` / `ResetSDKProbesForTest` route SDK-backed discoverers ahead of native command probes and env stubs, and `ProbeSourceSDK` plus a new `devices.sdk` metric let observers tell SDK-discovered devices apart from native and env-stub ones.
- The accel runtime now ships a Windows NVML SDK probe via stdlib `syscall.LoadLibrary` and `SyscallN` (no cgo), populating `Device.DriverVersion`, `Device.ComputeCapability`, and (via the enriched nvidia-smi parser) `Device.PCIBusID`. `INSYRA_ACCEL_DISABLE_NVML_SDK=1` lets tests opt out on hosts that ship NVIDIA drivers; the CLI `accel devices` output now also prints `driver=`, `compute=`, and `pci=` columns.
- 2026-07-28 audit: `accel` cannot execute anything on a GPU. `shader`, `dispatch`, `kernel`, `WGSL`, `SPIR-V`, and `PTX` appear zero times in the package's non-test code, `ExecutionResult` has no field that could hold a computed value, and `Buffer.Values` is read at only two sites, both of which take `len()` or format the values into a fingerprint hash. What the CLI prints as `execution.bytes_moved` comes from `bufferBytes/8` and `bufferBytes/2` constants in `accel/allocator_builtin.go`.
- 2026-07-28 audit: across the eight accel changes, 41 tasks were closed and the only 3 open ones belong to `add-accel-string-kernels-phase-2`, a deferral change. The previously named next change was 10/10 complete, so the entry sequence led to an empty queue. That is why M2, M4, and M5 are now marked done and M6 was opened.
- Measured on an Apple M3 with `CGO_ENABLED=0`, 16M `float32` (64 MiB): one column sum including upload and readback took 17.5 ms against 13.8 ms on the CPU, so the GPU lost; the same column resident on the device ran 2.0 ms per pass against 4.7 ms; a compute-bound kernel with 64 transcendental terms per element took 69.7 ms against 8.10 s. Transfer is roughly thirty times the dispatch for a plain reduction, which means `shouldFallbackForProfitability` in `accel/planner.go` is inverted — it judges a 16M-row sum profitable and the measurement says otherwise. Rewriting it needs the measured cost table that `add-accel-gpu-execution` produces, so it is deliberately not part of that change.
- `go test ./accel/...` currently fails 4 tests on any macOS host, and has done since before the 2026-07-28 rebase onto `dev`. The tests override only the CUDA and WebGPU builtin probes, so a real Metal device leaks in and the device counts come out one too high. Recorded under `## Follow-ups` in `AGENTS.md`.
- `gogpu` registers a software backend on every non-Android platform. It is a SPIR-V interpreter that reports `DeviceType: DeviceTypeCPU` and `Name: "Software Renderer"`, and while `core.selectAdapterIDs` prefers a hardware adapter when one exists, on a host with no GPU driver `RequestAdapter` still succeeds and hands back the software adapter. Accepting it would report a CPU interpreter as an acceleration device, so `add-accel-gpu-execution` rejects CPU-class adapters by requirement.
- Open question for the heterogeneous multi-GPU v1 decision: `gogpu`'s public `wgpu.Instance` exposes only `RequestAdapter`, and adapter enumeration lives on `core.Instance.EnumerateAdapters()`. Reaching every GPU on a multi-GPU host therefore means depending on `gogpu`'s `core` package rather than its stable public surface. `add-accel-gpu-execution` ships single-device execution only; the v1 multi-GPU default needs a decision once there is a measurement to justify the cost.
- 2026-07-28: M6 is met. On an Apple M3 with `CGO_ENABLED=0`, a 1 Mi `float64` column narrowed to `float32` was uploaded to `metal:wgpu:0`, reduced by a WGSL compute shader, and read back with a value matching the CPU reference: transfer 0.93 ms, dispatch 0.20 ms, readback 3.76 ms. Run it with `INSYRA_ACCEL_GPU_TESTS=1 go test ./...` inside `accel/backend/wgpu`.
- The execution seam changed shape. `BackendAllocator` / `AllocationRecord` / `AllocatorKind` are gone, replaced by `BackendExecutor` / `ExecuteRequest` / `ExecuteResponse` / `ExecutorKind`, and `ExecutionResult` now carries `Reductions`, `Counts`, `Precision`, `Transfer`, `Dispatch`, `Readback`, and `BytesUploaded`. `accel/allocator_builtin.go` and the `execution.bytes_moved` metric are deleted; the CLI no longer prints invented transfer figures.
- Precision is opt-in and the shape changed during implementation. An earlier draft refused `float64` outright, which would have made the GPU path unreachable for real data — `projectValues` types nearly every numeric column as `float64`, and WGSL has no `i64` either. `WorkloadEstimate.Precision` now defaults to `PrecisionExact` and refuses; `PrecisionFloat32` opts in. Only the within-workgroup tree runs at single precision, the host folds partials in `float64`.
- The first real benchmark found the actual bottleneck, and it is not the GPU. At 4 Mi the device does its work in 4.6 ms while the whole operation takes 354 ms, and `session.ProjectDataList` alone accounts for 357 ms. `datasetFingerprint` formats the entire column with `fmt.Sprintf("%v", buffer.Values)` before hashing (`accel/cache.go:332`). Recorded under `## Follow-ups` in `AGENTS.md`. Fix this before attempting the profitability rewrite — the current numbers cannot support one.
- 2026-07-28: `fix-accel-fingerprint-cost` landed. `datasetFingerprint` no longer renders columns with `fmt.Sprintf("%v", ...)`; it encodes raw value bits through a 32 KiB scratch buffer into xxhash. On a 4 Mi `float64` column the fingerprint went from 255 ms to 14.2 ms (17.9x, 132 MB/s to 2355 MB/s), `ProjectDataList` from 357 ms to 145 ms, and the end-to-end GPU column sum from 354 ms to 117 ms. `cespare/xxhash/v2` is promoted from indirect to direct; nothing new enters the build.
- The order-of-magnitude target on `BenchmarkProjectionOnly` was **not** met by that change, and the reason is now measured rather than guessed: `BenchmarkProjectValues` is 94 ms of the remaining 145 ms. `insyra.DataList` stores `[]any`, so projection unboxes every element. That is structural and needs its own change.
- The `accel/backend/wgpu` module is gone; the gogpu mechanics live in `accel/internal/wgpu` (a leaf package that imports nothing from `accel`) and the adapter that registers them is `accel/backend_wgpu.go`. There is one module again, so the re-tidy step that a core dependency change used to require has gone with it.
- Making the backend register itself introduced an order-dependent test failure: `ResetDiscoverersForTest` clears SDK probes and never restores them, so a test asserting "no devices" passed only when some earlier test had already wiped the registry. `INSYRA_ACCEL_DISABLE_NATIVE_PROBES` now also disables the builtin GPU probe, and every test in `./accel` passes when run individually.
- 2026-07-28: `speed-up-accel-projection` landed. `projectValues` no longer dispatches through `reflect` per element — `reflect.Value.Convert` was heap-allocating for every value, costing a 4 Mi column 4,194,308 allocations to produce numbers it already held. Concrete type switches with a reflect fallback for named types took it to 4 allocations. The validity bitmap then became the largest remaining node and was folded into the projection loop. `BenchmarkProjectValues` went 94 ms to 13 ms (7.2x, 354 MB/s to 2587 MB/s).
- Cumulative effect of the two host-cost changes on a 4 Mi `float64` column: `ProjectDataList` 357 ms to 43 ms (8.3x), end-to-end GPU column sum 354 ms to 47.6 ms (7.4x). Device work for the same column is 5.6 ms, so host cost is now roughly eight times the device rather than seventy. The remainder splits about evenly between projection, fingerprinting, and allocation, with nothing dominant left to remove.
- The 4 macOS accel test failures are fixed. `isolateBuiltinProbes` in `accel/testing_test.go` reports every builtin backend unavailable so a host GPU cannot leak into a test, and `TestIsolatedBuiltinProbesFindNoHostDevices` is the regression guard. `go test ./...` is green on macOS.
