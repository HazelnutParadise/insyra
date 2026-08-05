# Proposal: add-nn-sequential-fit

## Why

Training has no front door: `Sequential` offers Forward/Predict/Parameters, and the documented way to train is a hand-written epoch loop over tape calls. That loop is silent — a 2-epoch MNIST run is 12.5 seconds of nothing, a bigger job is minutes of apparent hang — and every user re-writes the same shuffling, batching, and stepping boilerplate. A `Fit` method with default per-epoch progress output gives training the Keras/scikit-learn-shaped entry the rest of the library already imitates, without removing the hand-written loop for those who need it.

## What Changes

- `Sequential.Fit(x, y *Tensor, cfg FitConfig) (*FitResult, error)`: epochs, batch size, deterministic seeded shuffling, optimizer selection from the existing tape optimizers (SGD, SGD-momentum, Adam, AdamW), loss selection from the existing losses (softmax cross-entropy, MSE, BCE-with-logits), optional validation tensors with per-epoch validation loss.
- Default progress: one `LogInfo` line per epoch (epoch k/N, mean training loss, validation loss when present, elapsed, rows/s) through the root logger — this is the "it looks hung" fix. `FitConfig.Progress` accepts a callback for custom reporting; `Quiet` silences the default line.
- Determinism is the contract, not an option: the same inputs, config, and seed produce the same parameter trajectory, and the ENG.md gate extends to Fit — **a Fit-driven MNIST run must reproduce the documented hand-written loop's loss curve to the last digit** under equivalent configuration.
- The hand-written tape loop remains fully supported and documented; Fit is sugar over it, and the sugar-changes-nothing proof is exact.
- Docs (`Docs/nn.md` training section rewritten around Fit with the hand loop kept as the advanced path), both changelogs, skills sync.

## Capabilities

### New Capabilities

- `nn-training-frontdoor`: packaged training entry on `Sequential` — deterministic, progress-reporting, optimizer- and loss-selecting, bit-identical to the hand-written loop it wraps.

### Modified Capabilities

None — existing `nn-training` requirements (op gradients, optimizer parity) are untouched; Fit composes them.

## Impact

- **Code**: `nn/sequential.go` (or a sibling `nn/fit.go`), config/result types; no changes to tape ops, optimizers, or losses themselves.
- **API**: additive — one method, two types. Tensor-in/tensor-out, consistent with the tape surface; `ml`-level DataTable integration is out of scope here.
- **Docs**: `Docs/nn.md`, `CHANGELOG.md` / `CHANGELOG_TW.md`, `skills/insyra/`.
- **Dependencies**: none.
