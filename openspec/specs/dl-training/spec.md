# dl-training Specification

## Purpose
TBD - created by archiving change add-dl-autodiff-mlp. Update Purpose after archive.
## Requirements
### Requirement: The tape differentiates the MLP family and PyTorch agrees

`dl` SHALL provide reverse-mode autodiff as a tape over the existing plain
kernels: training wrappers record operations, `Backward` walks the tape
calling per-op VJPs, and the inference kernels remain untouched. The MLP
family — MatMul, Add, Relu, Sigmoid, Tanh, Gemm as used by exported MLPs —
SHALL be differentiable, with softmax and cross-entropy fused into one loss
whose backward is the stable softmax−onehot form. One SGD step SHALL update
tracked parameters. First-step gradients and loss SHALL match PyTorch under
identical SafeTensors-loaded weights within f32 tolerance.

#### Scenario: First-step gradients match PyTorch

- **WHEN** the same two-layer MLP weights are loaded from one SafeTensors
  file by both PyTorch and dl, and both run one forward/backward on the
  same fixed batch
- **THEN** the loss and every parameter gradient SHALL agree within f32
  tolerance, verified through the reference gate

#### Scenario: The tape agrees with finite differences without any toolchain

- **WHEN** a tiny network's tape gradients are compared against central
  finite differences in pure Go
- **THEN** they SHALL agree within the tolerance the test derives from the
  perturbation, with no reference toolchain involved

#### Scenario: Inference is untouched

- **WHEN** code uses the existing kernels or graph interpreter without the
  tape
- **THEN** behavior and results SHALL be byte-for-byte unchanged

