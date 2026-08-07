## ADDED Requirements

### Requirement: An encoder block composes from layers and trains like torch

`nn` SHALL provide a MultiHeadAttention layer with torch weight
conventions and a Residual block combinator, so that an encoder block
composes from layers alone — no Func required — loads torch weights,
and trains one AdamW step matching PyTorch on loss, every gradient,
and every post-step parameter.

#### Scenario: The layer-built encoder matches torch through a step

- **WHEN** the same encoder weights load into both frameworks and both
  run one forward, backward, and AdamW step on the same fixed batch
- **THEN** loss, every gradient, and every updated parameter SHALL
  match within f32 tolerance

#### Scenario: Attention weights round-trip with torch conventions

- **WHEN** torch.nn.MultiheadAttention weights save via safetensors
  and load into the layer
- **THEN** Predict SHALL match torch's batch_first forward within f32
  tolerance

#### Scenario: The whole layer survives finite differences

- **WHEN** the MultiHeadAttention layer's gradients are checked
  against central finite differences on tiny shapes
- **THEN** they SHALL agree within the derived tolerance
