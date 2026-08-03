## ADDED Requirements

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
