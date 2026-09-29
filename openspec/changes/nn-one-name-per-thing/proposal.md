# Proposal: nn-one-name-per-thing

## Why

NN-1 of the API review ([#265](https://github.com/HazelnutParadise/insyra/issues/265)) found that aliases and twin constructors double much of `nn`'s surface:

- Twelve layers have two constructors that return the same layer: `Dense`/`NewDense`, `ReLU`/`NewReLU`, `Dropout`/`NewDropout`, `Func`/`NewFunc`, `MultiHeadAttention`/`NewMultiHeadAttention`, `Conv2D`/`NewConv2D`, `MaxPool2D`/`NewMaxPool2D`, `AvgPool2D`/`NewAvgPool2D`, `GlobalAvgPool`/`NewGlobalAvgPool`, `BatchNorm2D`/`NewBatchNorm2D`, `LayerNorm`/`NewLayerNorm` and `Embedding`/`NewEmbedding`.
- Type and constant aliases: `SoftmaxCrossEntropy = CrossEntropy`, `MSELoss = MSE`, `BCEWithLogitsLoss = BCEWithLogits`, `Classifier = BoundClassifier`, `Regressor = BoundRegressor`, `MaxPoolOptions` and `AveragePoolOptions = PoolOptions`, `DataType = DType`, and `Float32`, `Float16`, `Float64` for `DTypeFloat32`, `DTypeFloat16`, `DTypeFloat64`.
- Two float32 constructors, `NewTensor` and `NewFloat32Tensor` ("an explicit spelling of NewTensor"), and a third, `NewTensorWithDType(dtype, shape, data []float32)`, whose name promises any dtype while its data can only be float32: it refuses every dtype but `DTypeFloat32`, which makes it `NewTensor` with an argument that must be one value.
- Two float32 accessors with two contracts: `Tensor.Data()` returns nil for a tensor that is not float32, so a caller cannot tell an int64 tensor from an empty one, while `Float32Data()` returns an error, as `Int64Data`, `StringData` and `BoolData` do for their dtypes.

The owner ruled on #211 that each function has one name, and that an old name stays for one release as a Deprecated wrapper or alias that keeps its old meaning.

## What Changes

Every name below is **Deprecated** for one release, keeps its old meaning, and its doc comment names what replaces it:

- The twelve `New…` layer twins, in favour of the bare names (`NewDense` → `Dense`, …).
- `SoftmaxCrossEntropy`, `MSELoss`, `BCEWithLogitsLoss` → `CrossEntropy`, `MSE`, `BCEWithLogits` (the `FitConfig.Loss` selectors; the `Tape` methods of the same names are not affected).
- `Classifier`, `Regressor` → `BoundClassifier`, `BoundRegressor`.
- `MaxPoolOptions`, `AveragePoolOptions` → `PoolOptions`.
- `DataType` → `DType`; `Float32`, `Float16`, `Float64` → `DTypeFloat32`, `DTypeFloat16`, `DTypeFloat64`.
- `NewFloat32Tensor` and `NewTensorWithDType` → `NewTensor`. `NewTensorWithDType` still refuses every dtype but float32 while it lasts.
- `Tensor.Data` → `Tensor.Float32Data`, which reports a nil or non-float32 tensor as an error. `Data` keeps returning nil for such a tensor while it lasts.

`NewSigmoid`, `NewTanh`, `NewGelu` and `NewFlatten` keep their `New` prefix: they have no twin, and the bare names are the kernel functions `Sigmoid`, `Tanh`, `Gelu` and `Flatten`.

Every caller in the repository, library code, tests and `accel`'s tests, moves to the remaining names; only the tests that check the deprecated names still use them. The removals are recorded as an `AGENTS.md` follow-up.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `nn-inference`: adds "A tensor has one constructor and one accessor per dtype" and "A dtype, a binding and pooling options have one name each".
- `nn-training`: adds "A layer and a loss have one name each".

## Impact

- `nn/layers.go`, `nn/layers_catalog.go`, `nn/layers_attention.go`, `nn/fit.go`, `nn/protocol.go`, `nn/kernels.go`, `nn/tensor.go`; the `nn` tests and `accel/internal/wgpu`'s tests that read tensors.
- `Docs/nn.md`, both changelogs, `AGENTS.md` (removal follow-up), `api-review.md` (NN-1), `delivery-status.md`.
