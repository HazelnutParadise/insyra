# Engineering Plan: Insyra
_2026-08-01 - eng-architect - insyra:gpgpu_

What holds across tickets. Anything scoped to one change lives on that change under `openspec/`; anything that is a status report lives in `delivery-status.md`. This file is the set of decisions a later agent must not re-make from scratch.

## Architecture

```text
                        insyra  (root)
                DataList / DataTable / CCL
                  values stored as []any
                            |
        +-----------+-----------+-----------+-----------+
        |           |           |           |
     stats/       ml/         nn/        accel/
  float64      sklearn-    ONNX inference  pure-Go WebGPU
  throughout,  shaped      + (phase 2)     (gogpu/wgpu)
  validated    estimator   autodiff;           |
  against R    protocol    f32 tensors,   internal/wgpu
        ^         |        dtype-carrying  (leaf)
        +- wraps -+           |
                              | kernels are plain functions on tensors;
                              | the graph interpreter is ONE caller —
                              | the future llm package is another

Dependency direction: accel -> insyra. The root package therefore
CANNOT import accel; a registration seam is the only route, and the
device layer would have to move to a package with no insyra dependency
before the root could reach it directly.
```

`ml` wraps `stats` rather than reimplementing it, so the R-validated numerics are inherited rather than duplicated. `accel` is reachable only through `allpkgs` or a direct import; nothing in `stats` or the root package calls it.

## Data flow

**Fitting a model.** `DataTable` → `ml.Fit*` records the fitted column names → delegates to the `stats` function → wraps the result in a `Model`. No arithmetic is reimplemented at any step.

**Scoring.** `DataTable` → columns matched **by name** against `Model.Features()` → the wrapped `stats` predict. Matching by name rather than position is deliberate: `stats`' own predict validates only the count, so wrong-order predictors silently produce wrong numbers.

**Acceleration.** `Dataset` → the device ranks in `f32` and returns a shortlist per row plus the best rejected distance → the host recomputes that shortlist in `float64` and decides → rows whose boundary falls inside the `f32` error bound are recomputed against every candidate. The answer never depends on the device being right, only on it being close, and not on there being a device at all.

## Test seams

| Seam | What gets faked | Used by |
| --- | --- | --- |
| `ml.Estimator.Fit` (a closure, not an object) | the whole model | cross-validation, pipelines, grid search |
| `accel.BackendExecutor` | the device | every accel test that does not need real hardware |
| `accel` builtin probe overrides | device discovery | isolation from whatever GPU the host has |
| `Rscript` / `python` subprocess | the reference implementation | `stats` cross-language validation |
| Single-op ONNX graph generator | the reference runtime, per operator | `nn` kernel parity: each kernel gets a generated one-op `.onnx`, run by both `nn` and `onnxruntime`, outputs compared — per-op reference verification without hand-computing |

A pipeline step is a **fit function, not a configured object**. scikit-learn refits per fold by cloning the unfitted estimator, and cloning needs `inspect.signature`; Go has no equivalent that is not struct-tag reflection, which this repo uses nowhere. A closure refits by being called again.

`nn`'s autodiff (phase 2) is a **tape of plain functions, not a graph transform**. Each differentiable op pairs its existing forward kernel with a VJP that is itself a plain function on tensors; training-side wrappers record (op, inputs, output) onto a tape, and backward walks the tape calling VJPs. The inference kernels stay untouched — the tape wrapper is one more caller, exactly like the graph interpreter and the future llm package. Differentiating the ONNX graph directly was rejected: it would couple gradients to the interpreter, and the interpreter is deliberately just one consumer of the kernels. Gradient truth is PyTorch under fixed SafeTensors-loaded weights, first step, f32 tolerance; softmax + cross-entropy differentiates as one fused VJP because the separated form loses the cancellation that makes it stable.

## Test matrix

| Test type | What it covers | Where |
| --- | --- | --- |
| Cross-language | every `stats` result field against R, and predictions against R and Python | `stats/testdata/*.R`, `crosslang_*_test.go` |
| Bit-parity | device result identical to its CPU reference on the running platform | `accel/exact_test.go` |
| Conformance | a third-party `ml.Model` obeys the protocol, including that importances and feature names agree in number | `ml/mltest` |
| Round-trip | an exported model loads and scores in `onnxruntime` | `ml/onnx_export_test.go` |
| Leakage | a cross-validated pipeline's steps see training rows only | `ml/pipeline_features_test.go` |
| Op parity | every `nn` kernel against `onnxruntime` on generated one-op graphs, plus whole-model round trips | `nn` (planned) |
| Order-independence | permuting input rows fits an identical tree | `ml/decision_tree_test.go` |
| Calibration | where a device wins, across 96 shapes | `accel/shapemap_test.go` |

Cross-language tests **skip** without `Rscript` (jsonlite, cluster, dbscan) and `python` (numpy, scipy, statsmodels, sklearn, onnxruntime). Set **`INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1`** to turn every such skip into a failure naming what was missing and what went unverified; the `Reference Verification` workflow installs all of them and runs with it set. Without it the default is unchanged, so `go test ./...` still passes on a machine with none of them.

Every gate routes through `internal/reftest` so none can opt out of that switch. The one deliberate exception is a `psych::factor.scores` upstream bug, where R is present and broken rather than absent — failing there would fail on something nobody can install their way out of.

This is not hypothetical tidiness. Until 2026-08-01 the `Clustering Parity` workflow installed `numpy scipy statsmodels` for a gate that imports `scipy, numpy, statsmodels, sklearn`; measured in a clean environment, that leaves `sklearn` missing, so the workflow dedicated to running the parity suite reported green while running none of it. The ONNX round trip had never executed anywhere, and hid two defects that made every exported model invalid.

## Precision contract

`insyra/ml` does not have a precision. It has one contract assigned by role, and every model obeys the same one. Each row is sourced from what mainstream libraries actually do, read at source level.

Values that cannot be read as a finite number are refused rather than converted, and each family states which of three treatments it applies: refusal (regression, correlation, clustering, PCA, KNN), listwise deletion (factor analysis), or a learned per-node direction for a missing feature with refusal of a missing target (decision trees).

| Role | Type | Why |
| --- | --- | --- |
| Bulk feature values | `float32`, or quantised integers where they only feed comparisons | scikit-learn's tree module copies a `float64` X **down** to `float32` in the same `fit` that copies a `float32` y **up** to `float64` (`_tree.pxd`: `DTYPE_t # Type of X`, `DOUBLE_t # Type of y`) |
| Classification labels | integer or bool | exact and free |
| Regression targets entering a residual | `float64` | error propagates |
| Terms feeding a long reduction | widened at read time, in bounded chunks | scikit-learn's `_euclidean_distances_upcast` |
| **Reduction accumulator** | **fixed-point integers** | see below |
| A comparison key deciding a selection | widest precision available, on the host, always | scikit-learn's `_argkmin` heap is `float64` in both its `f32` and `f64` specialisations; no dissent found anywhere |
| Stored model parameters | `float64` | keeps `ml` aligned with a `float64` `stats`, and makes ONNX export lossless |
| Reported values | `float64` | the caller compares and reports them |

**The accumulator is the row nobody would guess.** Bit-exactness under a dispatcher that may move the CPU/device split point requires an accumulator that is **associative**, not merely deterministic — every regrouping of a floating-point sum is a different answer. Fixed point is associative. XGBoost removed its single-precision histogram option in 1.7 as "dangerous to use" and replaced it with `GradientPairInt64`; it arrived there because GPU atomics reorder and we arrive there because a dispatcher partitions, and it is the same fix.

The test that makes this real: permuting 600 rows spanning magnitudes 10⁻³ to 10³ must fit a bit-identical tree. Floating-point accumulation cannot pass it.

## When a device may be used

Decided by the **shape of the result**, not by how hot the operation is.

| Result shape | Default | Why |
| --- | --- | --- |
| A selection — which row, which index, what order | on | The `f32` ranking is a proposal; the host recomputes the shortlist in `float64` and picks, so the answer is exact |
| Values in a type the device holds exactly | on | bit-identical outright |
| New `float64` values | opt-in only | nothing verifies them more cheaply than recomputing, and WebGPU has no `f64` |

Two rules follow and are not negotiable:

- **A missing or broken device is a performance event, never a correctness one.** The verification half is a complete implementation, so no device means every row takes the path already written for untrustworthy rows.
- **Do not write a kernel for work measurement says the device loses.** Memory-bound work stays on the CPU permanently.

## Measured thresholds

All on an Apple M3, against a host using **all eight cores**. Every figure recorded before 2026-07-29 compared a GPU against one core and is overstated by up to the core count.

| Measurement | Value |
| --- | --- |
| Device beats 8 cores by, at best | 3.63x, over 96 shapes |
| Profitability floor | ~2048 distance evaluations per row (`dims × candidates`), `accel/exact.go` |
| Raw compute ceiling | GPU 30.4 G evals/s vs 6.6 G/s across 8 cores |
| Algorithmic pruning beats a device by | 1.4x–2.7x on structured data |
| KNN device floor (true direction) | ≥ ~2048 test rows AND ≥ 2048 work per row; device time is flat in test rows until saturation, so below the floor it loses outright (469ms vs CPU 324ms at 1k rows) |
| Linking the device layer into a hello-world | +1.9 s cold build, +200 KB, +41 packages |

The floor is one host's number and belongs in a calibrated dispatcher eventually. A discrete GPU would move it **up**, not down — PCIe transfer raises the bar that unified memory does not charge for.

## Hidden assumptions

- **Bit-parity is platform-dependent, not kernel-dependent.** It holds because Metal and Go/arm64 contract multiply-add identically. That is a property of the toolchains, so it is measured where it runs and cannot be inferred for a platform nobody has run on — risk: the device path is verified on Apple and Metal only.
- **The pure-Go GPU stack is one maintainer.** gogpu/wgpu, naga and goffi are substantially one person's work, 159 stars, and CI exercises no GPU hardware — risk: the observable-fallback design is the mitigation, but no roadmap item should assume the dependency survives.
- **A conversion with no failure channel is a fabrication.** `DataList.ToF64Slice` routes through `insyra.ToFloat64`, which yields `0` for anything it cannot parse, and returns a slice of the right length regardless — so every caller believed it held the caller's data. Regression and correlation read through it until 2026-08-01; one blank among six observations moved a Pearson coefficient from 0.9992 to 0.9879 with no error. `stats` now converts through one validating helper — risk: 54 call sites elsewhere in the library still use `ToF64Slice`, including `plot`, `gplot`, `quant` and the CLI. Those are display and reporting paths where a zero is visible rather than laundered into a coefficient, which is why they were left; a new numeric analysis must not join them.
- **`stats`' R validation covers fitting, not prediction.** None of the 22 reference scripts called `predict()` until 2026-07-29 — risk: any claim that wrapping `stats` inherits its validation is true only for the fitting half.
- **`Predict` returns a `DataList` whatever the model is.** A measurement, a class label and a cluster id are the same type, so nothing downstream can tell them apart. Four defects came from this — the fourth being a metric's *score*, whose direction is equally invisible in a `float64`: a logistic model satisfying `Classifier` while `Predict` returned probabilities, cluster ids scored by RMSE, and a metric unable to say it needed probabilities. Each is a different party guessing at what the other meant — risk: no check can recover the missing information, because it is not in the data. A model must *declare* what it predicts (`Classifier`, `Clusterer`) and a metric what it consumes; inference is not available.
  The same shape appeared twice more and was fixed the same way. A metric's score is a `float64` whichever way it improves, so `Metric` now declares a `Direction`; without it a caller ranking two means picked the worse model half the time. A pipeline's feature names and its estimator's importances are both `[]string`/`[]float64` with nothing tying them together, so a fitted pipeline now reports `TransformedFeatureNames`.
  The counter-example is `SimpleImputer`, which deliberately does not implement `InverseTransform` rather than implementing one that errors — its comment explains that a method present only to satisfy an assertion tells a caller the capability exists and then refuses at the call, so not having it is the only form of "no" an assertion can read. That is the same reasoning applied before the defect rather than after.
- **A `.onnx` file is untrusted input.** The `nn` decoder must error on malformed bytes, never panic, and must list every unsupported operator by name at load time rather than failing mid-run — risk: a graph interpreter that panics on attacker-shaped input is a denial-of-service primitive in any service that loads user models.
- **`nn`'s f32 inference is device-eligible by the existing contract, not a new rule.** ONNX models carry f32 weights, and f32 is "a value type the device holds exactly" in the result-shape table — bit-identical outright. GPU inference still lands only behind a measurement, like everything else.
- **`nn` v1 constraints for the decided GGUF/LLM future**: tensors carry a dtype (f32 is the only implemented one; the type system must not weld the door shut on quantised types), and kernels are plain functions the graph interpreter merely calls — the future `llm` package hard-codes transformer architectures against the same kernels instead of interpreting graphs.
- **A test that passes for the wrong reason, or never runs at all.** Caught five times in this phase: a benchmark arm named for a device that never ran, a numerical test that passed against the bound it was meant to exercise, a conformance probe failing on an unrelated assertion, cross-language validation that skipped wherever `Rscript` was absent, and an ONNX round-trip that had skipped on every machine it ever ran on — hiding two defects that made every exported model invalid — risk: a fix is not verified until its test has been shown to fail without it, and a suite is not green until its skips have been read.

## Migration plan

None outstanding. Two shape changes are recorded because their reasoning outlives them:

- `BackendAllocator` → `BackendExecutor` (breaking, unreleased). The old seam could carry neither an operation nor a result, so no real backend could implement it.
- Three device operations removed after measurement — a column sum at 0.7x, a distance matrix whose readback grew with the answer, and an `f32` nearest query no `float64` caller could use. Nothing removed had shipped.
