# Design: nn-layernorm-typed-constructors

## Add a typed constructor, keep the one that exists

`LayerNorm` keeps its `interface{}` parameter and the inputs it accepted, so no caller that compiles today stops compiling. Go has no union type for "an int or a slice of ints", so the choice between the two forms stays at runtime inside `LayerNorm`. `LayerNormShape(dims []int)` is the typed form for a caller who already holds a shape: it is named for torch's `normalized_shape`, and it cannot be called with the wrong type.

Both constructors build the same `layerNormLayer`, so saving, loading and ONNX export are unchanged. `LayerNormShape` copies its argument, so changing the caller's slice after construction does not change the layer.

## What is refused and where

Validation stays in `Build`, where every other catalog layer reports an invalid setting, so a zero, a negative size or an empty shape is reported with the layer's position by `NewSequential`. `LayerNormShape(nil)` and `LayerNormShape([]int{})` fail with `layernorm dimensions must not be empty`.

A size that is not an `int` or `[]int`, such as `LayerNorm(int64(4))`, still falls to the default arm, which builds a dimension of 0 and fails in `Build` with `got [0]`. That behaviour stays on this line. Naming the bad type is the part left on 0.4, where `LayerNorm` takes only `int`.

## Verification

- `LayerNorm(4)` builds weight `[4]`; `LayerNormShape([]int{2, 3})` builds `[2 3]`, normalizes a `[2 2 3]` input the same way `LayerNormalization` does, and is unaffected by changing the source slice afterwards.
- `LayerNorm(0)`, `LayerNorm(-2)`, `LayerNormShape(nil)`, `LayerNormShape([]int{})` and `LayerNormShape([]int{2, 0})` are refused by `NewSequential`, naming `layer 0 (LayerNorm)`.
- `NewLayerNorm(4)` and `NewLayerNorm([]int{2, 3})` build the same shapes as `LayerNorm`; `NewLayerNorm("4")` still fails with the dimension error.
