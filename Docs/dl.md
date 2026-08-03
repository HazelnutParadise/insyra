# `dl` Package

The `dl` package loads a focused subset of ONNX models and runs them with pure
Go. It is intended for small inference graphs such as exported multilayer
perceptrons, transformer encoder blocks, and MNIST-class CNN classifiers. The
graph is validated when it is loaded, and runtime inputs and outputs are bound
by name.

## Installation

```bash
go get github.com/HazelnutParadise/insyra/dl
```

## Tensors

`Tensor` stores row-major N-dimensional data and carries an explicit `DType`.
`float32` is the arithmetic type used by the inference kernels. `int64`,
`string`, and `bool` are also stored for categorical preprocessing and
classifier outputs.

```go
input, err := dl.NewTensor(
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

There are two opt-out switches:

- Set `INSYRA_ACCEL_DISABLE_WGPU=1` to disable the builtin WebGPU backend for
  the process.
- Call `dl.RegisterDeviceMatMul(nil)` to clear the hook programmatically and
  restore the CPU path.

The device path is not wired in `-race` builds because the upstream Metal
binding currently trips `checkptr`; those builds remain on the CPU path.

## Loading and running an ONNX model

```go
file, err := os.Open("model.onnx")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

model, err := dl.LoadONNX(file)
if err != nil {
    log.Fatal(err)
}

outputs, err := model.Run(map[string]*dl.Tensor{"input": input})
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

## Loading SafeTensors weights

`LoadSafeTensors` reads a complete SafeTensors file from an `io.Reader` and
returns a `map[string]*dl.Tensor` keyed by the file's tensor names:

```go
file, err := os.Open("weights.safetensors")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

weights, err := dl.LoadSafeTensors(file)
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

`F32`, `I64`, and `BOOL` load natively into `float32`, `int64`, and `bool`
Tensors. `F64`, `F16`, `BF16`, quantized dtypes, and other unsupported dtypes
are refused together in one error, with every tensor name and dtype listed.
There is no silent widening or narrowing, and the loader does not mmap or load
tensors lazily.

## Training and autodiff

`dl` also provides a small reverse-mode tape for MLP, attention-family, and
CNN training. The tape calls
the existing tensor kernels during the forward pass and stores one VJP record
per call; it does not transform an ONNX graph.

```go
tape := dl.NewTape()
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
mode `BatchNormalization`. `Gemm` accepts alpha,
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

Weight decay, AMSGrad, schedules, and device training are not part of this
API. The tape is intended for the fixed-weight CPU training path; the
inference kernels and ONNX graph runner remain unchanged.

CNN training uses the same tape and parameter flow. Mark convolution, batch
normalization affine, and linear weights with `Param`, then compose the
forward pass with the CNN wrappers before the activation, pooling, global
average, flatten, and classification layers. Convolution gradients support
explicit or automatic padding, strides, dilations, groups, and optional bias;
pooling gradients follow the forward window and `count_include_pad` rules.
BatchNormalization is inference-mode only: its running mean and variance are
constants, while input, scale, and bias receive gradients. Training-mode batch
statistics are refused rather than differentiated with different semantics.

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
| `Add`, `Sub`, `Mul`, `Div`, `Pow` | float32 elementwise operations with broadcasting |
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
| `Cast` | float32, int64, string, and bool conversions used by the exporter |
| `Constant` | typed tensor attribute |
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

## Real-model smoke test

For a manual smoke run against a local model, set
`INSYRA_DL_REAL_MODEL` to an `.onnx` path and run:

```bash
INSYRA_DL_REAL_MODEL=/path/to/model.onnx go test ./dl -run TestRealModelSmoke -v
```

The test loads the model and fails with the loader's complete unsupported
operator list when the graph is outside the supported boundary. For a loaded
model it supplies synthetic inputs from the declared shapes, runs the graph,
and prints each output shape. Dynamic dimensions use size `1`; mask-like
integer and boolean inputs use ones or `true`. This path is intentionally
manual and is not required by CI.

## The closed loop with `ml`

Linear, ridge, lasso, WLS, both decision trees, both random forests, both
gradient boosters, and the exported preprocessing pipeline load and run in
`dl`. The round-trip suite compares every output with the fitted `ml` model
and with `onnxruntime`. The `ai.onnx.ml` operator domain is implemented to
match the conventions in `ml/onnx_export.go`, including its binary
single-score tree-ensemble convention and the two-row binary
`LinearClassifier` convention used for logistic probabilities.

To use a loaded network as an `ml` model, bind it:

```go
model, _ := dl.LoadONNX(file)
regressor, _ := dl.BindRegressor(model, "input", []string{"x1", "x2"})
// regressor satisfies ml.Model structurally: name-bound columns, ml.Score,
// pipelines and conformance checks all work.
score, _ := ml.Score(regressor, table, target, ml.RMSEMetric{})
```

`BindClassifier(model, input, features, classes)` additionally satisfies
`ml.Classifier` and `ml.ProbaModel` — the label is the argmax and the
probability table follows the supplied class order.
