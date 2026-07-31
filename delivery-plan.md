# Delivery Plan

## Current Phase
Phase 2 - `insyra/ml` v1 shipped. Acceleration is dormant by choice: no call site was measured to win, and the runtime waits for one.

## Stage Objective
Run one operation end to end on a real device and return a number that matches the CPU path. Every accel surface built so far — discovery, projection, cache, scheduler, CLI — has been verified against stubs only, so the first correct GPU number is what turns the existing scaffolding from assumption into fact.

## Active Workstreams
- `M7`: roadmap stage A — the foundations default-on acceleration needs. Complete: `Session` is safe for concurrent use, the execution seam carries a whole dataset in one submission, and `accel.Default()` provides a process-shared session.

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
Every regression family in `stats` can score new data, and its predictions match R's `predict()` — including the five methods that already shipped and have never been checked against a reference. This is the first of several `stats` gaps that must close before `insyra/ml` can wrap it, and it is the one everything else depends on, because a prediction-shaped package cannot be built over results that cannot predict.

Superseded, for the record: the acceleration phase's own next output was undecided, and deliberately so. The exact-nearest operation works and is honestly measured, but the caller it was built for does not clear its own profitability threshold, so committing to a next kernel before the remaining candidates are measured would repeat the mistake. The two worth measuring are brute-force KNN, whose candidate count is the training-set size and therefore always large enough, and the DBSCAN neighbourhood scan. Both need measuring against a parallel host before anything is proposed.

## Next OpenSpec Change
All four `stats` changes are implemented, verified and archived: `add-stats-regression-predict`, `add-kmeans-assign`, `expose-glm-link`, `add-pca-transform-parameters`. `stats` can now predict, assign, publish its links and project new observations, so nothing in it blocks `insyra/ml` any longer. `add-kmeans-assign` is also the acceleration foothold: assigning a row to the nearest centre is the only selection-shaped operation `stats` has, and therefore the only one a device may accelerate by default.

All five planned `insyra/ml` changes are implemented and verified: `add-ml-estimator-protocol`, `add-ml-pipeline`, `add-ml-model-selection`, `add-ml-decision-tree`, and `add-ml-onnx-export`. The protocol change carries a `design.md` with the actual interfaces, so the shape is settled rather than left to be invented during implementation. The KNN question is dropped — v1 carries decision trees, so it no longer needs KNN to be worth installing.

Still open on the acceleration side, and not blocking: measuring brute-force KNN and the DBSCAN neighbourhood scan against a parallel host, and re-measuring the older accel figures that were taken against a single core. `add-accel-gpu-execution` task 1.13 (non-Apple verification) stays open in parallel; it needs a Windows or Linux machine with an NVIDIA or AMD GPU and cannot be done on this host.

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

- decision: Adopt as the accel end state: acceleration enabled by default, routed by a measurement-calibrated dispatcher that sends each workload to CPU, GPU, or a weighted split of both — never by moving work to the device unconditionally.
  rationale: Direction set by the project owner on 2026-07-28, and the measurements support it: the CPU/GPU boundary sits near 1–2.5 flop/byte, single-pass reductions never pay, and residency multiplies the win. Staged path: (A) make `Session` safe for concurrent use, add a process-shared default session, widen the execution seam to multiple columns; (B) ship kernels where measurement says the device wins — pairwise/distance for stats clustering and correlation, CCL expression chains, IRLS-style resident loops; (C) replace `shouldFallbackForProfitability` with a cost model calibrated from the execution ledger's measured transfer/dispatch/readback, and extend the weighted shard planner so the CPU participates as a compute resource; (D) flip the default per operation, gated on the parity requirement below.
  timestamp: 2026-07-28
  impacted_change_ids: `add-accel-gpu-execution`, `fold-gpu-backend-into-core`, future kernel and dispatcher changes
- decision: Default-on GPU execution requires bit-level parity with the CPU result.
  rationale: Chosen by the project owner over a documented-tolerance regime, after the pairwise measurement spike showed 0.2 relative error from naive f32 accumulation. Because float addition is not associative, parity is only well-defined against a deterministic reference reduction that both sides implement — a parallel sum cannot reproduce the CPU's sequential rounding order. Consequences: each GPU operation ships with a deterministic reference algorithm (double-float accumulation or correctly-rounded reduction), the CPU verification path computes the same reference, and an operation that cannot reach bit parity stays on the CPU rather than shipping as default. The existing `PrecisionFloat32` opt-in remains available for callers who explicitly accept narrowed results.
  timestamp: 2026-07-28
  impacted_change_ids: future kernel changes, `add-accel-gpu-execution`
- decision: The end state gives every user acceleration with zero code change, including programs that import only the root package. `allpkgs` includes `accel` now; root-level availability comes last, via dependency inversion.
  rationale: Set by the project owner, overriding the earlier recommendation to stop at subpackages, and the owner's case is stronger than the recommendation was: CCL lives in the root package (`AddColUsingCCL`, `ExecuteCCL`), and expression chains measured 11x on the device — so root-level default-on is not paying for unprofitable Sum/Mean acceleration, it is what puts the most-used profitable surface on the GPU. The dispatcher keeps memory-bound root operations on the CPU automatically. Reaching it requires inverting a dependency: accel's runtime core must stop importing the root package (its `Dataset`/`Buffer` layer already does not), with the DataList/DataTable-facing helpers moving so the root can import the dispatcher without a cycle. Accepted cost: every insyra consumer compiles the gogpu stack once that inversion lands. Staging: `allpkgs` includes `accel` immediately (registration is lazy, so the only cost is compile time); subpackage wiring next; root inversion last, after the dispatcher and parity gates exist.
  timestamp: 2026-07-28
  impacted_change_ids: `fold-gpu-backend-into-core`, future kernel and dispatcher changes

- decision: Do not build GPU kernels for operations the measurements say the device loses. Those stay on the CPU permanently, and the profitable ones target CPU and GPU running together rather than the GPU alone.
  rationale: Set by the project owner on 2026-07-28. The measured crossover sits near 1-2.5 flop/byte: a column sum was 0.7x and an affine map 0.3x against a plain Go loop, while a transform chain with transcendentals hit 11x, a heavy transcendental kernel 146x, and a pairwise n-squared kernel 62x. Writing kernels below the crossover would cost maintenance to ship a slowdown. In scope: pairwise and distance work (`stats` clustering, correlation, KNN), CCL expression chains carrying SQRT/LOG/EXP/POW, and resident multi-pass loops such as IRLS. Out of scope permanently: `Sum`, `Mean`, `Min`, `Max`, `Count`, single elementwise maps, and filters. Out of scope structurally: `DataList.Map` over a Go closure, `finance` decimal arithmetic, serial recurrences (`CumSum`, exponential smoothing), and sort-dominated statistics (`Median`, `Quartile`). For in-scope operations the dispatcher splits one workload across CPU and GPU by measured throughput rather than choosing one, extending the existing weighted shard planner to treat the CPU as a compute resource.
  timestamp: 2026-07-28
  impacted_change_ids: future kernel and dispatcher changes
  note: `OpSum` — the one operation implemented today — is on the permanent CPU list. It is scaffolding: it holds the end-to-end proving path and the benchmark harness only until a profitable kernel can take that role, and is then removed rather than kept. It must never become default-on.

- decision: Deep learning is a separate future subpackage that reuses the device layer, not an extension of the accel runtime.
  rationale: Set by the project owner on 2026-07-28. What transfers is the expensive, platform-specific half: adapter probing, software-adapter refusal, unified-versus-discrete memory classification, pipeline caching, chunked dispatch under the 65535 workgroup limit, staging readback, the cgo-free build across Metal, Vulkan and DX12, and the fallback and report surface. What does not transfer is everything above it — `Dataset` and `Buffer` are one-dimensional and bound to DataList projection where tensors need rank, strides and broadcasting; `ExecuteRequest` carries one column and one operation where training needs a graph; the content-hashed LRU cache is the wrong lifetime model for tensors that must stay resident by explicit ownership; and the CPU/GPU dispatcher is irrelevant to a workload that always wants the device. Autodiff, tensor types and GEMM kernels are all new work, and none of them depend on the accel runtime.
  timestamp: 2026-07-28
  impacted_change_ids: future deep-learning package
  constraint: The device layer sits at `accel/internal/wgpu`, so Go's internal rule blocks any sibling package from importing it — verified by compiling a probe package, which fails with "use of internal package not allowed". When the second consumer appears, move it to `insyra/internal/gpu` or a public `insyra/gpu`. Do not move it before then; one consumer belongs in `internal/`.
  shared_primitive: A device buffer that outlives a single call is needed by both the accel residency phase and any tensor runtime — `internal/wgpu.Sum` currently creates and releases five buffers per invocation. Build it in the device layer rather than in accel's `Dataset` layer so both consumers get it once.

- decision: Bit-level parity is a per-platform property verified by a test, not a portable guarantee, and the CPU reference is written in the natural fused form.
  rationale: Measured on 2026-07-28, Apple M3 + Metal, over 4096 outputs of a squared-distance kernel with the operation order pinned identically on both sides. GPU output is bit-identical to a Go reference written as `acc + diff*diff`, and differs from one written as `acc + float32(diff*diff)` in 1137 of 4096 outputs by up to 2 ulp. The reason is contraction, not associativity: Metal fuses multiply-add, Go fuses it too on arm64, and they fuse the same way. Go's fusion survives assignment to a named `float32` — only an explicit `float32(a*b)` conversion forbids it, confirmed against a hand-computed unfused value.
  consequence: Parity is not portable by construction. Go emits FMA on arm64 but not on amd64, so the same reference on an x86 host would stop fusing while a GPU backend still might, and parity would break on exactly the platforms not yet tested. The gate therefore has to run on the target: each operation ships a bit-parity test against its CPU reference, an operation that fails parity on a platform stays on the CPU there, and the fallback reason says so. Writing the reference in the explicitly-rounded form would be the wrong instinct — it guarantees divergence on the one platform measured.
  timestamp: 2026-07-28
  impacted_change_ids: future kernel changes

- decision: Accelerate existing float64 APIs by letting the GPU shortlist and the CPU decide, rather than by moving the arithmetic to the device.
  rationale: Measured on 2026-07-28, Apple M3, 200,000 rows by 16 dimensions against 64 centres, the shape `stats.KMeans` uses. The GPU computes every distance in f32 and returns only the four most likely centres per row; the CPU then recomputes those four in float64 and picks the nearest and second nearest. Result: 138.7 ms of pure float64 becomes 10.2 ms on the device plus 8.0 ms of verification, a 7.6x improvement, with **zero** assignments differing from the float64 reference across all 200,000 rows. Only 12 rows (0.006%) had a shortlist boundary inside the f32 error bound and were recomputed against every centre. This resolves what looked like a hard conflict between the roadmap's automatic-acceleration goal, the bit-parity requirement, and the fact that WebGPU has no f64: the final arithmetic stays float64, so the numbers cannot change, while the device removes most of the work. The pattern generalises to any operation that ranks or finds extrema before computing precisely on a few candidates — k-nearest neighbours, radius queries, picking the strongest correlations.
  timestamp: 2026-07-28
  impacted_change_ids: future kernel and dispatcher changes
  caveat: Measured on synthetic data with the f32 error bound taken from an earlier measurement on this host. Real data with many equidistant or duplicated points will raise the fallback rate, and the bound plus the boundary test need re-verifying against real inputs before the pattern is relied on.
  correction: 2026-07-29. The 7.6x above is overstated, because the float64 baseline it was measured against read its values straight out of the column-major buffers and so paid a cache miss per column per query point. Gathering the row once first makes the same baseline 13x faster. Against that baseline, on the same host and shape, the shipped implementation measures no gain at 16 query points, 3.2x at 64, and 7.1x at 256 — the technique holds and the answer is still exactly the float64 one, but the win depends on the query count and there is a floor below which the runtime declines the device. The rigorous per-row error bound now used in place of the spike's fixed 3e-7 does not change this: 0 of 200,000 rows needed a full recompute either way.
- decision: Handle an unavailable, unsupported, or failing GPU by falling back to the verification path that already exists, so device failure is a performance event and never a correctness one.
  rationale: The shortlist pattern makes this nearly free. The CPU already recomputes candidates in float64, and already recomputes a row in full when the shortlist cannot be trusted — so "no GPU" is just "every row takes the full path", which is the original CPU implementation unchanged. There is no second code path to keep correct, and no output difference to explain: whether the device is absent, refuses the operation, fails to compile a shader, times out on readback, or fails the platform's parity gate, the answer is identical and only the time changes. The existing fallback reasons already name which of those happened.
  timestamp: 2026-07-28
  impacted_change_ids: future kernel and dispatcher changes
- decision: Split the library into three precision tiers — `insyra` and its existing subpackages stay float64, `insyra/ml` carries classical machine learning designed for the device, and `insyra/dl` later carries deep learning.
  rationale: Set by the project owner. The existing packages keep exact float64 results and gain speed only through the shortlist pattern above, which does not change their numbers. `insyra/ml` is designed f32-first, so its CPU and device paths agree bit for bit by construction and the dispatcher can choose either or split between them — the roadmap's automatic CPU-and-GPU cooperation is achievable there rather than merely approximated. `insyra/dl` follows the same shape and reuses the device layer, as already decided. Keeping the tiers separate means a user always knows which guarantee they are getting, and `stats` users never compile a GPU stack they did not ask for.
  timestamp: 2026-07-28
  impacted_change_ids: future `insyra/ml` and `insyra/dl` work

- decision: Decide what the root package may accelerate by the shape of the result, not by how hot the operation is. Three tiers: a result that is a selection may be accelerated by default, a column the device holds exactly may be accelerated by default, and a result that is a new float64 value may only be accelerated when the caller opts into reduced precision.
  rationale: The bit-parity gate is what makes acceleration safe to turn on without asking, and only some result shapes can clear it. When the answer is a selection — which row, which index, what order — the device's f32 answer is a proposal, and the CPU settles the few boundary cases in float64, so the final answer is exact. When the column is a native float32, an integer inside int32 range, or a boolean, the device is exact outright and needs no verification. When the answer is a new float64 value, there is nothing cheaper than the computation to verify it against, and WebGPU has no f64 to compute it with, so parity cannot be recovered at any price. This puts CCL on both sides of the line: its predicates are selections and qualify, while its value-producing expressions do not, despite being the 11x and 146x measurements. That is a real cost of choosing bit parity, and it was chosen knowingly.
  timestamp: 2026-07-28
  impacted_change_ids: future root-package and `insyra/ml` work
- decision: Keep the root package reaching acceleration through a registration seam for now, and switch it to importing the device layer directly once a root-package operation is measured to win on a device.
  rationale: The dependency runs the wrong way today — `accel` imports the root package, so the root package cannot import `accel`. Two routes fix that: a seam the root declares and `accel` fills from `init`, or moving the device layer to a package with no insyra dependency so the root can import it outright. The second is what the end goal needs, since only a direct import reaches a program that imports nothing else, and it is affordable: measured on this host, linking the device layer costs a hello-world 1.9 s of cold build time, 200 KB of binary, and 41 extra packages. `accel/internal/wgpu` already depends on zero insyra packages, so the move is mechanical. What is missing is a reason: the tier rules above admit very few root-package operations, because root operations are mostly memory-bound and the device loses those. The wins measured so far — 7.6x on cluster assignment, 13.6x on nearest query — are all in `stats` and `ml` territory. So the seam goes in now at no cost to anyone, and the direct import waits for the first root operation that earns it. The measurement above means that switch is a scheduling call, not an architectural one.
  timestamp: 2026-07-28
  impacted_change_ids: future root-package work

- decision: Measure every acceleration claim against a host using all its cores, and treat every figure recorded before 2026-07-29 as overstated until re-measured.
  rationale: Every speedup this project has recorded compared a GPU against one CPU core on an eight-core machine. `accel` contains no parallelism at all — the single `go func` in the package is a discovery timeout — while the rest of the library parallelises its hot loops as a matter of course: `stats/internal/clustering/cluster.go` splits five of them, including the very KMeans assignment step this work was aiming at, and `stats/internal/knn` fans out across goroutines. So the comparison was never against the alternative a user actually has. Re-measured honestly on 96 shapes, exact nearest wins in 42 of them and never by more than 3.63x, against figures of 3.2x and 7.1x recorded the day before. The 13.6x recorded for `ExecuteNearestQuery`, and the 11x, 62x and 146x from the arithmetic-intensity sweep, were obtained the same way and are unreliable by up to the core count.
  timestamp: 2026-07-29
  impacted_change_ids: all prior accel measurements
- decision: Decide whether to use a device from the arithmetic one row carries — dimensions times query points — rather than from the query count.
  rationale: The two trade off directly, and a rule reading only one of them is wrong in both directions. Measured on an Apple M3 against a host using all eight cores: four dimensions need 512 query points before the device is worth it, sixteen need 64 to 128, sixty-four need 32. Those cluster near 2048 distance evaluations per row, and no other threshold misclassifies fewer of the 96 shapes — 2048 gets 88 right, against 84 at 4096 and 78 at 8192. The earlier floor of 32 query points was both dimension-blind and calibrated against a single core.
  timestamp: 2026-07-29
  impacted_change_ids: `speed-up-accel-exact-host-path`
- decision: Do not wire exact nearest into `stats.KMeans`.
  rationale: Three findings, each sufficient on its own. The cluster count is the query count, and `stats` callers use single digits to low tens, far below the 2048 evaluations per row where a device starts winning. Only the initial assignment is a bulk nearest computation; the main loop is Hartigan-Wong optimal-transfer and quick-transfer, which recompute distances incrementally per candidate transfer and have no shortlist to narrow. And that initial assignment is already parallel across cores, with its own measured threshold, at `stats/internal/clustering/cluster.go:198`. Candidates that do fit the shape, in order of promise: brute-force KNN at `stats/internal/knn/knn.go:275`, where the candidate count is the training-set size and so always clears the threshold; and the DBSCAN neighbourhood scan at `stats/internal/clustering/kdrange.go:66`, which compares every point against every other and whose result is a selection, though it needs a range kernel rather than a shortlist one.
  timestamp: 2026-07-29
  impacted_change_ids: future `insyra/ml` work

- decision: Judge a device against the fastest option the library already offers, which is often an algorithm rather than a parallel loop.
  rationale: Measured through the public `stats.KNearestNeighbors` API on 2026-07-29, 20,000 training rows and 1,000 test rows, five neighbours, against the device path. On unstructured data the device beats the best CPU option by 1.0x to 1.5x depending on dimension. On data with cluster structure it loses to the ball tree everywhere measured, by between 1.4x and 2.7x, because the tree stops looking at most of the candidates while the device still evaluates every one. Tree pruning is worth far more than a device — at 20,000 rows and 64 dimensions the ball tree takes 86 ms where parallel brute force takes 189 ms — and no threshold reading only shape can tell the two data regimes apart. So the device is not a general answer for nearest-neighbour work in this library; it is an answer for data with no exploitable structure, which the runtime cannot detect in advance.
  timestamp: 2026-07-29
  impacted_change_ids: future `insyra/ml` work
- decision: Treat `stats`' KNN algorithm choice as a separate, CPU-only question worth measuring on its own.
  rationale: `resolveAlgorithm` at `stats/internal/knn/knn.go:258` picks by row count and dimension alone: brute below 64 rows, a kd-tree at eight dimensions or fewer, a ball tree otherwise. Measured, that rule costs up to 3.3x on unstructured data — 50,000 rows by 64 dimensions takes 1.483 s through the ball tree it selects against 452 ms for parallel brute force — while on clustered data the same rule is right and the ball tree wins by roughly 2x. The rule cannot see the property that decides which is faster. This is a pure CPU improvement, larger than anything the device has offered so far, and it needs no acceleration work to collect.
  timestamp: 2026-07-29
  impacted_change_ids: none yet
  tracked_at: https://github.com/HazelnutParadise/insyra/issues/190
- decision: Fix three defects an adversarial review found in exact nearest before considering it done.
  rationale: A review panel reading the implementation returned three blocking findings, two of which reproduce. Asking for nine or more neighbours on an accelerated session panicked with an index out of range, because the shortlist width was clamped to the device's eight slots while the decision still indexed position m-1; the device is now skipped when it cannot hold the request. When every single-precision distance overflows to infinity the device's ordering is meaningless and its boundary is infinite, which passed the trust test for the wrong reason and left the row trusted; the host now checks the device's distances are finite before believing their order. The third, that a purely relative error bound says nothing once a squared term falls below the smallest normal float32, is sound as an argument but no failing case was constructed — the bound was widened anyway, along with a missing rounding of the difference itself.
  timestamp: 2026-07-29
  impacted_change_ids: `add-accel-exact-nearest`

- decision: The root package holds nothing worth accelerating by default.
  rationale: Surveyed on 2026-07-29 across `datalist.go`, `datatable.go`, `internal/ccl/`, `internal/algorithms/`, `internal/core/` and `parallel/`, judging every arithmetic or ordering operation against the measured rules. Seventy-one of seventy-four candidates fail outright, almost all for the same two reasons: the work is one pass over a column, which the device loses permanently, and the values live in `[]any`, so reaching the device means a pointer-chasing type switch that costs more than the arithmetic it feeds. `Sum`, `Mean`, `Var`, `Range` and the rest are the operations rule three names. `Sort` and `Rank` are ordering-shaped and would qualify on shape, but they compare mixed types through `CompareAny` — numbers, strings and times in one comparator, not expressible on a device — and already use every core. The three survivors are all CCL, and all three produce new float64 values, which the bit-parity rule puts behind an explicit opt-in whatever their speed.
  timestamp: 2026-07-29
  impacted_change_ids: future root-package work
- decision: Withdraw the 11x recorded for CCL transform chains, and with it the case for accelerating CCL at all.
  rationale: That figure was measured against the CCL interpreter, and the interpreter spends most of its time not interpreting. `evaluateWithContext` and `callFunction` each maintain a recursion-depth counter in a process-wide `sync.Map` keyed by goroutine id, costing a `goid.Get`, two map operations and a deferred closure with two more, per AST node per row. Removing both guards makes `AddColUsingCCL` over `SQRT(A*A + B*B) * 2 + A / B - 1` go from 15.05 ms to 3.11 ms at ten thousand rows and from 141.59 ms to 22.76 ms at a hundred thousand — 4.8x and 6.2x. Removing them is not the fix, since they prevent stack overflow, but threading the depth as a parameter would recover nearly all of it at the cost of one register. A CPU profile attributes only about eleven percent to it, which is wrong; the A/B measurement is what to trust. Against a repaired interpreter running on every core, the 11x is not a speedup at all. Tracked at https://github.com/HazelnutParadise/insyra/issues/191.
  timestamp: 2026-07-29
  impacted_change_ids: future root-package work
- decision: If `insyra/ml` is built to connect with the machine-learning ecosystem, the connection to build is ONNX export, not an API resembling a Go machine-learning library.
  rationale: Verified against the GitHub API on 2026-07-29. There is no living Go classical-machine-learning library to interoperate with. Gorgonia last shipped a release on 2023-12-03 and last took a commit on 2024-08-12; GoLearn last moved on 2024-01-15 and has never cut a tagged release; onnx-go was archived by its author, transferred, and last touched on 2024-09-02; spago's author paused it in January 2024 pointing at Rust's Candle; gota is archived. The two alive projects are GoMLX, which shipped v0.28.0 on 2026-07-21 and needs cgo with prebuilt shared libraries, and gonum, which is numerics rather than machine learning. What Go is actually used for in this space is infrastructure — Ollama, LocalAI, Milvus, Weaviate, Kubeflow — and Google's 2025 Go Developer Survey reports machine-learning involvement among Go developers falling fourteen points year on year to eleven percent. The established route is train in Python, export to ONNX, serve in Go, and every healthy Go library in the area is a consumer of an artifact produced elsewhere. ONNX's `ai.onnx.ml` domain covers exactly the classical models this package would produce, and an `.onnx` file is protobuf, so writing one needs no C dependency and no runtime. Export is cheap and makes an Insyra model readable by Python, Netron, Triton, C# and the browser; import is expensive and should be scoped narrowly if attempted at all.
  timestamp: 2026-07-29
  impacted_change_ids: future `insyra/ml` work
- decision: Do not expect a discrete GPU to rescue the acceleration case, and do not plan around it without measuring.
  rationale: The hardware headroom is real — an RTX 4090 has 1008 GB/s and 82.6 TFLOPS against a base M3's roughly 100 GB/s — but three things consume it. Host-to-device transfer over PCIe 4.0 x16 measures around 20 to 26 GB/s, a cost unified memory does not pay, which raises the arithmetic-intensity threshold rather than lowering it: the 2048-evaluations-per-row floor would be higher on NVIDIA, not lower. Machines with such a card also have larger CPUs, and the baseline is all of them. And the pure-Go WebGPU path is far from hardware peak — the only public figure naming hardware, born-ml/born on an RTX 3080, reports a 1024-cubed matmul at 58 ms, about 37 GFLOPS against that card's 29.77 TFLOPS peak, roughly a tenth of a percent; Cogent Core's axon README states outright that A100 performance is comparable to an M1. A reasonable expectation is 5x to 20x against an all-core CPU for resident, high-intensity work, and worse than Apple for anything crossing PCIe per call. This is inference from public figures and architecture, not measurement, and task 1.13 remains the way to settle it.
  timestamp: 2026-07-29
  impacted_change_ids: `add-accel-gpu-execution`
- decision: Treat the pure-Go GPU stack as a single-maintainer dependency and plan for its absence.
  rationale: gogpu/wgpu, naga, goffi and go-webgpu are substantially one person's work, the repository has 159 stars, and its CI runs only GitHub-hosted runners with Mesa software rendering on Linux — no NVIDIA or AMD hardware is exercised anywhere in it. It is active, having pushed v0.30.23 on 2026-07-26, but the bus factor is one. The observable-fallback design already means its disappearance is a performance event rather than a correctness one, which is the mitigation; the point of recording it is that no roadmap item should assume it will still be maintained. Two other projects now occupy the same pure-Go-machine-learning-on-WebGPU position — born-ml/born since November 2025 and georgebuilds/anneal since May 2026 — so `insyra/ml` should differentiate on the DataTable-to-model data path and on exact-by-construction numerics rather than by writing another tensor and autodiff layer.
  timestamp: 2026-07-29
  impacted_change_ids: future `insyra/ml` and `insyra/dl` work

- decision: Remove every accelerated operation except the exact nearest one, and require a measurement against an all-core host before any is added back.
  rationale: Three of the four were known not to pay, and each was a standing claim that acceleration helps where it does not. The column sum measured 0.7x — it moves one value per element and adds it once, which no device wins. The squared-distance matrix reads back a result growing with rows times query points, which is what the nearest-query operation was built to fix. And the single-precision nearest query answers in f32, which the float64 callers it was written for cannot use, so it was superseded by its own successor. Keeping them was not free: each was a kernel to maintain against a single-maintainer WebGPU backend, a method to document, and a changelog entry claiming a speedup measured against one core. What remains is the shortlist kernel, `ExecuteNearestExact`, and the runtime around it — discovery, cache, scheduler, and the fallback that makes a missing device a performance event rather than a correctness one. That runtime is the seam a future kernel lands in and would cost far more to rebuild than to keep. Nothing removed had shipped; none of it exists in v0.3.0.
  timestamp: 2026-07-29
  impacted_change_ids: `remove-unprofitable-accel-ops`
- decision: Imitate scikit-learn for the `insyra/ml` estimator protocol and public API, because Insyra already chose it.
  rationale: Not a preference — a fact in the working tree, found by reading it. `datatable_scale.go:36` declares `Scaler` with `Fit`, `Transform`, `FitTransform` and `InverseTransform`, and `datatable_scale.go:95-109` provides `NewStandardScaler`, `NewMinMaxScaler`, `NewRobustScaler` and `NewMaxAbsScaler` — scikit-learn's `preprocessing` module, the same four names and the same protocol. `datatable_encode.go` carries `OneHotEncoder`, `LabelEncoder` and `OrdinalEncoder` with `DropFirst` and `Unknown`, matching `drop='first'` and `handle_unknown`. `stats/knn.go:14-27` reproduces `KNeighborsClassifier`'s options verbatim, down to `uniform`/`distance` and `auto`/`brute`/`kd_tree`/`ball_tree`. So the library has been imitating scikit-learn in four places without saying so, and consistency with itself outranks any argument for a different precedent. What does not translate is the machinery underneath: scikit-learn discovers parameters by reflecting over `__init__`, clones estimators by calling the constructor with a string-keyed dict, and distinguishes fitted from unfitted by the presence of trailing-underscore attributes. Go has none of that, so the vocabulary is scikit-learn's while the typing follows Spark MLlib's shape, where fitting produces a separate Model object rather than mutating the estimator.
  timestamp: 2026-07-29
  impacted_change_ids: future `insyra/ml` work

- decision: `insyra/ml` does not have a precision. It has one precision contract, assigned by role, and every model in it obeys the same one.
  rationale: This supersedes both the f32-first tier decision of 2026-07-28 and the f32-capable amendment first drafted to replace it. Both argued at the level of the package — is `ml` single or double — and the primary-source survey of what mainstream libraries actually do says that axis is wrong. No library in the corpus picks one type. They assign precision by what the number is used for, and for the hottest role the answer is neither single nor double.

    The contract, each row sourced: bulk feature values in float32, or a quantised integer where they only ever feed comparisons — scikit-learn's tree module copies a float64 X down to float32 in the same `fit` call that copies a float32 y up to float64, and states the roles in the declarations themselves (`DTYPE_t # Type of X`, `DOUBLE_t # Type of y, sample_weight`). Classification labels as integers or booleans, exact and free. Regression targets entering a residual in float64. Terms feeding a long reduction widened at read time in bounded chunks rather than stored wide, which is what `_euclidean_distances_upcast` does. Any comparison key that decides a selection resolved at the widest precision available, on the host, always — scikit-learn's `_argkmin` heap is float64 in both its float32 and float64 specialisations, with no dissent found anywhere. Stored model parameters and reported values in float64, which is also what keeps `ml` lined up against a float64 `stats` and makes ONNX export lossless.

    The accumulator is the row that matters and the one nobody would guess: fixed-point integers, not float64. Bit-exactness under an arbitrary CPU-and-device split requires an accumulator that is associative, not merely deterministic, because the dispatcher is allowed to move the split point and every regrouping of a floating-point sum is a different answer. Fixed point is associative. XGBoost removed its single-precision histogram option in 1.7 with the wording "dangerous to use" and replaced it with `GradientPairInt64`; it arrived there because GPU atomics reorder, we arrive there because a dispatcher partitions, and it is the same fix.

    This dissolves the contradiction rather than trading it away. A model wrapping `stats` is not a "float64 model" sitting awkwardly inside an f32 package — it is a model whose reported values are float64, which is what the contract requires of every model. It and a decision tree with float32 features, fixed-point accumulation and float64 leaves obey the same rule, and a caller gets the same guarantee from both: reported values are float64 and bit-identical to calling `stats` directly.
  timestamp: 2026-07-29
  impacted_change_ids: future `insyra/ml` work
- decision: `insyra/ml` v1 ships decision trees, the one model family `stats` does not have.
  rationale: Set by the project owner. It also answers the reviewers' strongest objection to the plan: a v1 that only wraps what already exists is a shim, and nobody installs a shim. Trees are the right family to add rather than any other — they are what the import corpus is full of, so the scorer written for them serves both training and any later ONNX import; they are the family with no float64 legacy to preserve, so they can follow the role contract from the first line; and split finding is selection-shaped, which under the acceleration rules is the one shape a device may accelerate by default. The precision arrangement is not ours to invent: scikit-learn's tree module and XGBoost's quantised gradients are where the role contract above was derived from in the first place.
  timestamp: 2026-07-29
  impacted_change_ids: future `insyra/ml` work
- decision: Decision-tree split finding is device-eligible but unmeasured in this change, so it remains a CPU implementation and no kernel is proposed.
  rationale: Selecting the best split is selection-shaped, but eligibility is not evidence that a device wins. The tree uses float32 feature bins, fixed-point integer histograms, and a deterministic CPU tie-breaker; a GPU path needs a separate benchmark against the all-core CPU implementation and a host-side exact decision before it can change the default.
  timestamp: 2026-07-31
  impacted_change_ids: `add-ml-decision-tree`, future `insyra/ml` work
- decision: `stats` cannot be wrapped as it stands. Fix it in `stats` first, in separate changes, before any of `insyra/ml` is written.
  rationale: Verified by listing every exported method on a fitted result: there are seven, and five of them are prediction, all in the GLM family. `LinearRegressionResult`, `PolynomialRegressionResult`, `ExponentialRegressionResult` and `LogarithmicRegressionResult` have no `Predict` at all. `KMeansResult` carries `Cluster` and `Centers` but nothing assigns a new row to a centre. `PCAResult` lacks the means, standard deviations and scores that a `Transform` or an ONNX export would need. The GLM `Predict` that does exist takes a variadic of columns rather than a table and does not record which columns it was fitted on, `link` is unexported on all three, and an offset model refuses to predict outright. A prediction-shaped package cannot be built over that from the outside.
  timestamp: 2026-07-29
  impacted_change_ids: `add-stats-regression-predict`
- decision: Withdraw the claim that wrapping `stats` inherits its R validation. It inherits it for fitting only.
  rationale: `stats/testdata` holds 22 `.R` reference scripts and the cross-language tests shell out to `Rscript` and compare field by field — but not one of the 22 calls `predict()`. The validation covers fitting. The prediction path has never been checked against a reference, including the five `Predict` methods that already ship. Since `insyra/ml` is prediction-shaped, wrapping that path and describing it as R-validated would be exactly the kind of unearned claim this phase has spent its time removing. The prediction path gets its own references before anything wraps it.
  timestamp: 2026-07-29
  impacted_change_ids: `add-stats-regression-predict`

- decision: The role-based precision contract does not require rewriting anything in `stats`. It sorts `stats` into what can be accelerated as it stands and what stays on the CPU as it stands.
  rationale: The contract binds the inside of a model, not its interface, and the only thing it promises a caller is that reported values are float64 — which `stats` already produces. What it settles is the internals, and it settles them by the row that says a comparison key deciding a selection is resolved at the widest precision available, on the host, always. That is a description of the shortlist pattern. So any `stats` algorithm whose hot step is a selection — cluster assignment, nearest neighbours, a neighbourhood scan — can be accelerated without `stats` changing at all: the device proposes in single precision and `stats`' own float64 code is the host-side decision the contract asks for. Any algorithm whose hot step is a reduction producing values — the normal equations, an eigendecomposition, iteratively reweighted least squares — could only be accelerated by replacing its accumulator with fixed point, which means reimplementing mathematics that is already validated against R. So it is not accelerated; it is reused unchanged and runs on the CPU. Preprocessing is the same: elementwise, memory-bound, reused unchanged. Only the new tree family is written to the contract from the inside, and it is the family the contract was derived from in the first place.
  timestamp: 2026-07-29
  impacted_change_ids: future `insyra/ml` work

- decision: The four `stats` prerequisites are implemented, verified against R and Python, and archived. `insyra/ml` is unblocked.
  rationale: Verified on 2026-07-29 rather than accepted. The four regressions that could not predict now can, `KMeansResult.Assign` applies fitted centres to new observations, the logistic and Poisson results publish their link, and `PCAResult` returns the centring, scaling and training scores a projection needs. Every scenario in the four specs has a test behind it, and the two that matter most had never run on this machine: `TestKMeansAssign_R` and `TestCrossLangRegressionPredictions` were skipping because neither `Rscript` nor a `python` with the scientific stack was installed. Installing R 4.6.1 with jsonlite, cluster and dbscan, and building a virtual environment with numpy, scipy, statsmodels and scikit-learn, made both run — and both pass, the cross-language one taking 23.7 s of real comparison against both references. The full `stats` suite is 220 passed, 10 skipped, 0 failed, with every skip an opt-in slow corpus behind its own environment variable.
  timestamp: 2026-07-29
  impacted_change_ids: archived
- decision: Guard nil tables in the shared loader rather than in the one entry point that exposed the panic.
  rationale: `KMeansResult.Assign(nil)` panicked, and following it down showed the fault was never Assign's. Every clustering entry point loads through `numericMatrixFromTable` and `PCA` through its own equivalent, and both dereference the table before validating it — so `KMeans`, `DBSCAN`, `Silhouette`, `HierarchicalAgglomerative` and `PCA` all panicked the same way, on a nil interface and on a typed nil alike, and had done so long before any of this work. Putting the guard in Assign alone would have made a new method stricter than its siblings for no reason a caller could see. It goes in the loader, `PCA` gets its own copy since it does not use the loader, and a regression test covers six entry points against both kinds of nil.
  timestamp: 2026-07-29
  impacted_change_ids: none

- decision: A pipeline step in `insyra/ml` is a fit function, not a configured estimator object. This is what replaces scikit-learn's `clone`.
  rationale: scikit-learn refits a step per cross-validation fold by cloning the configured but unfitted estimator, and cloning reads the constructor's parameters back off the instance through `inspect.signature`. Go has no equivalent that is not reflection over struct tags, which this repository uses nowhere. A closure refits by being called again: configuration is captured at the call site in ordinary Go and checked by the compiler, cross-validation calls the function once per fold, and grid search closes over each combination. Three independent designs converged on this without conferring, which is the strongest signal any of them produced.
  timestamp: 2026-07-29
  impacted_change_ids: `add-ml-estimator-protocol`, `add-ml-pipeline`, `add-ml-model-selection`
- decision: `ml.Transformer` is typed to match `insyra.Scaler.Transform` and `insyra.Encoder.Transform` exactly, concrete `*DataTable` on both sides.
  rationale: Not an aesthetic choice. With that signature, all four scalers and all three encoders satisfy the interface with no adapter and no change to the root package — verified by compiling the assertions against the tree as it stands. Widening either side to `IDataTable` breaks it, and that reuse is the largest single piece of foundation `ml` inherits. `Model.Predict` takes the same concrete type for symmetry; the interface-to-concrete direction is the one that costs nothing, since `stats` functions already accept the interface.
  timestamp: 2026-07-29
  impacted_change_ids: `add-ml-estimator-protocol`
- decision: `ml.Model` requires `Features() []string`, which scikit-learn treats as advisory.
  rationale: Nothing in `stats` records which columns a model was fitted on. `LinearRegressionResult`, `GLMResult` and `KMeansResult` all omit it, and `predictFromCoefficients` validates only the count — so passing predictors in the wrong order silently produces wrong numbers rather than an error. `ml` matches columns by name against this list, which closes that. It is the one place `ml` is deliberately stricter than what it wraps.
  timestamp: 2026-07-29
  impacted_change_ids: `add-ml-estimator-protocol`

- decision: Add a fitted imputer to the root package. Imputation is the one preprocessing operation Insyra offers only as an in-place mutation, and that form cannot be used by a model.
  rationale: Found by Codex while starting the ONNX export change, which listed imputers among the things to export when nothing produces one. Checking why went further than the export list. `FillWithMean`, `FillWithMedian`, `FillWithMode` and the rest exist on both `DataList` and `DataTable`, and every one computes its statistic from whatever data it is called on and mutates it in place. That is right for cleaning a table by hand and wrong for a model: a model must impute new observations with the training statistic, and imputing them with their own is leakage that raises no error and makes the score look better than it is. `insyra.StandardScaler` already has the correct shape — fit, remember, transform — so scaling is safe and imputation is the exception. The concrete consequence is that `FillWithMean(cols ...string) *DataTable` returns no error and is a method on a table rather than an operation over one, so it does not satisfy the transformer protocol at all: `insyra/ml` pipelines currently have no way to handle missing data. The imputer follows the `Scaler` interface, the in-place methods stay as they are, and ONNX export of an imputer becomes possible once it lands.
  timestamp: 2026-07-29
  impacted_change_ids: `add-fitted-imputer`, `add-ml-pipeline`, `add-ml-onnx-export`

- decision: `insyra/ml` v1 is implemented, reviewed and archived. All five changes are in, and the remaining open changes are the two accel leftovers.
  rationale: Reviewed adversarially rather than accepted — four reviewers, one per change, each required to write a failing test rather than argue, and each finding attacked by an independent agent whose default was that it was wrong. Twenty-one findings came back, nine of them blocking, and the strongest were all the same shape: an interface satisfied in form while the substance was missing. `LogisticModel` had `Classes()` so a type assertion reported it a `Classifier`, and its `Predict` returned probabilities — `ml.Accuracy` scored it exactly zero with a nil error. A fitted pipeline implemented only `Model`, silently dropping the wrapped estimator's `Classifier`, `ProbaModel` and `Importances`. `FitPoissonRegression` accepted an offset, fitted successfully, and returned a model whose `Predict` could never work. All were fixed before this verification finished, and each was independently re-probed here rather than taken on trust.
  timestamp: 2026-08-01
  impacted_change_ids: archived
  caveat: The adversarial verification did not complete. Fifteen of the twenty-one attack agents failed on a weekly usage limit, so most findings below blocking severity were never independently challenged. Nine blocking claims were made; five were re-probed directly here and all five are fixed; one was confirmed by an attacker before the limit hit and is fixed; one is now https://github.com/HazelnutParadise/insyra/issues/192; two were addressed in the implementer's own remediation and not independently re-checked. The twelve important and minor findings are reviewer claims, not confirmed defects, and remain unexamined.
- decision: The permutation test is what makes the tree's precision claim real, and it passes.
  rationale: Fixed-point accumulation is only worth its complexity if it buys order-independence, and order-independence is only believable if it is measured. Six hundred rows carrying magnitudes from 1e-3 to 1e3, permuted, fit a bit-identical tree. A float64 accumulator cannot pass that test, because summing the same values in a different order gives a different sum — which is the whole reason XGBoost left floating point for `GradientPairInt64`. The implementer's own suite covers the same property, so the claim now has two independent tests behind it.
  timestamp: 2026-08-01
  impacted_change_ids: archived

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
- 2026-07-28: `accel.Session` is safe for concurrent use. One mutex guards every field; public methods lock and internal callers use `*Locked` bodies, because Go mutexes are not reentrant and several public methods call one another. The lock is held across backend execution deliberately — releasing it around the device call would expose half-updated cache and report state. Projection stays outside the lock, since it touches no session state and is the expensive step. `internal/wgpu.Sum` is serialized process-wide, because every session shares one device handle and gogpu's queue-concurrency guarantees are unverified.
- No test or benchmark that reaches a device can run under `-race` on macOS: `-race` enables `checkptr`, which aborts inside gogpu's Metal completion-block trampoline. Upstream and pre-existing — reproduced at the previous commit. `requireGPU` and `gpuTestsEnabled` skip under the `race` build tag; recorded under `## Follow-ups` in `AGENTS.md`.
- 2026-07-28: roadmap stage A is complete. `accel.Default()` gives the process one lazily created session — discovery runs once, the resident cache is shared so a column can stay on the device across operations, and `Close` on it is a no-op because library code holding it cannot know it is shared. Importing the package still opens no device, verified with a consumer that imports `accel` and never calls `Default`.
- The execution seam now carries a whole dataset. One submission encodes every column's dispatch and copy into a single command buffer landing in a single staging buffer, so a wide table waits on one map instead of one per column. Measured over 262,144 rows per column: readback at eight columns fell from 5.52 ms to 1.25 ms, and the whole operation from 36.8 ms to 24.2 ms. The first attempt collapsed only the runtime loop and measured nothing — the batching has to reach the command buffer.
- Reported gogpu/wgpu#280 upstream: Metal compute aborts under `-race` because checkptr rejects pointer arithmetic in the GPU completion block. Includes a standalone reproduction.
- 2026-07-28: the first stage-B kernel landed. `OpSquaredDistance` computes the squared Euclidean distance from every row to each query point, and the parity gate passes on this host — 160,000 values bit-identical at 16 queries and 6,400,000 at 64, spanning multiple dispatches. `SquaredDistancesCPU` is the reference, written in the natural fused form the parity decision requires.
- The device wins, but by 1.7x at 16 queries and 3.2x at 64 rather than the 62x the spike suggested, because this operation materialises the whole rows-by-queries matrix and readback dominates: 16 ms of the 18 ms at 64 queries. The next slice folds argmin into the kernel so KMeans and KNN get one value per row instead of one per pair.
- 2026-07-28: `OpNearestQuery` collapses the distance matrix on the device, reporting the closest query point per row. At 100,000 rows and 16 dimensions the device takes 5.47 ms at 16 query points and 5.92 ms at 64, while the CPU goes 23.5 to 80.5 ms — 4.3x and 13.6x, and 3.1x better than the matrix kernel at 64 purely from not sending the matrix back. Parity holds across 200,000 rows against 32 queries, every index and distance bit-identical.
- Note on the reported cost: `Submit` is asynchronous, so the readback timer captures the device finishing as well as the copy. For compute-heavy kernels the readback figure is mostly the device working. It is the right number for deciding whether to dispatch, but not bus time.
- The 4 macOS accel test failures are fixed. `isolateBuiltinProbes` in `accel/testing_test.go` reports every builtin backend unavailable so a host GPU cannot leak into a test, and `TestIsolatedBuiltinProbesFindNoHostDevices` is the regression guard. `go test ./...` is green on macOS.
