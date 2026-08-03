## ADDED Requirements

### Requirement: The catalog trains a CNN and embeds tokens, torch-verified

The layer catalog SHALL include Conv2D, MaxPool2D, AvgPool2D,
GlobalAvgPool, BatchNorm2D, LayerNorm, and Embedding. BatchNorm2D
SHALL use batch statistics with running-stat updates in training and
running statistics in Predict, with its three-term gradient verified
against `torch.nn.BatchNorm2d` in train mode. Embedding SHALL
scatter-add its gradient into the table, verified against
`torch.nn.Embedding`. A Sequential CNN SHALL converge on MNIST to at
least 97% test accuracy, and a torch-trained CNN SHALL load and
predict within f32 tolerance.

#### Scenario: BatchNorm trains like torch

- **WHEN** a small network with BatchNorm2D trains several steps in
  both frameworks under identical weights and batches
- **THEN** loss, every gradient, every parameter, and the running
  statistics SHALL match within f32 tolerance at every step

#### Scenario: Embedding gradients scatter correctly

- **WHEN** repeated token indices appear in one batch
- **THEN** their gradient rows SHALL accumulate, matching torch and
  finite differences

#### Scenario: A Sequential CNN converges

- **WHEN** the gated MNIST CNN run trains within its bounded epochs
- **THEN** test accuracy SHALL reach at least 97% with a sane loss
  curve

#### Scenario: A torch CNN loads and predicts

- **WHEN** a torch Sequential CNN's safetensors weights load via
  LoadWeights
- **THEN** Predict SHALL match torch eval-mode forward within f32
  tolerance
