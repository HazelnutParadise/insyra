## ADDED Requirements

### Requirement: The practitioner toolkit matches torch piece by piece

The tape SHALL provide MSE and fused BCE-with-logits losses, momentum
SGD with torch's velocity convention, a cosine annealing schedule, and
global-norm gradient clipping, each verified against PyTorch, with the
composed recipe matching torch at every step of a multi-step
trajectory.

#### Scenario: The composed recipe matches torch stepwise

- **WHEN** a small net trains several steps with BCE-with-logits,
  momentum SGD, cosine annealing, and gradient clipping in both
  frameworks under identical weights and batches
- **THEN** loss, the clipped gradient norm, the learning rate, and
  every parameter SHALL match within f32 tolerance at every step

#### Scenario: Losses survive finite differences

- **WHEN** each new loss is checked against central finite differences
  on tiny shapes
- **THEN** gradients SHALL agree within the derived tolerance

#### Scenario: Clipping uses the global norm

- **WHEN** gradients whose global norm exceeds the maximum are clipped
- **THEN** every gradient SHALL scale by maxNorm over the total norm,
  matching torch's clip_grad_norm_
