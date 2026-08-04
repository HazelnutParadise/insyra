## ADDED Requirements

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
