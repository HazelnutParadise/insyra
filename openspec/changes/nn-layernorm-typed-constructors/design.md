# Design: nn-layernorm-typed-constructors

## Two constructors, not one generic

Go has no union type for "an int or a slice of ints". A generic `LayerNorm[T int | []int](dims T)` compiles for both, but a caller cannot pass it as a value without instantiating it and the documentation would show a constraint instead of a size. Two constructors say what each argument is. `LayerNorm(dim int)` keeps the short name for the common case, one feature dimension, which is also how `Docs/nn.md` and the encoder example call it (`nn.LayerNorm(16)`), so those calls compile unchanged.

`LayerNormShape` is named for torch's `normalized_shape`. Both constructors build the same `layerNormLayer`, so saving, loading and ONNX export are unchanged.

## What is refused and where

Validation stays in `Build`, where every other catalog layer reports an invalid setting, so a zero, a negative size or an empty shape is reported with the layer's position by `NewSequential`. `LayerNormShape(nil)` and `LayerNormShape([]int{})` fail there with `layernorm dimensions must not be empty`.

## Verification

- `reflect.TypeOf(nn.LayerNorm).In(0)` is `int` and `LayerNormShape` exists (red: `interface{}` and no such function).
- `LayerNormShape([]int{2, 3})` builds weight and bias of shape `[2 3]` and normalizes a `[N 2 3]` input the same way the old `LayerNorm([]int{2, 3})` did; changing the caller's slice after construction does not change the layer.
- `LayerNorm(4)` builds `[4]`; `LayerNormShape(nil)` and `LayerNorm(0)` fail in `NewSequential` naming the layer.
- `NewLayerNorm(4)`, `NewLayerNorm([]int{2, 3})` and `NewLayerNorm(int64(4))` behave as before.
