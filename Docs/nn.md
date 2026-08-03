# `nn` Package

The `nn` package loads a focused subset of ONNX models and runs them with pure
Go. It is intended for small inference graphs such as exported multilayer
perceptrons, transformer encoder blocks, and MNIST-class CNN classifiers. The
graph is validated when it is loaded, and runtime inputs and outputs are bound
by name.

## Installation

```bash
go get github.com/HazelnutParadise/insyra/nn
```

## Tensors

`Tensor` stores row-major N-dimensional data and carries an explicit `DType`.
`float32` is the arithmetic type used by the inference kernels. `int64`,
`string`, and `bool` are also stored for categorical preprocessing and
classifier outputs.

```go
input, err := nn.NewTensor(
    []int{2, 3},
    []float32{1, 2, 3, 4, 5, 6},
)
if err != nil {
    log.Fatal(err)
}

fmt.Println(input.DType())   // float32
fmt.Println(input.Shape())   // [2 3]
fmt.Println(input.Strides()) // [3 1]
```

`Shape`, `Strides`, and the typed data accessors return copies. Elementwise
float32 arithmetic and batched `MatMul` use NumPy-style broadcasting. `MatMul`
keeps a tight two-dimensional path and broadcasts all leading batch dimensions.
Large `MatMul` and `Conv` workloads distribute independent output work across
all CPU cores. Each output keeps the serial accumulation order, so parallel
results are bit-identical to serial results; small workloads stay serial.
`NewInt64Tensor`, `NewStringTensor`, and `NewBoolTensor` build the control and
label tensors used by ONNX graphs.

## Device MatMul

Large 2-D float32 MatMuls use the device by default when the measured 16Mi
multiply-accumulate floor is reached. No blank import is required. Batched and
below-floor products stay on the CPU because measurement found that separate
device dispatches do not pay for their transfer cost.

Device errors and missing hardware fall back to the exact CPU result. The
fallback is observable through `accel.Default().Report()`.

Acceleration has two layers. The programmatic primary switch is
`insyra.Config.SetAcceleration(false)`, which makes eligible `nn` MatMuls and
the accelerator bridge stay on their exact CPU paths. It defaults to enabled
and can be re-enabled with `insyra.Config.SetAcceleration(true)`.

`INSYRA_ACCEL_DISABLE_WGPU=1` is the operations override for the builtin
WebGPU backend. It wins over Config, so the backend stays off even when Config
acceleration is enabled. Both layers must allow the device before a call uses
it. `nn.RegisterDeviceMatMul(nil)` remains a low-level nn-only escape hatch
that clears the MatMul hook.

The device path is not wired in `-race` builds because the upstream Metal
binding currently trips `checkptr`; those builds remain on the CPU path.

## Loading and running an ONNX model

```go
file, err := os.Open("model.onnx")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

model, err := nn.LoadONNX(file)
if err != nil {
    log.Fatal(err)
}

outputs, err := model.Run(map[string]*nn.Tensor{"input": input})
if err != nil {
    log.Fatal(err)
}
result := outputs["output"]
```

`LoadONNX` accepts an `io.Reader`, materialises initializers, reads the default
ONNX opset, and rejects malformed input without panicking. Unsupported
operators are reported together at load time. `Model.Inputs()` and
`Model.Outputs()` expose the declared names, dtypes, and shapes. A `-1` shape
dimension is dynamic and accepts any non-negative runtime dimension.

`Model.Run` requires every declared input, rejects extra inputs, and validates
each input's name, dtype, rank, and fixed dimensions. It returns named output
tensors and reports the node name when a graph operation fails.

ONNX `FLOAT16` and `BFLOAT16` initializers are widened value-exactly to
`float32` when loaded. A `Cast` whose target is `FLOAT16` or `BFLOAT16` rounds
the f32 value to that storage format and widens it back immediately. The graph
interpreter computes in f32; it does not claim half-precision arithmetic or
half-precision outputs.

## Loading SafeTensors weights

`LoadSafeTensors` reads a complete SafeTensors file from an `io.Reader` and
returns a `map[string]*nn.Tensor` keyed by the file's tensor names:

```go
file, err := os.Open("weights.safetensors")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

weights, err := nn.LoadSafeTensors(file)
if err != nil {
    log.Fatal(err)
}
kernel := weights["layer.weight"]
```

The loader validates the 8-byte header length, JSON entries, shape element
counts, byte offsets, non-overlap, and complete contiguous coverage of the data
region before materialising any tensor. The optional `__metadata__` entry is
accepted as a string-to-string object and ignored. Malformed input returns an
error naming the defect and tensor rather than panicking.

The supported loading contract is:

| SafeTensors dtype | Tensor dtype | Conversion |
| --- | --- | --- |
| `F32` | `float32` | native |
| `F16` | `float32` | exact IEEE binary16 widening |
| `BF16` | `float32` | exact top-16-bit binary32 widening |
| `I64` | `int64` | native |
| `BOOL` | `bool` | native |

`F64`, quantized dtypes, and other unsupported dtypes are refused together in
one error, with every tensor name and dtype listed. There is no silent
narrowing, and the loader does not mmap or load tensors lazily.

## Training and autodiff

`nn` also provides a small reverse-mode tape for MLP, attention-family, and
CNN training. The tape calls
the existing tensor kernels during the forward pass and stores one VJP record
per call; it does not transform an ONNX graph.

```go
tape := nn.NewTape()
w1, err := tape.Param(weights["layer1.weight"])
if err != nil { log.Fatal(err) }
b1, err := tape.Param(weights["layer1.bias"])
if err != nil { log.Fatal(err) }
w2, err := tape.Param(weights["layer2.weight"])
if err != nil { log.Fatal(err) }
b2, err := tape.Param(weights["layer2.bias"])
if err != nil { log.Fatal(err) }

hidden, err := tape.MatMul(input, w1.Value())
if err != nil { log.Fatal(err) }
hidden, err = tape.Add(hidden, b1.Value())
if err != nil { log.Fatal(err) }
hidden, err = tape.Relu(hidden)
if err != nil { log.Fatal(err) }
logits, err := tape.MatMul(hidden, w2.Value())
if err != nil { log.Fatal(err) }
logits, err = tape.Add(logits, b2.Value())
if err != nil { log.Fatal(err) }

loss, err := tape.SoftmaxCrossEntropy(logits, labels)
if err != nil { log.Fatal(err) }
if err := tape.Backward(loss); err != nil { log.Fatal(err) }
if err := tape.SGD(0.01); err != nil { log.Fatal(err) }
gradient := w1.Grad()
```

The differentiable wrappers are `MatMul`, `Add`, `Relu`, `Sigmoid`, `Tanh`,
`Gemm`, `Mul`, `Div`, `Softmax`, `LayerNormalization`, `Gelu`, `Erf`, `Sqrt`,
`Pow`, `ReduceMean`, and the shape wrappers `Transpose`, `Reshape`, `Flatten`,
`Squeeze`, `Unsqueeze`, `Slice`, `Concat`, and `Split`, together with the CNN
wrappers `Conv`, `MaxPool`, `AveragePool`, `GlobalAveragePool`, and inference-
mode `BatchNormalization`, plus training-mode `Dropout`. `Gemm` accepts alpha,
beta, and transpose attributes. `SoftmaxCrossEntropy`
takes logits
with shape `[N, C]` and int64 labels with shape `[N]`, and returns one mean-loss
scalar. Its backward pass emits the fused stable `(softmax - onehot) / N`
gradient; a separated softmax plus log-loss path is not provided. `SGD` applies
one in-place `w -= learningRate * gradient` step to every tracked parameter.
Gradients are float32 and are available through `Parameter.Grad()` or
`Tape.Grad(parameter.Value())` after `Backward`; an unconnected tracked
parameter receives a zero tensor. Exact-form GELU is differentiable; the tanh
approximation is refused by the tape until its VJP is covered.

Adam keeps first and second moments per tracked parameter and applies one
bias-corrected step with PyTorch's defaults (`betas=(0.9, 0.999)`, `eps=1e-8`):

```go
if err := tape.Adam(0.003); err != nil { log.Fatal(err) }
```

`AdamW` uses the same moment defaults but applies decoupled weight decay
separately from the moment update:

```go
if err := tape.AdamW(0.003, 1e-2); err != nil { log.Fatal(err) }
```

`NewStepLR(initialRate, gamma, stepSize)` returns a schedule whose `LR(step)`
starts at `initialRate` for step 0 and decays at `stepSize`, `2*stepSize`, and
so on. Pass its result into `Adam` or `AdamW` for each optimizer step:

```go
schedule, err := nn.NewStepLR(1e-3, 0.5, 2)
if err != nil { log.Fatal(err) }
for step := 0; step < 5; step++ {
	// build the forward graph, call Backward, then:
	if err := tape.AdamW(schedule.LR(step), 1e-2); err != nil { log.Fatal(err) }
}
```

### Training toolkit

The tape also provides mean-reduced `MSELoss(pred, target)` and fused,
numerically stable `BCEWithLogitsLoss(logits, targets)`. Both record one VJP;
the binary loss expects float32 targets in `[0, 1]` and does not expose a
separate sigmoid-plus-log training path.

`SGDMomentum(rate, momentum)` keeps one velocity per tracked parameter and uses
torch's convention `v = momentum*v + gradient`, then `w -= rate*v`:

```go
loss, err := tape.BCEWithLogitsLoss(logits, targets)
if err != nil { log.Fatal(err) }
if err := tape.Backward(loss); err != nil { log.Fatal(err) }
norm, err := tape.ClipGradNorm(1)
if err != nil { log.Fatal(err) }
_ = norm // pre-clip global L2 norm
if err := tape.SGDMomentum(0.01, 0.9); err != nil { log.Fatal(err) }
```

`NewCosineAnnealingLR(initialRate, tMax)` returns
`initialRate*(1+cos(pi*step/tMax))/2`, with `LR(0)` equal to the initial
rate. Read the scheduled rate before the optimizer step, then advance the
schedule after it, matching `torch.optim.lr_scheduler.CosineAnnealingLR`.
`ClipGradNorm(maxNorm)` computes one global L2 norm over all tracked gradients,
scales every gradient when needed, and returns the pre-clip norm.

Dropout is a training wrapper using the tape-owned seeded RNG. Use
`nn.NewTape(seed)` when a reproducible mask is needed; kept values are scaled
by `1/(1-p)` and the backward pass uses that same mask:

```go
tape := nn.NewTape(42)
hidden, err := tape.Dropout(hidden, 0.2)
if err != nil { log.Fatal(err) }
```

The tape is a training surface and has no mode flag. For evaluation, callers
simply do not call `Dropout`, so the eval path is the identity.

For a convergence loop, keep the marked parameters and tape across shuffled
minibatches so Adam retains its per-parameter moments. A typical MNIST-shaped
loop is a seeded `784 -> 128 -> 10` MLP with batch size 128:

```go
tape := nn.NewTape()
w1, _ := tape.Param(weights["w1"]) // [784, 128], seeded He initialization
b1, _ := tape.Param(weights["b1"]) // [128]
w2, _ := tape.Param(weights["w2"]) // [128, 10], seeded He initialization
b2, _ := tape.Param(weights["b2"]) // [10]

for epoch := 0; epoch < 5; epoch++ {
	order := rng.Perm(len(trainLabels))
	for start := 0; start < len(order); start += 128 {
		input, labels := makeBatch(trainImages, trainLabels, order, start, 128)
		hidden, _ := tape.MatMul(input, w1.Value())
		hidden, _ = tape.Add(hidden, b1.Value())
		hidden, _ = tape.Relu(hidden)
		logits, _ := tape.MatMul(hidden, w2.Value())
		logits, _ = tape.Add(logits, b2.Value())
		loss, _ := tape.SoftmaxCrossEntropy(logits, labels)
		_ = tape.Backward(loss)
		_ = tape.Adam(1e-3)
	}
}
```

The repository's convergence proof keeps data loading and seeded initialization
test-side, so `nn` does not add a public dataset or random-initialization API.

AMSGrad and device training are not part of this API. The tape is intended for
the fixed-weight CPU training path; the inference kernels and ONNX graph runner
remain unchanged.

CNN training uses the same tape and parameter flow. Mark convolution, batch
normalization affine, and linear weights with `Param`, then compose the
forward pass with the CNN wrappers before the activation, pooling, global
average, flatten, and classification layers. Convolution gradients support
explicit or automatic padding, strides, dilations, groups, and optional bias;
pooling gradients follow the forward window and `count_include_pad` rules.
The standalone `BatchNormalization` kernel is inference-mode only: its running
mean and variance are constants, while input, scale, and bias receive
gradients. The autodiff tape also exposes training-mode BatchNorm, which
normalizes with biased batch variance, updates running variance with the
unbiased estimator, and differentiates through the batch statistics.

## Layers and Sequential

The layer surface is training sugar over the same tape. It has one `Forward`
method and no train/eval mode flag: `NewSequential` builds layers eagerly on a
tape, `Forward` records the training path, and `Predict` uses a throwaway tape
while structurally skipping layers marked `TrainingOnly`.

```go
tape := nn.NewTape(20260803)
model, err := nn.NewSequential(
    tape,
    nn.Dense(784, 128),
    nn.ReLU(),
    nn.Dropout(0.2),
    nn.Dense(128, 10),
)
if err != nil { log.Fatal(err) }

logits, err := model.Forward(tape, batch)
if err != nil { log.Fatal(err) }
predictions, err := model.Predict(batch) // Dropout is omitted structurally.
if err != nil { log.Fatal(err) }
_ = logits
_ = predictions
```

The catalog layers are:

| Layer | Behavior |
| --- | --- |
| `Dense(in, out)` | He-initialized affine layer; torch Linear weights transpose at load time |
| `Conv2D(in, out, kernel, opts...)` | NCHW convolution with torch `[out,in/groups,kh,kw]` weights, padding, strides, dilations, groups, and optional bias |
| `MaxPool2D` / `AvgPool2D` | NCHW pooling; omitted stride defaults to the kernel size, matching torch |
| `GlobalAvgPool` | Reduces spatial dimensions to `[N,C,1,1]` |
| `BatchNorm2D(features)` | Batch statistics and running-stat updates in `Forward`; running statistics in `Predict` |
| `LayerNorm(dims)` | Learned suffix normalization over an integer or shape slice |
| `Embedding(vocab, dim)` | Int64 `[N]` or `[N,S]` lookup with scatter-add gradients |
| `MultiHeadAttention(embed, heads)` | Mask-free batch-first self-attention over `[batch, sequence, embed]` |
| `Residual(layers...)` | Adds the input to a composable sub-stack; inference honors nested `EvalLayer` paths |
| `ReLU`, `NewSigmoid`, `NewTanh`, `NewGelu`, `Dropout`, `NewFlatten`, `Func` | Stateless, training-only, shape, or callback layers |

`MultiHeadAttention(embed, heads)` accepts and returns `[batch, sequence,
embed]` tensors. It is self-attention only in v1: it has no attention mask,
causal mask, or cross-attention input. `embed` must be divisible by `heads`.
The forward path uses the existing differentiable batched `MatMul`, axis
`Softmax`, `Transpose`, and `Reshape` tape operations, so its parameters receive
the existing VJPs without a layer-specific backward implementation.

Its state dict follows `torch.nn.MultiheadAttention`:
`in_proj_weight`, `in_proj_bias`, `out_proj.weight`, and `out_proj.bias`.
`LoadWeights` accepts torch's `[out,in]` matrices and stores the internal
`[in,out]` layout. In a `Sequential`, a direct layer therefore exposes names
such as `0.in_proj_weight`; a MHA nested in `Residual` uses the recursive name
`0.0.in_proj_weight`. `SaveWeights` reverses these transposes. ONNX export
refuses `MultiHeadAttention` and `Residual` by layer position and kind because
the mask-free composite has no exported layer mapping yet.

An encoder can be assembled without `Func`:

```go
encoder, err := nn.NewSequential(tape,
    nn.Residual(nn.MultiHeadAttention(16, 4)),
    nn.LayerNorm(16),
    nn.Residual(nn.Dense(16, 32), nn.NewGelu(), nn.Dense(32, 16)),
    nn.LayerNorm(16),
)
```

`Func` accepts a tape callback for residual or other composite blocks. The
`EvalLayer` interface is optional: when a layer implements
`PredictForward(x)`, `Sequential.Predict` uses it instead of recording its
training path. `BatchNorm2D` uses this seam and does not need a global mode
flag. `NewSigmoid`, `NewTanh`, `NewGelu`, and `NewFlatten` use the `New` prefix
because the package already exports kernel functions with the shorter names.

`Parameters` returns parameters in layer order. `NamedParameters` follows
torch `nn.Sequential`: layer positions include parameterless layers, so a
model with `Dense`, `ReLU`, `Dropout`, `Dense` exposes `0.weight`, `0.bias`,
`3.weight`, and `3.bias`. `LoadWeights` accepts the map returned by
`LoadSafeTensors`. Torch Linear stores weights as `[out,in]`; `LoadWeights`
transposes them into the tape's internal `[in,out]` layout. Conv2D weights
already match torch and are copied without a transpose. BatchNorm2D loads
`weight`, `bias`, `running_mean`, and `running_var`; torch's
`num_batches_tracked` buffer is tolerated and ignored. Missing, extra, or
mis-shaped names return an error.

### Saving weights and exporting ONNX

`SaveSafeTensors` writes a deterministic, contiguous SafeTensors file for
`F32`, `I64`, and `BOOL` tensors. Names are sorted before the header and data
region are written, so saving the same map twice produces identical bytes.

`Sequential.SaveWeights` uses torch `nn.Sequential` names, includes
`BatchNorm2D`'s `running_mean` and `running_var`, and transposes Dense weights
back to torch Linear's `[out,in]` layout. This is the inverse of
`LoadWeights`, which accepts that torch layout and stores `[in,out]` for the
tape. The resulting file can be loaded with `safetensors.torch.load_file`.

`Sequential.ExportONNX` writes a single-input inference graph with output
`output`. Dense, activations, Conv2D, BatchNorm2D inference, pooling,
GlobalAvgPool, Flatten, and LayerNorm are exported with their trained values
and attributes. Dropout is omitted because `Predict` treats it as an identity.
`Func` and `Embedding` currently return an error naming the layer position and
kind. The first Dense input is declared as `[-1, in]`; the first Conv2D input is
`[-1, in, -1, -1]`. The resulting graph can be passed to `nn.LoadONNX` or an
independent ONNX runtime.

Dimensions are explicit for `Dense`, so adjacent mismatches fail during
`NewSequential` with the layer index and kind. A catalog CNN proof uses a
30,000-row training subset, retains the second convolution's spatial features
before its classifier, and evaluates all 10,000 test images: mean losses are
`0.364155` and `0.110628`, with test accuracy rising from `95.42%` to `97.27%`
in two epochs.

## Supported operators

| Operator | Notes |
| --- | --- |
| `Gemm` | 2-D matrix product with `alpha`, `beta`, `transA`, and `transB` |
| `MatMul` | 2-D fast path plus N-D matrix products with leading-batch broadcasting |
| `Conv` | 2-D NCHW convolution with explicit or `auto_pad` padding, strides, dilations, bias, groups, and depthwise groups |
| `MaxPool` | 2-D NCHW max pooling with kernel shape, padding, strides, and `auto_pad` |
| `AveragePool` | 2-D NCHW average pooling with padding, strides, `auto_pad`, and `count_include_pad` |
| `GlobalAveragePool` | 2-D spatial average pooling retaining singleton spatial dimensions |
| `BatchNormalization` | inference-mode channel normalization with five inputs and configurable epsilon |
| `Pad` | constant padding from attributes or initializer inputs; reflect and edge modes are refused |
| `Add`, `Sub`, `Mul`, `Div`, `Pow` | float32 elementwise operations with broadcasting; shape arithmetic also supports int64 |
| `Clip` | opset-11+ float32 clipping with optional scalar min/max inputs |
| `Relu`, `Sigmoid`, `Tanh`, `Gelu`, `Erf`, `Sqrt` | elementwise activations and math |
| `LayerNormalization` | suffix normalization with configurable axis and epsilon |
| `ReduceMean` | reduction over one or more axes with optional keepdims |
| `Softmax` | numerically stable, configurable axis |
| `Identity` | independent tensor copy |
| `Reshape` | row-major reshape with `-1` inference |
| `Flatten` | collapse dimensions around an axis |
| `Transpose` | explicit permutation or reversed dimensions at any rank |
| `Concat`, `Squeeze`, `Unsqueeze`, `Expand`, `Shape`, `Gather` | standard-domain shape and feature assembly |
| `Slice`, `Split` | standard-domain tensor partitioning and slicing |
| `Equal`, `Greater`, `GreaterOrEqual`, `Where` | broadcast comparisons and selection |
| `Cast` | float32, int64, string, and bool conversions; half targets round and widen to f32 |
| `Constant` | typed tensor attribute |
| `ConstantOfShape` | fills a runtime int64 shape with the typed scalar attribute, defaulting to float32 zero |
| `ai.onnx.ml:OneHotEncoder` | string or int64 categories to float32 indicator columns |
| `ai.onnx.ml:LabelEncoder` | string, int64, or float keys to int64 codes |
| `ai.onnx.ml:Scaler` | `(value - offset) * scale` |
| `ai.onnx.ml:LinearRegressor` | float32 linear prediction with `NONE` or `LOGISTIC` post-transform |
| `ai.onnx.ml:LinearClassifier` | class labels and scores, including `NONE` and `LOGISTIC` |
| `ai.onnx.ml:TreeEnsembleRegressor` | tree traversal, base values, target weights, and post-transform |
| `ai.onnx.ml:TreeEnsembleClassifier` | class probabilities and the binary single-score convention |

The standalone kernel functions can be called without constructing a graph.
Their results are checked against one-operator ONNX graphs executed by
`onnxruntime`, including the padding, stride, dilation, grouping, and pooling
attribute combinations. The package also tests fixed-weight MLP, transformer
encoder, and MNIST-class CNN round trips. The encoder proof contains two-head
self-attention, residual connections, a feed-forward GELU block, and
LayerNormalization; the CNN proof covers convolution, BatchNormalization,
pooling, and a softmax classifier.

## Real-model validation

The gated real-model parity test validates two published checkpoints without
downloading anything. Set `INSYRA_NN_REAL_MODELS_DIR` to a directory containing
these exact files:

- `mobilenetv2-12.onnx` (opset 12)
- `minilm-l6-v2.onnx` (opset 14)

Run the gate with the local `onnxruntime` environment:

```bash
env INSYRA_NN_REAL_MODELS_DIR=$HOME/.cache/insyra-nn-models \
  PATH="$HOME/.cache/insyra-crosslang-venv/bin:$PATH" \
  go test ./nn/ -run RealModel -count=1 -v
```

It feeds deterministic image or token tensors to both `nn` and `onnxruntime`
and compares every output element within f32 tolerance. If the variable is
unset or either file is absent, the gate skips and names
`INSYRA_NN_REAL_MODELS_DIR`; it never accesses the network.

## Real-model smoke test

For a manual smoke run against a local model, set
`INSYRA_NN_REAL_MODEL` to an `.onnx` path and run:

```bash
INSYRA_NN_REAL_MODEL=/path/to/model.onnx go test ./nn/ -run TestRealModelSmoke -v
```

The test loads the model and fails with the loader's complete unsupported
operator list when the graph is outside the supported boundary. For a loaded
model it supplies synthetic inputs from the declared shapes, runs the graph,
and prints each output shape. Dynamic dimensions use size `1`; mask-like
integer and boolean inputs use ones or `true`. This path is intentionally
manual and is not required by CI.

## Performance and positioning

`nn` is measured against ONNX Runtime's CPU provider on the same
machine (8-core Apple M3, best of 5, identical inputs) at the two
validated real-model shapes:

| Model | `nn` (default) | `nn` (CPU only) | ONNX Runtime CPU | Gap |
| --- | --- | --- | --- | --- |
| MobileNetV2, batch 1 | 170 ms | 200 ms | 7.3 ms | ~23x |
| MiniLM-L6-v2, batch 8 × seq 128 | 3.34 s | 4.02 s | 63 ms | ~53x |

The gap is structural, not accidental. ONNX Runtime's hot loops are
hand-written assembly microkernels tuned per CPU architecture; `nn` is
pure Go, and the Go compiler does not auto-vectorize. Closing it would
mean per-architecture assembly, which would surrender the properties
this package exists for. Choose accordingly:

- **Use `nn`** when the model runs inside a Go program with zero
  non-Go dependencies, when auditability matters — every operator and
  gradient here is verified against `onnxruntime`, PyTorch, and finite
  differences, and the parallel and GPU paths are bit-identical to the
  serial CPU reference — or when the workload is modest (small models,
  batch scoring, embedded scoring next to `ml`). The default GPU path
  narrows the largest matrix products.
- **Use ONNX Runtime** (or another native runtime) when inference
  throughput or latency is the requirement. The numbers above are the
  honest distance.

An assembly GEMM microkernel is the single lever that would move both
rows at once; it stays unbuilt until a real workload demands it, per
the same measure-first rule that gates every kernel in this project.

## The closed loop with `ml`

Linear, ridge, lasso, WLS, both decision trees, both random forests, both
gradient boosters, and the exported preprocessing pipeline load and run in
`nn`. The round-trip suite compares every output with the fitted `ml` model
and with `onnxruntime`. The `ai.onnx.ml` operator domain is implemented to
match the conventions in `ml/onnx_export.go`, including its binary
single-score tree-ensemble convention and the two-row binary
`LinearClassifier` convention used for logistic probabilities.

To use a loaded network as an `ml` model, bind it:

```go
model, _ := nn.LoadONNX(file)
regressor, _ := nn.BindRegressor(model, "input", []string{"x1", "x2"})
// regressor satisfies ml.Model structurally: name-bound columns, ml.Score,
// pipelines and conformance checks all work.
score, _ := ml.Score(regressor, table, target, ml.RMSEMetric{})
```

`BindClassifier(model, input, features, classes)` additionally satisfies
`ml.Classifier` and `ml.ProbaModel` — the label is the argmax and the
probability table follows the supplied class order.
