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
`float32` is the implemented arithmetic type. Other declared types are
refused by name instead of being silently converted.

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

`Shape`, `Strides`, and `Data` return copies. Elementwise `Add`, `Sub`, and
`Mul` use NumPy-style trailing-dimension broadcasting. Incompatible shapes
produce an error that names both shapes.

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
| `Add`, `Sub`, `Mul` | float32 elementwise operations with broadcasting |
| `Relu`, `Sigmoid`, `Tanh` | elementwise activations |
| `Softmax` | numerically stable, configurable axis |
| `Identity` | independent tensor copy |
| `Reshape` | row-major reshape with `-1` inference |
| `Flatten` | collapse dimensions around an axis |
| `Transpose` | explicit permutation or reversed dimensions |
| `Cast` | float32 to float32 only |
| `Constant` | float32 tensor attribute |

The standalone kernel functions can be called without constructing a graph.
Their results are checked against one-operator ONNX graphs executed by
`onnxruntime`, and the package also tests a fixed-weight MLP round trip.
