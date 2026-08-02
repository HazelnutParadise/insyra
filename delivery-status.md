# Delivery Status

## Current Phase
`insyra/dl` phase 1 (ONNX inference): the MLP, attention, and CNN operator families are all done and archived — the decided op-family order is complete. Next is M17 (device inference, measured) and M18 (training). `insyra/ml` v1 is shipped, audited and merged to dev (PR #194). Acceleration has exactly one wired call site (KNN), which measurement earned.

## Stage Objective
`insyra/dl`: run ONNX models in pure Go, verified per-operator and per-model against `onnxruntime`, with the op families landing in the decided order MLP → attention → CNN, then phase 2 adds autodiff and optimisers on the same tensors (first-step gradients verified against PyTorch under fixed initial weights via SafeTensors). GGUF/LLM is a decided future track that reuses the kernels; only two v1 constraints serve it now — dtype-carrying tensors and kernels as plain functions.

## Active Workstreams
`add-dl-parallel-cpu-kernels` (M19, ordered before M17) — implementation and verification complete; operator review and commit happen here.

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
| M13 | An ONNX MLP runs in pure Go | planning | done | a PyTorch-class MLP `.onnx` loads and predicts in `dl`, every kernel passing generated one-op parity against `onnxruntime` and the whole model matching within f32 tolerance |
| M14 | `dl` models join the `ml` protocol | planning | done | a `dl` model satisfies `ml.Model`, takes `DataTable` input, and `ml`'s own exports read back and run |
| M15 | Attention-family ops | planning | done | a fixed-weight two-head encoder block — batched MatMul, axis-Softmax, Gelu FFN, residuals, LayerNormalization — matches `onnxruntime` end to end; every operator carries one-op parity rows; `INSYRA_DL_REAL_MODEL` smokes local models. Kernels stay plain functions the future llm package can call |
| M16 | CNN-family ops | planning | done | a fixed-weight MNIST-class CNN — Conv, BatchNormalization, pooling, Pad — matches `onnxruntime` end to end, with one-op parity enumerating the attribute combinations rather than sampling defaults |
| M19 | An honest CPU baseline for dl's hot kernels | planning | done | MatMul and Conv use all cores with exact output parity. Across four best-of-5 runs on the 8-core M3 (idle to loaded): encoder layer 3.35s → 0.73–1.11s (3.0x–4.6x, ~3.8x reproducible idle), CNN forward 526ms → 97–169ms (3.1x–5.4x, ~4.4x reproducible idle). The encoder sum keeps ~90ms of deliberately serial small ops; its MatMul share alone is ~4.4x. Ordered before M17 — a device claim measured against one core has been withdrawn once already and will not be manufactured again |
| M17 | Inference reaches the device | planning | pending | f32 kernels behind the accel seam, landed only where measured to win against the M19 all-core baseline |
| M18 | Training (phase 2) | planning | pending | autodiff + SGD/Adam on the same tensors; first-step gradients match PyTorch under fixed SafeTensors-loaded weights |

Milestone order is the blocking sequence. OpenSpec has no dependency relationship between changes, so nothing else carries it.

## Current Blockers
None. One coverage gap is carried in `AGENTS.md` follow-ups rather than here, because it waits on hardware nobody has rather than on a decision: the device path is verified on Apple and Metal only.

## Next Verifiable Output
M17 can now be measured against the all-core M19 baseline. The device path still owes a per-platform bit-parity result before any `dl` f32 kernel is proposed.

## Next Ticket
`M17` — device inference, measured against the all-core M19 baseline. `dl` tensors are natively f32, so the "types the device holds exactly" row applies if per-platform bit parity holds; where it does not, the ticket must decide tolerance versus bit-exact behavior before any kernel lands.

## Decision Log
Deltas that still change what someone would do. The standing technical decisions they produced — the precision contract, the device rules, the measured thresholds — live in [ENG.md](ENG.md); the full history is in git.

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
- **One sample-weight design question remains open, deliberately.** Tree sample weights would push float weights into histogram accumulators the precision contract fixed as integers for associativity — an architecture decision against ENG.md, not a feature. The cross-validation channel turned out not to need a protocol break: scikit-learn routes sample_weight to fit and scores unweighted, so `Estimator` gained an optional `FitWeighted` and `CrossValidateWeighted` subsets weights with each fold's own indices. The metric protocol is untouched.
- **Two pure-CPU wins are measured and unclaimed**, and both are worth more than anything acceleration offered. `stats`' KNN auto-selection picks a ball tree up to 3.3x slower than parallel brute force on unstructured data (#190). CCL spends 4.8x–6.2x more time on recursion-depth bookkeeping than on evaluating (#191).
- **A skipped verification now fails when it was supposed to run.** `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` turns every missing-toolchain skip into a failure, and the `Reference Verification` workflow installs R, the Python scientific stack, scikit-learn and onnxruntime and runs with it set. Before this, `Clustering Parity` had been reporting green while running nothing — its gate imports `sklearn` and the workflow never installed it.
- **The refusal of unreadable numeric input is a breaking change** for anyone whose data contains blanks. What used to return a number now returns an error; the number it used to return was wrong. `ToF64Slice` still backs 54 call sites in `plot`, `gplot`, `quant` and the CLI — display paths where a zero is visible rather than laundered into a coefficient. A new numeric analysis must not join them.
- **Nothing calls `accel`.** It is reachable through `allpkgs` or a direct import only, so its dormancy costs users nothing today.
- **The race detector cannot reach the device on macOS**, upstream and pre-existing (gogpu/wgpu#280). Device tests skip under the `race` build tag.
