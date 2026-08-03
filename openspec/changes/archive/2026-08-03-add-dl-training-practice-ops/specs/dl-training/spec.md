## ADDED Requirements

### Requirement: Practice ops train the way practitioners train

The tape SHALL provide inverted dropout (train-mode masking scaled by
1/(1−p) from the seeded RNG, eval-mode identity, gradients routed
through the mask), decoupled weight decay applied separately from the
Adam moment update matching `torch.optim.AdamW`, and a step-decay
learning-rate schedule matching `torch.optim.lr_scheduler.StepLR`.
Multi-step AdamW+StepLR trajectories SHALL match PyTorch under
identical weights with dropout disabled; dropout SHALL be verified by
mask statistics, scaling, determinism under seed, and eval identity.

#### Scenario: AdamW and StepLR trajectories match PyTorch

- **WHEN** the same small MLP trains several steps in both frameworks
  with decoupled weight decay and step decay, dropout disabled
- **THEN** loss and every parameter SHALL match within f32 tolerance at
  every step

#### Scenario: Dropout behaves and reproduces

- **WHEN** train-mode dropout runs under a fixed seed
- **THEN** the kept fraction SHALL be statistically consistent with
  1−p, kept values SHALL be scaled by 1/(1−p), the mask SHALL be
  identical across runs with the same seed, gradients SHALL flow only
  through kept elements, and eval mode SHALL be the identity

#### Scenario: Decoupled means decoupled

- **WHEN** the same configuration runs with coupled L2 emulation and
  with decoupled decay
- **THEN** the trajectories SHALL differ, and only the decoupled one
  SHALL match `torch.optim.AdamW`
