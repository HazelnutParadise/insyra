# Tasks

## 1. Tape and VJPs

- [x] 1.1 Tape: record (op, inputs, output) through training wrappers; `Backward` walks in reverse accumulating f32 gradients; parameters trackable; inference kernels untouched
- [x] 1.2 VJPs as plain functions: MatMul 2-D, Add with broadcast gradient reduction, Relu, Sigmoid, Tanh, and the Gemm cases exported MLPs use
- [x] 1.3 Fused SoftmaxCrossEntropy: forward mean loss, backward softmax−onehot; separated softmax+log is not offered

## 2. Optimizer

- [x] 2.1 SGD step over tracked parameters (`w -= lr*grad`), no momentum

## 3. Proof

- [x] 3.1 Pure-Go central finite-difference check on a tiny network, ungated
- [x] 3.2 PyTorch parity fixture: python (torch in the crosslang venv) writes fixed two-layer MLP weights via safetensors, runs one forward/backward on a fixed batch, emits loss and every gradient; Go loads the same file, replays, compares within f32 tolerance; gated via `internal/reftest`
- [x] 3.3 Inference untouched: existing dl suites pass unchanged

## 4. Sync

- [x] 4.1 Docs/dl.md training section; changelog entries both languages; skills note the tape and SGD
