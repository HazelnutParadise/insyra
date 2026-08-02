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

### Requirement: An encoder block trains one Adam step matching PyTorch

The tape SHALL differentiate the attention family — batched MatMul with
broadcast-reduced gradients, axis-Softmax, LayerNormalization with input,
scale, and bias gradients, Gelu, and the shape operations the encoder block
backpropagates through — and an Adam optimizer with bias-corrected moments
SHALL update tracked parameters. A fixed two-head encoder block loaded from
one SafeTensors file by both PyTorch and dl SHALL agree on the loss, every
parameter gradient, and every post-step parameter value within f32
tolerance after one Adam step.

#### Scenario: One Adam training step agrees end to end

- **WHEN** the same encoder weights are loaded by PyTorch and dl, and both
  run one forward, backward, and Adam step on the same fixed batch
- **THEN** loss, every gradient, and every updated parameter SHALL agree
  within f32 tolerance through the reference gate

#### Scenario: Every new VJP survives finite differences without a toolchain

- **WHEN** each attention-family VJP is compared against central finite
  differences on a tiny shape in pure Go
- **THEN** they SHALL agree within the tolerance the test derives from the
  perturbation

#### Scenario: Unneeded gradients refuse rather than guess

- **WHEN** backpropagation reaches an operation this change does not
  differentiate
- **THEN** the tape SHALL return an error naming the operation, never a
  silent zero gradient

### Requirement: A CNN trains one Adam step matching PyTorch

The tape SHALL differentiate the CNN family — Conv with input, weight, and
bias gradients under the exercised attribute combinations, MaxPool routing
to the selected maxima, AveragePool and GlobalAveragePool spreading under
the forward's count_include_pad semantics, and inference-mode
BatchNormalization with input, scale, and bias gradients over loaded
statistics. A fixed MNIST-class CNN loaded from one SafeTensors file by
both PyTorch (eval-mode BatchNorm) and dl SHALL agree on the loss, every
parameter gradient, and every post-step parameter within f32 tolerance
after one Adam step.

#### Scenario: One CNN Adam step agrees end to end

- **WHEN** the same CNN weights are loaded by PyTorch and dl and both run
  one forward, backward, and Adam step on the same fixed batch
- **THEN** loss, every gradient, and every updated parameter SHALL agree
  within f32 tolerance through the reference gate

#### Scenario: Every CNN VJP survives finite differences without a toolchain

- **WHEN** each CNN-family VJP is compared against central finite
  differences on tiny shapes, including grouped Conv and asymmetric pads
- **THEN** they SHALL agree within the tolerance the test derives from the
  perturbation

#### Scenario: Training-mode batch statistics are refused

- **WHEN** backpropagation would require gradients of batch-computed mean
  or variance
- **THEN** the tape SHALL refuse naming the operation, matching the
  forward's inference-only contract

