# Delivery Status

## Current Phase
`insyra/nn` phase 2 (training): the MLP, attention, and CNN operator families are done, and M21 is complete — the tape now trains a seeded 784→128→10 MLP to ≥95% MNIST test accuracy with a sane loss curve. `insyra/ml` v1 is shipped, audited and merged to dev (PR #194). Acceleration now has one opt-in wired call site, KNN; large 2-D `nn` MatMul is default-on with CPU fallback.

## Stage Objective
`insyra/nn`: run ONNX models in pure Go, verified per-operator and per-model against `onnxruntime`, with the op families landing in the decided order MLP → attention → CNN, then phase 2 adds autodiff and optimisers on the same tensors (first-step gradients verified against PyTorch under fixed initial weights via SafeTensors). GGUF/LLM is a decided future track that reuses the kernels; only two v1 constraints serve it now — dtype-carrying tensors and kernels as plain functions. M23 is complete: half precision is storage-only, with exact f16/bf16 widening into f32.

## Active Workstreams
None pending review. `add-dl-mnist-convergence` (M21) is implemented and ready to archive after merge; `openspec/changes/` otherwise holds only `archive/`.

## Milestones
| id | target | owner | status | verification_signal |
| --- | --- | --- | --- | --- |
| M6 | One operation executes on a real device | planning | done | a `float32` column reduced on a discovered device returned the CPU's value, with measured transfer/dispatch/readback replacing fabricated constants |
| M7 | Foundations for default-on acceleration | planning | done | `Session` safe for concurrent use, the execution seam carries a whole dataset in one submission, `accel.Default()` provides a process-shared session |
| M8 | Find a call site a device wins | planning | done | answered, negatively. 96 shapes measured against an all-core host: the device wins 42, by at most 3.63x, and no `stats` call site clears the threshold. Three operations were removed rather than shipped |
| M9 | `stats` can be wrapped | planning | done | every regression predicts and matches R's `predict()`; KMeans assigns new rows; logistic and Poisson publish their link; PCA returns its projection parameters |
| M10 | `insyra/ml` v1 | planning | done | protocol, pipelines, model selection, decision trees and ONNX export archived; 21 review findings raised, the blocking ones fixed, the rest adjudicated |
| M11 | `ml` holds up under audit | planning | done | ONNX round-trip passes against a real `onnxruntime` for all five model shapes; `stats` refuses input it cannot read instead of scoring zeros; metrics declare a direction; pipelines name the columns their estimator saw |
| M12 | A skip cannot pass for a pass | planning | done | strict mode fails on a missing toolchain and skips without it, verified in both directions; every gate routes through one helper; CI installs all three toolchains and runs with it set |
| M13 | An ONNX MLP runs in pure Go | planning | done | a PyTorch-class MLP `.onnx` loads and predicts in `nn`, every kernel passing generated one-op parity against `onnxruntime` and the whole model matching within f32 tolerance |
| M14 | `nn` models join the `ml` protocol | planning | done | a `nn` model satisfies `ml.Model`, takes `DataTable` input, and `ml`'s own exports read back and run |
| M15 | Attention-family ops | planning | done | a fixed-weight two-head encoder block — batched MatMul, axis-Softmax, Gelu FFN, residuals, LayerNormalization — matches `onnxruntime` end to end; every operator carries one-op parity rows; `INSYRA_NN_REAL_MODEL` smokes local models. Kernels stay plain functions the future llm package can call |
| M16 | CNN-family ops | planning | done | a fixed-weight MNIST-class CNN — Conv, BatchNormalization, pooling, Pad — matches `onnxruntime` end to end, with one-op parity enumerating the attribute combinations rather than sampling defaults |
| M19 | An honest CPU baseline for dl's hot kernels | planning | done | MatMul and Conv use all cores with exact output parity. Across four best-of-5 runs on the 8-core M3 (idle to loaded): encoder layer 3.35s → 0.73–1.11s (3.0x–4.6x, ~3.8x reproducible idle), CNN forward 526ms → 97–169ms (3.1x–5.4x, ~4.4x reproducible idle). The encoder sum keeps ~90ms of deliberately serial small ops; its MatMul share alone is ~4.4x. Ordered before M17 — a device claim measured against one core has been withdrawn once already and will not be manufactured again |
| M17 | Inference reaches the device | planning | done | large 2-D MatMul runs on the device through `accel.DeviceMatMul` by default, bit-equal to the CPU on hardware (asserted with ==); the floor is measured at 16Mi MACs with the 4M–8M noise band refused; dl falls back observably when the backend is unavailable; the measured encoder layer dropped from ~0.9s all-core CPU to 234ms (14.3x over the pre-M19 serial baseline). Batched products stay CPU by measurement |
| M18 | Training (phase 2) | planning | done | SafeTensors, the tape (MLP VJPs, fused softmax–cross-entropy, SGD), attention-family gradients, Adam, and CNN VJPs are complete — a fixed two-head encoder block and a deterministic MNIST-class CNN each take one Adam step in dl and PyTorch agrees on loss, every gradient, and every post-step parameter; every VJP is pinned by ungated finite differences |
| M20 | Real checkpoints run | planning | done | `mobilenetv2-12.onnx` and `minilm-l6-v2.onnx` run in dl and match `onnxruntime` on deterministic fixed inputs. The gated test covers `Clip`, `ConstantOfShape`, runtime-computed int64 shape tensors, and the required broadcast/Cast/Where paths |
| M21 | Training converges for real | planning | done | a fixed-seed 784→128→10 MLP reaches 95.84% test accuracy in two epochs; mean training loss is 0.350281 then 0.163855; a dataset-free micro-convergence test reaches 100% |
| M22 | Training practice ops | planning | done | inverted dropout (seeded, frozen-mask finite-difference), AdamW with a coupled-vs-decoupled divergence assertion, and StepLR — a five-step AdamW+StepLR trajectory matches PyTorch on loss and every parameter at every step |
| M23 | f16/bf16 | planning | done | SafeTensors and ONNX f16/bf16 values widen bit-exactly into f32; Cast rounding and PyTorch/onnxruntime gates pass |
| M24 | Performance is positioned honestly | planning | done | measured at the validated real-model shapes on the M3, best of 5: MobileNetV2 batch-1 170ms vs ORT's 7.3ms (~23x), MiniLM b8×s128 3.34s vs 63ms (~53x); disabling the device shows the gap is CPU-structural (pure Go vs per-architecture assembly), so the decision is a written positioning in Docs/nn.md rather than an assembly chase — the GEMM microkernel lever stays unbuilt until a workload demands it |

| M25 | Sequential layer surface | planning | in progress | Layer interface (Build/Forward/Parameters, TrainingOnly marker), two-phase construction, torch-name weight interop; the Sequential MNIST run reproduces the hand-written curve digit-for-digit under the same seed |
| M26 | The complete layer catalog | planning | pending | Conv2D, pooling, training-mode BatchNorm2D (new VJP with running stats vs torch train mode), LayerNorm, Embedding (scatter-add VJP vs torch); a Sequential CNN converges on MNIST and a torch CNN loads and predicts |

Milestone order is the blocking sequence. OpenSpec has no dependency relationship between changes, so nothing else carries it.

## Current Blockers
None. Hardware coverage remains Apple/Metal-only, carried as the standing `AGENTS.md` follow-up.

## Next Verifiable Output
M25: the Sequential surface with the digit-for-digit MNIST reproduction; then M26 completes the layer catalog.

## Next Ticket
M24 the honest performance position. Two measured negatives remain guardrails: no batched device kernel without a single-dispatch batched measurement, and no device Conv without its own measurement.

Note for any host running the reference suites locally: the crosslang venv moved to `~/.cache/insyra-crosslang-venv` on 2026-08-03 after macOS's tmp cleaner destroyed the old /private/tmp venv (deleted `pyvenv.cfg` and parts of numpy's binaries, producing no-module false negatives). CI is unaffected — it installs its own toolchains.

## Decision Log
Deltas that still change what someone would do. The standing technical decisions they produced — the precision contract, the device rules, the measured thresholds — live in [ENG.md](ENG.md); the full history is in git.

- decision: The performance gap to ONNX Runtime is positioned, not chased.
  rationale: Measured at the validated real-model shapes (M3, best of 5, identical inputs): MobileNetV2 batch-1 170ms vs 7.3ms, MiniLM b8×s128 3.34s vs 63ms — 23x and 53x. Disabling the device path shows the gap lives in CPU kernel throughput: ORT's MLAS runs hand-written per-architecture assembly at ~250 GFLOP/s where pure Go reaches ~4-8, and the Go compiler does not auto-vectorize. Assembly would surrender the portability and auditability the package exists for, so Docs/nn.md now states the honest numbers and the choice they imply. The one lever that moves everything — an assembly GEMM microkernel — is recorded and stays unbuilt until a real workload demands it.
  timestamp: 2026-08-03
  impacted_ticket_ids: none; a future assembly-GEMM change if demand appears

- decision: `nn` device MatMul is default-on; the opt-in bridge is removed.
  rationale: The dependency-cycle argument only held for the bridge package. Direct `dl → accel → insyra` is acyclic, and the measured compile cost is affordable at about 1.9 seconds of cold build time and 200 KB. The measured 8.9x–52x device win clears the bar the root package did not. `INSYRA_ACCEL_DISABLE_WGPU=1` disables the backend, and `nn.RegisterDeviceMatMul(nil)` clears the hook programmatically.
  timestamp: 2026-08-03
  impacted_ticket_ids: make-dl-device-matmul-default

- decision: Device matmul is earned for large single-dispatch shapes, bit-identically on Apple/Metal; per-batch small dispatches are refused by measurement.
  rationale: Measured on the M3 against the M19 all-core baseline, best of 5, upload+dispatch+readback included: [4096,256]×[256,256] 9.3ms vs 82.4ms (8.9x), FFN up 58.5ms vs 683ms (11.7x), FFN down 55.4ms vs 730ms (13.2x), [4096,4096]² 1.87s vs 97.2s (52x) — and every shape, including the losers, returned maxULP=0 because the prototype's per-output thread accumulates along k in the CPU's serial order. The attention-shaped batched products lose (1.08x–2.08x against) when driven as 128 separate dispatches, so they stay on the CPU until a single-dispatch batched kernel is measured. Bit parity is an observation about Apple/Metal, not a property of the kernel; the wiring must assert it per platform and fall back to CPU where it fails, which the Apple-only follow-up already anticipates.
  timestamp: 2026-08-02
  impacted_ticket_ids: add-dl-device-matmul-measurement (archived), the M17 wiring change

- decision: The device-matmul floor is 16Mi MACs, measured, and the noise band below it is refused.
  rationale: The hardware ladder (best of 5, upload+dispatch+readback, all-core CPU opponent) crosses over near 4M MACs, but 4M–8M sit inside the noise band (device/CPU 0.90 and 0.96, flipping run to run); 16Mi is the first rung that wins dependably (0.74, improving monotonically to 0.135 at 268M). Every rung returned maxULP=0. A floor placed at the crossover would trade bit-identical dependability for wins that evaporate under load, so the floor sits at the first dependable rung instead.
  timestamp: 2026-08-02
  impacted_ticket_ids: add-dl-device-matmul

- decision: dl's device milestone waits for a parallel CPU baseline; the ticket cut from the measurement is pure-CPU.
  rationale: Profiled on an 8-core M3 at realistic sizes, MatMul is 98% of an encoder layer (3.37 s) and Conv is 98% of a CNN forward (530 ms) — and both are single-threaded naive loops. A GPU measured against that baseline would repeat the withdrawn one-core comparison with a larger multiplier. Output-element parallelism preserves each element's accumulation order, so the CPU win is bit-identical and costs no precision-contract decision, unlike the device path, which still owes a per-platform bit-parity answer.
  timestamp: 2026-08-02
  impacted_ticket_ids: add-dl-parallel-cpu-kernels, the future M17 change

- decision: KNN is the first operation wired to the device, behind two measured floors — and the first benchmark overstated it.
  rationale: The transposed benchmark (dataset=train) won every shape up to 4.1x; re-measured in the wiring's own direction (dataset=test, the side the kernel parallelises over), the device LOSES below ~2k test rows — 469ms against the CPU's 324ms at 1k — because its wall time is nearly flat in test rows until it saturates. Above the floor it wins 1.4x (2k), 2.9x (4k), 3.7x (10k) on 100k×32. accel/knnbridge therefore gates on per-row work ≥ 2048 AND test rows ≥ 2048, and device parity with brute force is asserted exactly on hardware. The lesson is the same one this project keeps re-learning: the direction of a measurement is part of the measurement.
  timestamp: 2026-08-01
  impacted_ticket_ids: archived

- decision: A verification that skips is not a verification, and the suite has to be able to say so. Gates route through `internal/reftest`; `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` makes a missing toolchain a failure.
  rationale: The ONNX round-trip needed `onnxruntime`, which was on no machine it ever ran on, so it skipped everywhere and reported nothing. Executed for the first time, it failed immediately on two defects that made every exported model unloadable. Installing the toolchains was only half the fix — measured in a clean environment, the `Clustering Parity` workflow's own installs left `sklearn` missing, so the job dedicated to running the parity suite had been green while running none of it. Toolchains go missing again; the suite has to fail rather than shrug.
  timestamp: 2026-08-01
  impacted_ticket_ids: archived

- decision: A conversion that cannot fail is a conversion that fabricates. Numeric input in `stats` goes through one validating helper.
  rationale: Regression and correlation read every value through `ToF64Slice`, which yields 0 for anything unparseable and returns a full-length slice regardless. One blank among six observations moved a Pearson coefficient from 0.9992 to 0.9879, silently. Three treatments of unreadable values now coexist deliberately — refusal, listwise deletion, a learned tree direction — and each is documented; substituting a zero is not one of them.
  timestamp: 2026-08-01
  impacted_ticket_ids: archived

- decision: Do not ship a control that provably cannot change the answer.
  rationale: A positive-class option for `ROCAUCMetric` was proposed, built, and then withdrawn on measurement: the metric receives the whole probability table, so naming the other class swaps both the label and the score column and the two cancel exactly — all three choices returned 0.50838574423480087. It was the only one of six reported findings that had not been measured before being reported.
  timestamp: 2026-08-01
  impacted_ticket_ids: archived

- decision: Judge every acceleration claim against a host using all its cores.
  rationale: `accel` contains no parallelism at all, while the rest of the library parallelises its hot loops as a matter of course. Every speedup recorded before this compared a GPU against one core on an eight-core machine, so none described the alternative a user actually has. Re-measured honestly, the best case fell from 7.1x to 3.63x.
  timestamp: 2026-07-29
  impacted_ticket_ids: all prior accel measurements

- decision: Decide whether a device may be used from the shape of the result, not from how hot the operation is.
  rationale: A selection can be made exact by having the device propose and the host decide; a new value cannot. This puts CCL on both sides of the line — its predicates qualify, its value expressions do not, despite being the 11x and 146x measurements. That is the cost of choosing bit parity, chosen knowingly.
  timestamp: 2026-07-28
  impacted_ticket_ids: future kernel work

- decision: Remove every accelerated operation except exact-nearest, and require a measurement before adding any back.
  rationale: Three of four were known not to pay. Keeping them meant a kernel to maintain against a single-maintainer backend, a method to document, and a changelog entry claiming a speedup measured against one core.
  timestamp: 2026-07-29
  impacted_ticket_ids: archived

- decision: `insyra/ml` imitates scikit-learn, with fitting returning a separate value.
  rationale: Not a preference — the library had already imitated it in four places without saying so. `datatable_scale.go` ships `Scaler` with sklearn's four verbs and the same four scaler names; `stats/knn.go` reproduces `KNeighborsClassifier`'s options verbatim. Consistency with itself outranks any other precedent. What does not translate is the machinery: sklearn discovers parameters by reflecting over `__init__`, which Go cannot do, so a pipeline step is a fit function rather than a configured object.
  timestamp: 2026-07-29
  impacted_ticket_ids: archived

- decision: `ml` wraps `stats` and reimplements nothing, except the one family `stats` lacks.
  rationale: The numerics are already validated against R. Only decision trees are written from scratch — the family with no `float64` legacy, the one the ONNX corpus is full of, and the one whose split finding is selection-shaped.
  timestamp: 2026-07-29
  impacted_ticket_ids: archived

- decision: Keep the root package reaching acceleration through a seam, and switch to a direct import only when a root operation is measured to win.
  rationale: `accel` imports the root package, so the root cannot import `accel`. Moving the device layer would fix that and costs a hello-world 1.9 s of cold build and 200 KB — affordable, but nothing in the root package clears the threshold, so there is no reason yet.
  timestamp: 2026-07-29
  impacted_ticket_ids: future root-package work

- decision: Withdraw the string-kernel deferral rather than leave it open.
  rationale: It deferred work to a Phase 2 that no longer exists, and the result-shape rule would exclude string kernels regardless of phasing. Deferring something to a future that has been ruled out is not a plan.
  timestamp: 2026-08-01
  impacted_ticket_ids: archived

- decision: A fix is not verified until the test covering it has been shown to fail without it, and a claim about code is not established until the code has been read.
  rationale: The first half caught three unearned claims, including one of my own — a probe written to prove a conformance fix passed with the fix disabled, because the fake model was failing an unrelated assertion first. The second half was learned the hard way: `SimpleImputer` was described in three commit messages and in `ENG.md` as having claimed a reversibility it lacked, on the strength of a handoff summary. It never did. It deliberately omits `InverseTransform` and its comment explains why, which is the same reasoning applied correctly. The false premise also reached a review brief as the exemplar of a failure class.
  timestamp: 2026-08-01
  impacted_ticket_ids: none

## Source Links
- [ENG.md](ENG.md) — architecture, test seams, the precision contract, standing assumptions. Read before changing any of them.
- [AGENTS.md](AGENTS.md) — the operating contract, including the acceleration rules and the open follow-ups.
- [openspec/changes/archive/](openspec/changes/archive/) — 41 archived changes, each holding its own proposal, spec deltas and tasks.
- [openspec/specs/](openspec/specs/) — the current capability specs, which reflect the code as it stands.
- [Docs/accel.md](Docs/accel.md), [Docs/ml.md](Docs/ml.md) — the user-facing surfaces.
- Open issues: [#190](https://github.com/HazelnutParadise/insyra/issues/190) KNN algorithm selection, [#191](https://github.com/HazelnutParadise/insyra/issues/191) CCL recursion-depth overhead.

## Handoff Notes
- **M17 wiring handoff.** `make-dl-device-matmul-default` now contains the default-on `nn` hook, exported `accel.DeviceMatMul`, production WGSL MatMul, CPU fallback tests, a hardware parity gate, and the runnable floor ladder. `INSYRA_ACCEL_DISABLE_WGPU=1` and `nn.RegisterDeviceMatMul(nil)` are the two switches. `delivery-status.md` changed in this handoff; `AGENTS.md` did not.
- **M18 training handoff.** `add-dl-cnn-gradients` adds direct-loop Conv, pooling, and inference-mode BatchNormalization VJPs, ungated finite-difference coverage, and a gated PyTorch SafeTensors CNN one-step parity test. The full `./dl/` suite passed with the moved reference venv. `delivery-status.md` changed in this handoff; `AGENTS.md` did not.
- **M20 real-checkpoint handoff.** `add-dl-real-model-support` adds `Clip`, `ConstantOfShape`, runtime control tensors, deterministic real-model parity for MobileNetV2 and MiniLM-L6-v2, and the matching docs, changelogs, and skill note. The normal and strict `./dl/` suites passed; `delivery-status.md` changed in this handoff; `AGENTS.md` did not.
- **M21 convergence handoff.** `add-dl-mnist-convergence` adds test-side IDX validation, fixed-seed He initialization, a dataset-free micro-convergence test, and a gated MNIST 784→128→10 Adam run. The specified MNIST command reaches 93.91% after epoch 1 and 95.84% after epoch 2 in 12.5 seconds; the uncached `./dl/` suite passes, and missing `INSYRA_NN_MNIST_DIR` skips cleanly. `delivery-status.md` changed in this handoff; `AGENTS.md` did not.
- **M23 half-precision handoff.** `add-dl-half-precision` adds exact f16/bf16 widening for SafeTensors and ONNX initializers, round-to-nearest-even Cast targets that widen back to f32, hand-built boundary tests, a PyTorch SafeTensors fixture, and an ONNXRuntime parity row. The specified strict `./dl/` suite passes in 20.445 seconds; `delivery-status.md` changed in this handoff; `AGENTS.md` did not.
- **Hardware blocker.** The sandbox only exposes a software adapter. The new device tests skip with `ErrUnavailable`; 2.1's measured floor, 3.3's all-`nn` hardware run, and 3.4's encoder timing remain unchecked.
- **One sample-weight design question remains open, deliberately.** Tree sample weights would push float weights into histogram accumulators the precision contract fixed as integers for associativity — an architecture decision against ENG.md, not a feature. The cross-validation channel turned out not to need a protocol break: scikit-learn routes sample_weight to fit and scores unweighted, so `Estimator` gained an optional `FitWeighted` and `CrossValidateWeighted` subsets weights with each fold's own indices. The metric protocol is untouched.
- **Two pure-CPU wins are measured and unclaimed**, and both are worth more than anything acceleration offered. `stats`' KNN auto-selection picks a ball tree up to 3.3x slower than parallel brute force on unstructured data (#190). CCL spends 4.8x–6.2x more time on recursion-depth bookkeeping than on evaluating (#191).
- **A skipped verification now fails when it was supposed to run.** `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` turns every missing-toolchain skip into a failure, and the `Reference Verification` workflow installs R, the Python scientific stack, scikit-learn and onnxruntime and runs with it set. Before this, `Clustering Parity` had been reporting green while running nothing — its gate imports `sklearn` and the workflow never installed it.
- **The refusal of unreadable numeric input is a breaking change** for anyone whose data contains blanks. What used to return a number now returns an error; the number it used to return was wrong. `ToF64Slice` still backs 54 call sites in `plot`, `gplot`, `quant` and the CLI — display paths where a zero is visible rather than laundered into a coefficient. A new numeric analysis must not join them.
- `nn` now imports `accel` directly for default-on large MatMul wiring. Other callers still reach `accel` through `allpkgs` or a direct import.
- **The race detector cannot reach the device on macOS**, upstream and pre-existing (gogpu/wgpu#280). Device tests skip under the `race` build tag.
