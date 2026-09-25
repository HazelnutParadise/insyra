## Why

`nn`'s two strongest checks never run in CI. The MNIST convergence tests and the parity comparison against five published ONNX checkpoints are gated on `INSYRA_NN_MNIST_DIR` and `INSYRA_NN_REAL_MODELS_DIR`, no workflow sets either, and nothing in the repository carries the data — so `go test ./...` skips them everywhere and a skip reads exactly like a pass (#303, TS-5). Between them they cover the training loop end to end and the loader's whole operator surface, which is the part of `nn` most likely to rot unnoticed.

Measured on 2026-09-20 on an M3: the five tests take 117.7s together (MNIST 93s, real-model parity 24s) and need 288MB of checkpoints plus 11MB of MNIST.

## What Changes

- A new workflow, Neural Network Data Gates, runs the five gated tests when `nn/` changes, when its own files change, and on manual dispatch. No schedule: `nn` only rots when `nn` changes.
- A new `.github/nn-data-manifest.txt` pins every file by sha256 next to an immutable source — a commit in `onnx/models`, a revision on Hugging Face, and PyTorch's MNIST mirror.
- The job caches the data by the manifest's hash, checks every file against its checksum before testing, and fails when any of the five tests did not run rather than reporting a green skip.
- GPU gates stay manual, because no hosted runner has a GPU. `ENG.md` records how to run them and `Docs/nn.md` points a local setup at the manifest.

No library code or user-visible behaviour changes, so there is no changelog entry.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `verification-integrity`: gains a requirement for checks gated on data rather than on a toolchain — CI must supply that data from a pinned source and must fail when the check skips.

## Impact

- New `.github/workflows/nn-data-gates.yml` and `.github/nn-data-manifest.txt`; `ENG.md`, `Docs/nn.md`.
- `api-review.md` TS-5, `delivery-status.md`, issue #303.
