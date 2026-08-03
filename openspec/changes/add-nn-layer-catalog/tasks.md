# Tasks

## 1. New math

- [ ] 1.1 Training-mode BatchNorm tape op: batch statistics forward, running-stat momentum update, three-term VJP; finite differences ungated
- [ ] 1.2 Embedding tape op: lookup forward, scatter-add VJP with repeated-index accumulation; finite differences ungated

## 2. Layers

- [ ] 2.1 Conv2D (torch weight layout, bias, full attribute set), MaxPool2D, AvgPool2D, GlobalAvgPool
- [ ] 2.2 BatchNorm2D (train vs Predict semantics), LayerNorm, Embedding; torch-convention names throughout LoadWeights

## 3. Proof

- [ ] 3.1 Gated: BatchNorm multi-step torch train-mode parity including running stats; Embedding torch parity; torch CNN load-and-predict parity
- [ ] 3.2 Gated: Sequential CNN MNIST convergence ≥97% within bounded epochs and wall time
- [ ] 3.3 Existing suites pass unchanged

## 4. Sync

- [ ] 4.1 Docs/nn.md layer table completed; changelogs both languages; skills
