# Tasks

## 1. Attention VJPs

- [ ] 1.1 Batched MatMul VJP with batch-shape broadcast reduction; axis-Softmax VJP; LayerNormalization VJP (input, scale, bias)
- [ ] 1.2 Gelu (exact), Erf, Sqrt, Pow, ReduceMean VJPs
- [ ] 1.3 Shape-op VJPs the encoder needs: Transpose (inverse permutation), Reshape/Flatten, Squeeze/Unsqueeze, and whichever of Slice/Concat/Split the encoder wrappers use; unreached ops refuse naming the operation

## 2. Adam

- [ ] 2.1 Bias-corrected Adam over tracked parameters, per-parameter moment state, PyTorch-default semantics; no weight decay, no amsgrad

## 3. Proof

- [ ] 3.1 Central finite-difference checks for every new VJP, ungated, tiny shapes
- [ ] 3.2 Encoder one-step parity fixture (torch + safetensors in the venv): deterministic two-head encoder weights, one forward/backward/Adam step on a fixed batch, JSON to stdout with loss, gradients, post-step parameters; Go replays from the same file and compares within f32 tolerance, gated via `internal/reftest`
- [ ] 3.3 Inference untouched: existing dl suites pass unchanged

## 4. Sync

- [ ] 4.1 Docs/dl.md training section extended; changelog entries both languages; skills note attention training and Adam
