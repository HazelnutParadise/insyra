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

### Requirement: Training converges on a real dataset

The tape SHALL train an MLP classifier on the real MNIST dataset from
seeded initialization to at least 95% test-set accuracy within a bounded
number of epochs, with the final training loss well below the initial
loss. The run SHALL be deterministic under its fixed seed and SHALL be
gated on an operator-provided dataset directory, skipping cleanly and
attempting no network access when it is absent. A dataset-free
micro-convergence test SHALL verify the loop mechanics everywhere.

#### Scenario: MNIST reaches the accuracy target

- **WHEN** the gated test runs with the four IDX files present
- **THEN** the trained MLP SHALL score at least 95% on the 10k test
  images and the loss curve SHALL end well below where it started

#### Scenario: The micro-convergence test runs everywhere

- **WHEN** the ungated test trains the tiny two-class problem
- **THEN** it SHALL reach perfect accuracy in bounded steps with no
  dataset or toolchain involved

#### Scenario: Absent data skips cleanly

- **WHEN** `INSYRA_NN_MNIST_DIR` is unset or files are missing
- **THEN** the test SHALL skip naming the variable and touch no network

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

### Requirement: Sequential composes layers without changing a single digit

`nn` SHALL provide a `Layer` interface (Build, Forward, Parameters)
and a `Sequential` that builds layers eagerly with construction-time
dimension errors naming the layer, trains through `Forward(tape, x)`,
and predicts through `Predict(x)` on a throwaway tape that
structurally skips `TrainingOnly` layers. Parameter names SHALL follow
torch's `nn.Sequential` convention and `LoadWeights` SHALL accept
torch Linear layout. The MNIST training run expressed through
Sequential SHALL reproduce the hand-written tape run's loss curve and
accuracy digit-for-digit under the same seed.

#### Scenario: The sugar changes nothing

- **WHEN** the gated MNIST run is expressed through Sequential with
  the same seed as the hand-written test
- **THEN** per-epoch losses and final accuracy SHALL equal the
  recorded values exactly

#### Scenario: A PyTorch Sequential loads and predicts

- **WHEN** weights trained by a torch `nn.Sequential` MLP are saved
  via safetensors and loaded with `LoadWeights`
- **THEN** `Predict` SHALL match torch's forward within f32 tolerance

#### Scenario: Dropout cannot reach inference

- **WHEN** a model containing Dropout runs `Predict`
- **THEN** the dropout layer SHALL be skipped structurally and the
  output SHALL equal the dropout-free forward exactly

#### Scenario: Dimension errors name the layer at construction

- **WHEN** adjacent layers disagree on dimensions
- **THEN** `NewSequential` SHALL fail naming the layer index and kind,
  before any training begins

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

### Requirement: What nn trains, nn persists, and everyone can read

`nn` SHALL write SafeTensors files that its own loader, the Python
`safetensors` library, and torch read back identically, and a trained
`Sequential` SHALL export an ONNX graph carrying its trained weights
and running statistics that `nn`'s own runtime matches exactly and
`onnxruntime` matches within f32 tolerance. Layers without an ONNX
mapping SHALL refuse export naming the layer.

#### Scenario: SafeTensors round-trips three ways

- **WHEN** trained weights are saved with SaveSafeTensors
- **THEN** LoadSafeTensors SHALL read identical tensors, and torch
  SHALL load the file as a state dict and reproduce Predict through
  the reference gate

#### Scenario: The ONNX export runs everywhere

- **WHEN** a trained Sequential MLP and CNN export to ONNX
- **THEN** nn.LoadONNX SHALL run them matching Predict exactly, and
  onnxruntime SHALL match within f32 tolerance, with BatchNorm
  carrying the trained running statistics

#### Scenario: Unmappable layers refuse by name

- **WHEN** a model containing Func or Embedding exports
- **THEN** ExportONNX SHALL return an error naming the layer and its
  position

