# `dl` Package

The `dl` package loads a focused subset of ONNX models and runs them with pure
Go. It is intended for small inference graphs such as exported multilayer
perceptrons. The graph is validated when it is loaded, and runtime inputs and
outputs are bound by name.

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
float32 arithmetic uses NumPy-style trailing-dimension broadcasting.
`NewInt64Tensor`, `NewStringTensor`, and `NewBoolTensor` build the control and
label tensors used by ONNX graphs.

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

## Supported operators

| Operator | Notes |
| --- | --- |
| `Gemm` | 2-D matrix product with `alpha`, `beta`, `transA`, and `transB` |
| `MatMul` | 2-D matrix product |
| `Add`, `Sub`, `Mul`, `Div` | float32 elementwise operations with broadcasting |
| `Relu`, `Sigmoid`, `Tanh` | elementwise activations |
| `Softmax` | numerically stable, configurable axis |
| `Identity` | independent tensor copy |
| `Reshape` | row-major reshape with `-1` inference |
| `Flatten` | collapse dimensions around an axis |
| `Transpose` | explicit permutation or reversed dimensions |
| `Concat`, `Unsqueeze`, `Gather` | standard-domain shape and feature assembly |
| `GreaterOrEqual`, `Where` | standard-domain categorical missing-value handling |
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
`onnxruntime`, and the package also tests a fixed-weight MLP round trip.

## The closed loop with `ml`

Linear, ridge, lasso, WLS, both decision trees, both random forests, both
gradient boosters, and the exported preprocessing pipeline load and run in
`dl`. The round-trip suite compares the result with the fitted `ml` model and
with `onnxruntime`. The `ai.onnx.ml` operator domain is implemented to match
the conventions in `ml/onnx_export.go`, including the binary single-score
tree-classifier convention. One exporter/runtime mismatch remains for the
binary `LinearClassifier`: `ml` writes one score with `post_transform=LOGISTIC`,
while the current `onnxruntime` reference returns that score and its
complement without applying the logistic transform. The strict closure test
keeps this case visible instead of treating the two references as equivalent.

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
