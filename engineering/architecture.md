# Architecture: Insyra

How the parts of the module fit together, how they are tested, and the contracts every change keeps. Why each hard-to-reverse choice was made is in its record under [decisions/](decisions/); this file links to them instead of repeating them.

## Components and boundaries

```text
                        insyra  (root)
                DataList / DataTable / CCL
                  values stored as []any
                            |
        +-----------+-----------+-----------+-----------+
        |           |           |           |
     stats/       ml/         nn/        accel/
  float64      sklearn-    ONNX inference  pure-Go WebGPU
  throughout,  shaped      + autodiff;     (gogpu/wgpu)
  validated    estimator   f32 tensors,        |
  against R    protocol    dtype-carrying, internal/wgpu
        ^         |        device-resident (leaf)
        +- wraps -+           |
                              | kernels are plain functions on tensors;
                              | the graph interpreter is ONE caller —
                              | the future llm package is another

Dependency direction: accel -> insyra, and nn -> accel.
The root package cannot import accel; it reaches it only
through a registration seam.
```

- `ml` wraps `stats` rather than reimplementing it ([0006](decisions/0006-ml-wraps-stats.md)), and follows scikit-learn's estimator model ([0005](decisions/0005-ml-imitates-scikit-learn.md)).
- Nothing in `stats` or the root package calls `accel` ([0004](decisions/0004-root-package-reaches-accel-through-a-seam.md)). `nn` imports it directly for device MatMul and resident tensors ([0009](decisions/0009-nn-device-matmul-is-default-on.md), [0019](decisions/0019-residency-belongs-to-the-tensor.md)); other callers reach it through `allpkgs` or a direct import.
- The command language runs the same way from the shell and from Go, so it sits below both: `internal/dsl/` holds the commands, the registry, environments and the session; `cli/` (the Cobra shell, the REPL, readline) and `engine/dsl` (the Go entry point) both import it. Dependency direction: `cli -> internal/dsl` and `engine/dsl -> internal/dsl`, never `engine -> cli`. `TestEngineDSLDoesNotDependOnTheCLI` holds the line with `go list -deps`. The public names a Go program spells (`cli/env.Manager`, `cli/commands.ExecContext`) are aliases in thin `cli/` packages that pull in neither Cobra nor readline, apart from `cli/commands`, which also builds the shell.
- `0.4` and `dev` are two release lines with different rules for what may change ([0012](decisions/0012-two-release-lines.md)).

## Data flow

**Fitting a model.** `DataTable` → `ml.Fit*` records the fitted column names → delegates to the `stats` function → wraps the result in a `Model`. No arithmetic is reimplemented at any step.

**Scoring.** `DataTable` → columns matched **by name** against `Model.Features()` → the wrapped `stats` predict. `stats`' own predict validates only the count, so matching by position would let wrong-order predictors produce wrong numbers silently.

**Acceleration of a selection.** `Dataset` → the device ranks in `f32` and returns a shortlist per row plus the best rejected distance → the host recomputes that shortlist in `float64` and decides → rows whose boundary falls inside the `f32` error bound are recomputed against every candidate. The answer never depends on the device being right, only on it being close, and not on there being a device at all ([0001](decisions/0001-device-use-follows-result-shape.md)).

**Training on a device.** A tensor carries the device it lives on. `To` and `ToHost` move it explicitly, operations with device kernels keep their results there, and nothing reads back by itself ([0019](decisions/0019-residency-belongs-to-the-tensor.md)).

## Test seams

| Seam | What gets faked | Used by |
| --- | --- | --- |
| `ml.Estimator.Fit` (a closure, not an object) | the whole model | cross-validation, pipelines, grid search |
| the internal `backendExecutor`, reached through `internal/acceltest` (`accel.BackendExecutor` is a Deprecated alias) | the device | every accel test that does not need real hardware |
| `accel` builtin probe overrides | device discovery | isolation from whatever GPU the host has |
| `Rscript` / `python` subprocess | the reference implementation | `stats` cross-language validation |
| Single-op ONNX graph generator | the reference runtime, per operator | `nn` kernel parity: each kernel gets a generated one-op `.onnx`, run by both `nn` and `onnxruntime`, outputs compared |

A pipeline step is a **fit function, not a configured object**: a closure refits by being called again ([0005](decisions/0005-ml-imitates-scikit-learn.md)).

`nn`'s layer surface is **one Forward, no mode flag**. A `Layer` has `Build(tape)` (parameters materialise on attach, seeded, in layer order), `Forward(tape, x)`, and `Parameters()`. `Sequential.Predict` runs Forward on a throwaway tape and skips layers marked `TrainingOnly`, so dropout cannot leak into inference because no mode bit exists to forget. Dimensions are explicit, as in PyTorch, so shape errors fire at construction and name the layer. Parameter names follow torch's `nn.Sequential` convention (`0.weight`, `3.bias`, …), and `LoadWeights` transposes torch's `[out,in]` Linear layout at the boundary, so a PyTorch-trained Sequential loads directly. The sugar-changes-nothing proof is exact: the Sequential MNIST run must reproduce the hand-written tape run's loss curve to the last digit under the same seed.

`nn`'s autodiff is a **tape of plain functions, not a graph transform**. Each differentiable operation pairs its forward kernel with a VJP that is itself a plain function on tensors; training-side wrappers record (op, inputs, output) onto a tape, and backward walks the tape calling VJPs. The inference kernels stay untouched, and the graph interpreter stays one consumer of them among others. Differentiating the ONNX graph directly was rejected: it would couple gradients to the interpreter. Gradient truth is PyTorch under fixed SafeTensors-loaded weights, first step, f32 tolerance. Softmax and cross-entropy differentiate as one fused VJP, because the separated form loses the cancellation that makes it stable.

### Test matrix

| Test type | What it covers | Where |
| --- | --- | --- |
| Cross-language | every `stats` result field against R, and predictions against R and Python | `stats/testdata/*.R`, `crosslang_*_test.go` |
| Bit-parity | device result identical to its CPU reference on the running platform | `accel/exact_test.go` |
| Conformance | a third-party `ml.Model` obeys the protocol, including that importances and feature names agree in number | `ml/mltest` |
| Round-trip | an exported model loads and scores in `onnxruntime` | `ml/onnx_export_test.go` |
| Leakage | a cross-validated pipeline's steps see training rows only | `ml/pipeline_features_test.go` |
| Op parity | every `nn` kernel against `onnxruntime` on generated one-op graphs, plus whole-model round trips | `nn` |
| Order-independence | permuting input rows fits an identical tree | `ml/decision_tree_test.go` |
| Calibration | where a device wins, across 96 shapes | `accel/shapemap_test.go` |

Cross-language tests **skip** without `Rscript` (jsonlite, cluster, dbscan) and `python` (numpy, scipy, statsmodels, sklearn, onnxruntime, and cvxpy for the `quant` portfolio comparison). **`INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1`** turns every such skip into a failure naming what was missing and what went unverified. Every gate routes through `internal/reftest`, and the `Reference Verification` workflow installs every toolchain and runs with the variable set ([0008](decisions/0008-a-skipped-verification-can-fail.md)). `clustering-parity`, `knn-parity` and `reference-verification` set up Python, R and their packages through one composite action, `.github/actions/setup-reference-toolchains`.

`nn`'s MNIST convergence and real-model parity need **data** the repository does not carry, 288 MB of published checkpoints and 11 MB of MNIST. The `Neural Network Data Gates` workflow fetches them from the sources pinned by sha256 in `.github/nn-data-manifest.txt`, checks every file before testing, and fails when one of the five tests skipped. It runs when `nn/` changes.

The **GPU** gates (`INSYRA_ACCEL_GPU_TESTS=1` over `./accel/...` and `./nn/`) have no hosted runner and stay manual. A self-hosted runner on a maintainer's machine is out: the repository is public, so pull requests from forks would run on it. Run them on a machine with a device before changing `accel` or the device path, from a shell that sees the real adapter (under `-race` the CPU references take about 50 minutes on the M3, so pass `-timeout 150m`), and record what they measured in the change that needed them, or in a decision record when the measurement decides a choice.

## Precision contract

`insyra/ml` has one precision contract, assigned by role, and every model obeys it ([0020](decisions/0020-precision-is-assigned-by-role.md)).

Values that cannot be read as a finite number are refused rather than converted, and each family states which of three treatments it applies: refusal (regression, correlation, clustering, PCA, KNN), listwise deletion (factor analysis), or a learned per-node direction for a missing feature with refusal of a missing target (decision trees) ([0007](decisions/0007-numeric-input-is-refused-not-zeroed.md)).

| Role | Type | Source |
| --- | --- | --- |
| Bulk feature values | `float32`, or quantised integers where they only feed comparisons | scikit-learn's tree module copies a `float64` X **down** to `float32` in the same `fit` that copies a `float32` y **up** to `float64` (`_tree.pxd`: `DTYPE_t # Type of X`, `DOUBLE_t # Type of y`) |
| Classification labels | integer or bool | exact and free |
| Regression targets entering a residual | `float64` | error propagates |
| Terms feeding a long reduction | widened at read time, in bounded chunks | scikit-learn's `_euclidean_distances_upcast` |
| **Reduction accumulator** | **fixed-point integers** | associative under any split ([0020](decisions/0020-precision-is-assigned-by-role.md)) |
| A comparison key deciding a selection | widest precision available, on the host, always | scikit-learn's `_argkmin` heap is `float64` in both its `f32` and `f64` specialisations |
| Stored model parameters | `float64` | keeps `ml` aligned with a `float64` `stats`, and makes ONNX export lossless |
| Reported values | `float64` | the caller compares and reports them |

Permuting 600 rows spanning magnitudes 10⁻³ to 10³ must fit a bit-identical tree.

## Exact decimals

An exact decimal anywhere in Insyra is a `github.com/TimLai666/go-decimal` `decimal.Decimal` ([0021](decisions/0021-one-decimal-type.md)).

Core does not import it. The numeric read path, `IsNumeric` and sorting recognise a decimal by shape — a value that reports its own text and its scale, where that text parses as a number — so `internal/utils`, which every cell passes through, stays independent of any one decimal package. Two decimals compare through `decimal.Cmp`, exactly; a decimal against any other number compares as `float64`. A decimal is not a map key (its `big.Int` holds a slice) and goes through `ToMapKey` like any other uncomparable value ([0027](decisions/0027-a-cell-is-identified-by-type-and-content.md)).

## When a device may be used

Decided by the **shape of the result**, not by how hot the operation is ([0001](decisions/0001-device-use-follows-result-shape.md)). The table and its two rules are in `AGENTS.md` under "How the CPU and the GPU Divide the Work". A kernel exists only where a measurement against every core shows the device winning, except #379's resident path, which the owner ruled for so a resident computation need not leave the device ([0014](decisions/0014-device-results-are-correctly-rounded.md), [0002](decisions/0002-acceleration-measured-against-all-cores.md), [0003](decisions/0003-a-kernel-only-where-the-device-wins.md)).

**What a device result is.** A float32 result that may come from a device is the correctly rounded value of the exact operation. `EdgeSum` is the exact sum of its products rounded once, and `tanh(x)` is the true value rounded once. A result that is exactly zero is `+0`, as when accumulating into a zeroed output ([0014](decisions/0014-device-results-are-correctly-rounded.md)).

**How a device kernel computes it.** In integers. A kernel holds float32 values as bit patterns (`array<u32>`, never materialised as `f32`), forms exact products and sums in integer limbs, and rounds once to nearest-even in integer code. An exact sum does not depend on the order it is added in, so a kernel may split one sum across threads freely.

**The device register.** [`accel/internal/wgpu/exact_sum.go`](/accel/internal/wgpu/exact_sum.go) counts in units of 2^-298, like `nn`'s CPU accumulator. It holds 27 digits of 24 bits in `u32` and defers carries for up to 64 additions. A product always spans three digits, and the deferral bound keeps every digit's true value below 2^31, so nothing ever wraps. Every operation is unsigned, so no result depends on how a shader compiler treats signed overflow or the right shift of a negative number. A kernel that needs more exact arithmetic extends this library instead of writing its own ([0016](decisions/0016-device-kernels-share-one-exact-register.md)). A caller's own kernel (`accel.NewKernel`) is compiled without this library, Insyra cannot check what it computes, and a tape that runs one uses `PrecisionFloat32`. The caller-kernel bindings and the shared submit lock are the same ones Insyra's own kernels use.

**How that is verified.** By construction, and exhaustively wherever the input space allows: a unary float32 function is compared on all 2^32 inputs against a high-precision oracle. Sums are compared against `math/big` on random inputs and on cancellation, subnormals, overflow, NaN, infinities and signed zeros. A CPU fast path behind correct rounding uses only IEEE 754 `+ − × ÷`, with an explicit conversion around every product that feeds an addition or subtraction, and no `math` function. A CI test hashes the fast path's bits over its whole input range. `nn.Tanh` is the pattern ([0015](decisions/0015-cpu-fast-paths-use-only-basic-ieee-operations.md)).

## Measured thresholds

All on an Apple M3, against a host using **all eight cores** ([0002](decisions/0002-acceleration-measured-against-all-cores.md)). Every figure recorded before 2026-07-29 compared a GPU against one core and is overstated by up to the core count.

| Measurement | Value |
| --- | --- |
| Device beats 8 cores by, at best | 3.63x, over 96 shapes |
| Exact-nearest profitability floor | ~2048 distance evaluations per row (`dims × candidates`), `accel/exact.go` |
| Raw compute ceiling | GPU 30.4 G evals/s vs 6.6 G/s across 8 cores |
| Algorithmic pruning beats a device by | 1.4x–2.7x on structured data |
| KNN device floor (in the call site's direction) | ≥ ~2048 test rows AND ≥ 2048 work per row; below it the device loses outright (469 ms vs CPU 324 ms at 1k rows) |
| Exact-nearest rows per submission | 16,000; larger inputs run in sequential chunks ([0011](decisions/0011-multi-device-ships-before-multi-gpu-measurement.md)) |
| Multi-device `auto` floor per assignment | 32k rows at 32 dimensions, 8k rows at 128 dimensions ([0011](decisions/0011-multi-device-ships-before-multi-gpu-measurement.md)) |
| `nn` device MatMul floor | 16Mi multiply-accumulates; batched products stay on the CPU ([0009](decisions/0009-nn-device-matmul-is-default-on.md)) |
| Linking the device layer into a hello-world | +1.9 s cold build, +200 KB, +41 packages |

These are one host's numbers and belong in a calibrated dispatcher eventually. A discrete GPU would move the floors **up**, not down: PCIe transfer raises the bar that unified memory does not charge for.

## Standing assumptions

- **Device MatMul's bit-parity is platform-dependent, not kernel-dependent.** It holds because Metal and Go/arm64 contract multiply-add identically, which WGSL does not promise. MatMul predates the integer rule above — risk: the device path is verified on Apple and Metal only, and a conforming implementation that reassociates would break it.
- **The pure-Go GPU stack is one maintainer.** gogpu/wgpu, naga and goffi are substantially one person's work, and CI exercises no GPU hardware. Its translation is not always right: the device parity tests caught two naga miscompilations (gogpu/naga#95, #133) — risk: the observable-fallback design is the mitigation, device results are compared bit for bit rather than trusted, and no roadmap item should assume the dependency survives.
- **`ToF64Slice` fabricates zeros.** `stats` no longer reads through it ([0007](decisions/0007-numeric-input-is-refused-not-zeroed.md)) — risk: display and reporting paths in `plot`, `gplot` and the CLI still use it; a new numeric analysis must not join them.
- **`stats`' R validation covers fitting, not prediction.** None of the 22 reference scripts called `predict()` until 2026-07-29 — risk: any claim that wrapping `stats` inherits its validation is true only for what the scripts call.
- **`Predict` returns a `DataList` whatever the model is.** A measurement, a class label and a cluster id are the same type, so nothing downstream can tell them apart — risk: no check can recover the missing information, because it is not in the data. A model *declares* what it predicts (`Classifier`, `Clusterer`), a metric declares what it consumes and its `Direction`, and a fitted pipeline reports `TransformedFeatureNames`. `SimpleImputer` deliberately does not implement `InverseTransform`, because a method present only to refuse tells an interface assertion the capability exists.
- **A `.onnx` file is untrusted input.** The `nn` decoder errors on malformed bytes, never panics, and lists every unsupported operator by name at load time — risk: a graph interpreter that panics on attacker-shaped input is a denial-of-service primitive in any service that loads user models.
- **`nn`'s f32 inference is device-eligible by the existing contract.** ONNX models carry f32 weights, and f32 is a value type the device holds exactly — risk: none beyond the measurement every kernel needs first.
- **`nn` v1 keeps the door open for GGUF/LLM.** Tensors carry a dtype, and kernels are plain functions the graph interpreter merely calls, so a future `llm` package can hard-code transformer architectures against the same kernels — risk: a change that welds the type system to f32 or hides kernels inside the interpreter closes that track.
- **A test can pass for the wrong reason, or never run.** Caught five times: a benchmark arm named for a device that never ran, a numerical test that passed against the bound it was meant to exercise, a conformance probe failing on an unrelated assertion, cross-language validation that skipped wherever `Rscript` was absent, and an ONNX round trip that had skipped on every machine — risk: a fix is not verified until its test has been shown to fail without it, and a suite is not green until its skips have been read.

## Decisions

- [0001 — Whether a device may be used follows the shape of the result](decisions/0001-device-use-follows-result-shape.md)
- [0002 — Acceleration is measured against every core, in the call site's direction](decisions/0002-acceleration-measured-against-all-cores.md)
- [0003 — A device kernel exists only where measurement shows the device wins](decisions/0003-a-kernel-only-where-the-device-wins.md)
- [0004 — The root package reaches acceleration through a seam](decisions/0004-root-package-reaches-accel-through-a-seam.md)
- [0005 — `insyra/ml` imitates scikit-learn, with fitting returning a separate value](decisions/0005-ml-imitates-scikit-learn.md)
- [0006 — `ml` wraps `stats` and reimplements only decision trees](decisions/0006-ml-wraps-stats.md)
- [0007 — Numeric input that cannot be read is refused, not turned into zero](decisions/0007-numeric-input-is-refused-not-zeroed.md)
- [0008 — A verification that skips can be made to fail](decisions/0008-a-skipped-verification-can-fail.md)
- [0009 — `nn`'s large MatMul runs on the device by default](decisions/0009-nn-device-matmul-is-default-on.md)
- [0010 — The performance gap to ONNX Runtime is stated, not chased](decisions/0010-onnx-runtime-gap-is-positioned.md)
- [0011 — Multi-device execution ships before a multi-GPU host has measured it](decisions/0011-multi-device-ships-before-multi-gpu-measurement.md)
- [0012 — Breaking changes go on `0.4`; `dev` stays on the 0.3.x line](decisions/0012-two-release-lines.md)
- [0013 — Issue #379 is a general sparse edge sum, not a recurrent-step operation](decisions/0013-sparse-edge-sum-is-a-general-operation.md)
- [0014 — A float32 result that may come from a device is the correctly rounded exact value](decisions/0014-device-results-are-correctly-rounded.md)
- [0015 — A CPU fast path behind correct rounding uses only IEEE 754's basic operations](decisions/0015-cpu-fast-paths-use-only-basic-ieee-operations.md)
- [0016 — Device kernels share one exact register and work on resident inputs](decisions/0016-device-kernels-share-one-exact-register.md)
- [0018 — Go 1.26 is the minimum](decisions/0018-go-1-26-is-the-minimum.md)
- [0019 — Residency belongs to the tensor, and the tape has a rounded mode](decisions/0019-residency-belongs-to-the-tensor.md)
- [0020 — `insyra/ml` assigns precision by role, with a fixed-point accumulator](decisions/0020-precision-is-assigned-by-role.md)
- [0021 — One exact decimal type across the module](decisions/0021-one-decimal-type.md)
- [0022 — A bare column reference is an index, never a column name](decisions/0022-a-bare-selector-is-never-a-column-name.md)
- [0023 — Streaming APIs are range-over-func iterators](decisions/0023-streaming-apis-are-iterators.md)
- [0024 — `lp` defaults to a solver written in Go](decisions/0024-lp-defaults-to-a-pure-go-solver.md)
- [0025 — Factor rotation follows GPArotation's engine and psych's defaults](decisions/0025-factor-rotation-follows-the-r-references.md)
- [0026 — `insyra/parallel` stays, as groups that are run and awaited](decisions/0026-parallel-stays.md)
- [0027 — A cell is identified by its type and content](decisions/0027-a-cell-is-identified-by-type-and-content.md)
- [0028 — An exported function is declared with `func` and has one name](decisions/0028-an-exported-function-is-a-func.md)
