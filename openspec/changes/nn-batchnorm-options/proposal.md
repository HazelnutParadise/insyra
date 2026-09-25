## Why

The third batch of #213 (NN-3). Training batch normalization read `options ...float32` by position, momentum first and epsilon second, so setting epsilon meant restating momentum and a call like `…, 0.2, 1e-5)` did not say which number was which. Both are training hyperparameters, statistical settings under the owner's 2026-09-25 rule, so they go in an options struct. The rest of `nn`'s optional parameters each mirror a single ONNX attribute and already fit the rule.

## What Changes

- **BREAKING**: `Tape.BatchNormalizationTraining` and its alias `BatchNormTraining` take `opts ...BatchNormOptions{Momentum, Epsilon}`. A zero field keeps torch.nn.BatchNorm2d's default (0.1, 1e-5); more than one struct is an error.
- `BatchNorm2D`'s training path passes its momentum and epsilon through the struct.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `optional-values`: training batch normalization takes its settings by name.

## Impact

- `nn/autodiff_cnn.go`, `nn/layers_catalog.go`, `nn/autodiff_catalog_test.go`, `nn/batchnorm_options_test.go` (new).
- `Docs/nn.md`, both changelogs.
