# Change: The CNN operator family — an image classifier runs in pure Go

## Why
The third and last inference operator family. MLP proved the pipeline, attention carries the embedding and LLM value; convolution completes the coverage a general ONNX runtime is expected to have, and it is the family whose weight sits in one operator: `Conv` with padding, strides, dilations and groups is where implementations quietly disagree, which is exactly what the one-op parity harness exists to catch — the generated cases enumerate the attribute combinations rather than sampling them.

## What Changes
- CNN-family kernels: `Conv` (2-D; padding including `auto_pad` conventions, strides, dilations, groups), `MaxPool`, `AveragePool` (including `count_include_pad`), `GlobalAveragePool`, `BatchNormalization` (inference form), `Pad` (constant mode)
- One-op parity rows per kernel with attribute-combination coverage: asymmetric padding, stride>1, dilation>1, grouped and depthwise convolution, pool padding modes
- Whole-model proof: an MNIST-class CNN — conv/pool stacks into a dense head — built with the Python `onnx` package with fixed weights, run by both sides, matched within f32 tolerance
- Documentation: operator table in `Docs/dl.md`, changelogs, skills

## Impact
- Affected specs: `dl-inference`
- Affected code: `dl/` kernels; docs, changelogs, skills
- Blocked by: `add-dl-onnx-mlp-inference`
- Additive
