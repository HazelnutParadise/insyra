# Design: nn-one-name-per-thing

## Which name stays

The rule: the name `Docs/nn.md` and the repository's own code already use, and where that leaves a choice, the name PyTorch uses.

| Pair | Stays | Evidence |
| --- | --- | --- |
| twelve layer twins | bare name (`Dense`, `ReLU`, …) | `Docs/nn.md` writes only the bare names (`nn.Dense` ten times, `nn.NewDense` never), and so do the tests; PyTorch names its layers the same way (`nn.ReLU()`, `nn.Conv2d`, `nn.LayerNorm`). |
| loss selectors | `CrossEntropy`, `MSE`, `BCEWithLogits` | The types are declared under these names and the aliases point at them; `Docs/nn.md` and every `Fit` test write `nn.CrossEntropy{}` and `nn.MSE{}`. They sit in `FitConfig.Loss`, which already says loss, the way `Optimizer: nn.Adam{}` carries no suffix. |
| bound adapters | `BoundClassifier`, `BoundRegressor` | What `BindClassifier`/`BindRegressor` return; the short aliases appear nowhere. `Classifier` also collides in meaning with `ml.Classifier`, the interface the adapter satisfies. |
| pool options | `PoolOptions` | The type every pooling kernel and layer signature takes; the aliases appear nowhere. |
| dtype | `DType`, `DTypeFloat32`, … | `Tensor.DType()` returns it; the full enum is `DType`-prefixed; `DTypeFloat32` is used 98 times, the short constants never. |
| float32 constructor | `NewTensor` | The one `Docs/nn.md` shows; 48 uses against 27 for `NewFloat32Tensor`; PyTorch's `torch.tensor` makes float32 by default. |
| float32 accessor | `Float32Data` | See below. |

`NewSigmoid`, `NewTanh`, `NewGelu` and `NewFlatten` are not twins and stay: their bare names belong to the kernel functions on tensors. `Docs/nn.md` already says so.

## `Data` and `Float32Data`

The finding asked for `Data` to report an error or to go. Giving `Data` an error result would leave two names for one accessor, and `Data` is the one that does not say which dtype it reads, in a set where `Int64Data`, `StringData` and `BoolData` do. `Float32Data` stays; `Data` is deprecated and, following the #211 ruling, keeps returning nil for a non-float32 tensor until it is removed. The change to its call sites is mechanical: `x.Data()` becomes `x.Float32Data()` with its error checked. Tests in package `nn` read through one helper that fails the test on the error.

## `NewTensorWithDType`

Fixing the behaviour would mean a `data any` parameter, the kind of untyped argument this review removes elsewhere, and it would duplicate `NewInt64Tensor`, `NewStringTensor` and `NewBoolTensor`. Renaming it leaves a function whose only valid dtype argument is `DTypeFloat32`. It is deprecated in favour of `NewTensor`, with its refusal of other dtypes intact.

## Verification

- A test parses the `nn` sources and requires every deprecated name's doc comment to contain `Deprecated: use <replacement>` and the removal sentence (red: none has one).
- The deprecated constructors build layers with the same parameters, in the same order and with the same values under one seed, as the names that replace them; the aliases are the same types; the constants have the same values; `Data` still returns nil for an int64 tensor while `Float32Data` reports the dtype; `NewTensorWithDType` still refuses `DTypeInt64`.
- `golangci-lint` (staticcheck SA1019) fails on any use of a deprecated name from another package, which covers `accel`'s tests.
