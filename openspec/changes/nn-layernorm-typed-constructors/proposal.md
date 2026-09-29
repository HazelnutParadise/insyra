# Proposal: nn-layernorm-typed-constructors

## Why

NN-1 of the API review ([#265](https://github.com/HazelnutParadise/insyra/issues/265)) also found `LayerNorm(dims interface{}) Layer`. It accepts an `int` or a `[]int`, mirroring torch's `normalized_shape`, and anything else compiles and becomes a layer normalizing a dimension of size 0. Measured on 2026-09-30 against `origin/0.4` (7228ed2e): `LayerNorm(int64(4))` builds a layer whose `Build` fails with `layernorm dimensions must be positive, got [0]`, an error that names neither the type that was passed nor the constructor that dropped it.

## What Changes

- **BREAKING**: `LayerNorm(dim int) Layer` normalizes the last dimension, of size `dim`. A call with an integer constant, `LayerNorm(16)`, compiles unchanged.
- New `LayerNormShape(dims []int) Layer` normalizes the trailing `len(dims)` dimensions, as torch's `normalized_shape` list does. It copies `dims`. A call that passed a slice to `LayerNorm` moves to it.
- `NewLayerNorm(dims interface{})`, deprecated by `nn-one-name-per-thing`, keeps its old meaning for one release and now names both constructors: an `int` builds `LayerNorm`, a `[]int` builds `LayerNormShape`, and any other value builds the layer it built before.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `nn-training`: adds "LayerNorm takes a typed size".

## Impact

- `nn/layers_catalog.go` and the `nn` tests.
- `Docs/nn.md`, both changelogs, `api-review.md` (NN-1), `delivery-status.md`.
