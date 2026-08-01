# Engineering Plan: Insyra
_2026-08-01 - eng-architect - insyra:gpgpu_

What holds across tickets. Anything scoped to one change lives on that change under `openspec/`; anything that is a status report lives in `delivery-status.md`. This file is the set of decisions a later agent must not re-make from scratch.

## Architecture

```text
                        insyra  (root)
                DataList / DataTable / CCL
                  values stored as []any
                            |
        +-------------------+-------------------+
        |                   |                   |
     stats/               ml/                accel/
  float64 throughout   sklearn-shaped     pure-Go WebGPU
  validated against R   estimator          (gogpu/wgpu)
        ^                protocol               |
        |                   |            internal/wgpu
        +---- wraps --------+            (leaf: imports
                                          nothing from accel)

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

A pipeline step is a **fit function, not a configured object**. scikit-learn refits per fold by cloning the unfitted estimator, and cloning needs `inspect.signature`; Go has no equivalent that is not struct-tag reflection, which this repo uses nowhere. A closure refits by being called again.

## Test matrix

| Test type | What it covers | Where |
| --- | --- | --- |
| Cross-language | every `stats` result field against R, and predictions against R and Python | `stats/testdata/*.R`, `crosslang_*_test.go` |
| Bit-parity | device result identical to its CPU reference on the running platform | `accel/exact_test.go` |
| Conformance | a third-party `ml.Model` obeys the protocol | `ml/mltest` |
| Order-independence | permuting input rows fits an identical tree | `ml/decision_tree_test.go` |
| Calibration | where a device wins, across 96 shapes | `accel/shapemap_test.go` |

Cross-language tests **skip** without `Rscript` (jsonlite, cluster, dbscan) and `python` (numpy, scipy, statsmodels, sklearn). A skipped verification looks exactly like a passing one — install both before trusting a green run.

## Precision contract

`insyra/ml` does not have a precision. It has one contract assigned by role, and every model obeys the same one. Each row is sourced from what mainstream libraries actually do, read at source level.

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
| Linking the device layer into a hello-world | +1.9 s cold build, +200 KB, +41 packages |

The floor is one host's number and belongs in a calibrated dispatcher eventually. A discrete GPU would move it **up**, not down — PCIe transfer raises the bar that unified memory does not charge for.

## Hidden assumptions

- **Bit-parity is platform-dependent, not kernel-dependent.** It holds because Metal and Go/arm64 contract multiply-add identically. That is a property of the toolchains, so it is measured where it runs and cannot be inferred for a platform nobody has run on — risk: the device path is verified on Apple and Metal only.
- **The pure-Go GPU stack is one maintainer.** gogpu/wgpu, naga and goffi are substantially one person's work, 159 stars, and CI exercises no GPU hardware — risk: the observable-fallback design is the mitigation, but no roadmap item should assume the dependency survives.
- **`stats`' R validation covers fitting, not prediction.** None of the 22 reference scripts called `predict()` until 2026-07-29 — risk: any claim that wrapping `stats` inherits its validation is true only for the fitting half.
- **`Predict` returns a `DataList` whatever the model is.** A measurement, a class label and a cluster id are the same type, so nothing downstream can tell them apart. Three defects came from this: a logistic model satisfying `Classifier` while `Predict` returned probabilities, cluster ids scored by RMSE, and a metric unable to say it needed probabilities. Each is a different party guessing at what the other meant — risk: no check can recover the missing information, because it is not in the data. A model must *declare* what it predicts (`Classifier`, `Clusterer`) and a metric what it consumes; inference is not available.
  The counter-example is `SimpleImputer`, which deliberately does not implement `InverseTransform` rather than implementing one that errors — its comment explains that a method present only to satisfy an assertion tells a caller the capability exists and then refuses at the call, so not having it is the only form of "no" an assertion can read. That is the same reasoning applied before the defect rather than after.
- **A test that passes for the wrong reason.** Caught three times in this phase: a benchmark arm named for a device that never ran, a numerical test that passed against the bound it was meant to exercise, and a conformance probe failing on an unrelated assertion — risk: a fix is not verified until the test covering it has been shown to fail without it.

## Migration plan

None outstanding. Two shape changes are recorded because their reasoning outlives them:

- `BackendAllocator` → `BackendExecutor` (breaking, unreleased). The old seam could carry neither an operation nor a result, so no real backend could implement it.
- Three device operations removed after measurement — a column sum at 0.7x, a distance matrix whose readback grew with the answer, and an `f32` nearest query no `float64` caller could use. Nothing removed had shipped.
