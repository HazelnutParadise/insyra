# Change: Sequential — a layer surface over the tape, proved digit-for-digit

## Why

The tape trains everything, but composing a model by hand is the
GradientTape level of ergonomics. A layer surface makes `nn` friendly
without a second API: layers are specs that materialize onto the tape,
Sequential composes them, and PyTorch's naming convention buys direct
weight interop. The design is recorded in ENG.md: one Forward, no mode
flag, structural train/eval separation, explicit dimensions, exact
reproduction as the sugar-changes-nothing proof.

## What Changes

- `Layer` interface: `Build(t *Tape) error` (parameters materialize on
  attach, seeded, in layer order), `Forward(t *Tape, x *Tensor)
  (*Tensor, error)`, `Parameters() []*Parameter`. A `TrainingOnly`
  marker interface tags layers that exist only for training.
- `NewSequential(tape, layers...)` builds every layer eagerly —
  dimension mismatches fail at construction naming the layer index and
  kind. `Forward(tape, x)` is the training path; `Predict(x)` runs on a
  throwaway tape and structurally skips `TrainingOnly` layers.
- v1 layers: `Dense(in, out)` (He init from the tape's seeded RNG,
  bias), `ReLU`, `Sigmoid`, `Tanh`, `Gelu`, `Dropout(p)`
  (TrainingOnly), `Flatten`, and `Func` (wraps any
  `func(*Tape, *Tensor) (*Tensor, error)` — the escape hatch that
  keeps one API sufficient).
- Weight interop: parameter names follow torch `nn.Sequential`
  (`0.weight`, `0.bias`, `3.weight`, … — indices count all layers,
  parameterless ones skip numbers exactly as torch does).
  `LoadWeights(map[string]*Tensor)` accepts `LoadSafeTensors` output
  and transposes torch's `[out,in]` Linear layout at the boundary,
  documented. `NamedParameters()` exposes the same names.
- Docs, changelogs both languages, skills — same change.

## Non-Goals

- No Conv/pooling/BatchNorm/LayerNorm/Embedding layers (M26, next
  change), no Functional or subclassing API, no model serialization
  writer, no loss layers (the fused SoftmaxCrossEntropy stays outside
  the model, standard practice).

## Impact

- Affected specs: `nn-training`
- Affected code: new nn layer files + tests, a torch interop fixture,
  docs, changelogs, skills.
