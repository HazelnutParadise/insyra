# Tasks

## 1. CNN VJPs

- [ ] 1.1 Conv VJP: input, weight, bias gradients; explicit pads, strides, groups covered; float64 accumulation; unexercised attribute combinations refused naming them
- [ ] 1.2 MaxPool VJP routing to first-maximum positions; AveragePool and GlobalAveragePool VJPs under the forward's count_include_pad semantics
- [ ] 1.3 Inference-mode BatchNormalization VJP (input, scale, bias over loaded statistics); constant Pad VJP if the fixture needs it; training-mode statistics refused

## 2. Proof

- [ ] 2.1 Central finite-difference checks for every new VJP, ungated, including grouped Conv and asymmetric pads
- [ ] 2.2 CNN one-step parity fixture (torch eval-mode BatchNorm + safetensors): deterministic MNIST-class weights, one forward/backward/Adam step on a fixed batch, JSON on stdout; Go replays from the same file and compares loss, gradients, post-step parameters within f32 tolerance, gated via `internal/reftest`
- [ ] 2.3 Inference untouched: existing dl suites pass unchanged

## 3. Sync

- [ ] 3.1 Docs/dl.md training section covers CNN training; changelog entries both languages; skills updated
