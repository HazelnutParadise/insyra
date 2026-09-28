## MODIFIED Requirements

### Requirement: Sequential composes layers without changing a single digit

`nn` SHALL provide a `Layer` interface (Build, Forward, Parameters)
and a `Sequential` that builds layers eagerly with construction-time
dimension errors naming the layer, trains through `Forward(tape, x)`,
and predicts through `Predict(x)` on a throwaway tape that
structurally skips `TrainingOnly` layers. Parameter names SHALL follow
torch's `nn.Sequential` convention and `LoadWeights` SHALL accept
torch Linear layout. The MNIST training run expressed through
Sequential SHALL reproduce the hand-written tape run's loss curve and
accuracy digit-for-digit under the same seed, compared against that
run executed in the same process rather than against recorded numbers,
which f32 arithmetic does not carry between platforms.

#### Scenario: The sugar changes nothing

- **WHEN** the gated MNIST run is expressed through Sequential and
  through the hand-written tape loop in the same process, under the
  same seed and hyperparameters
- **THEN** every epoch's mean loss and accuracy SHALL be identical
  between the two runs
- **AND** the absolute claims that remain SHALL be ones any platform
  meets: the loss falls and the accuracy reaches its target

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
