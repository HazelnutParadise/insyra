# Proposal: nn-layernorm-typed-constructors

## Why

NN-1 of the API review ([#265](https://github.com/HazelnutParadise/insyra/issues/265)) also found `LayerNorm(dims interface{}) Layer`. It accepts an `int` or a `[]int`, mirroring torch's `normalized_shape`, and anything else compiles and becomes a layer normalizing a dimension of size 0. Measured on 2026-09-30 against `origin/0.4` (7228ed2e): `LayerNorm(int64(4))` builds a layer whose `Build` fails with `layernorm dimensions must be positive, got [0]`, an error that names neither the type that was passed nor the constructor that dropped it.

## What Changes

- New `LayerNormShape(dims []int) Layer` normalizes the trailing `len(dims)` dimensions, as torch's `normalized_shape` list does. It copies `dims`. A caller who holds a shape can pass it to a constructor that takes only a shape.
- `LayerNorm` keeps its `interface{}` parameter and every input it accepted before. An `int` builds one dimension, and a `[]int` builds the same layer as `LayerNormShape`.
- `NewLayerNorm(dims interface{})`, deprecated by `nn-one-name-per-thing`, keeps its old meaning and does the same as `LayerNorm`. It is removed in 0.4.
- Left on 0.4: `LayerNorm(dim int)`, which narrows the parameter to `int` as a breaking change, and the error that names the type of a bad input. On this line `LayerNorm(int64(4))` still fails in `Build` with `got [0]`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `nn-training`: adds "LayerNormShape normalizes trailing dimensions" and "LayerNorm and NewLayerNorm keep the inputs they accepted".

## Impact

- `nn/layers_catalog.go` and the `nn` tests.
- `Docs/nn.md`, `CHANGELOG.md` and `CHANGELOG_TW.md`.
