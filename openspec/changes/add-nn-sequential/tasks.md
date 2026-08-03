# Tasks

## 1. Surface

- [ ] 1.1 Layer interface + TrainingOnly marker; Sequential with eager Build, construction-time dimension errors naming layer index and kind
- [ ] 1.2 v1 layers: Dense (He from tape RNG), ReLU/Sigmoid/Tanh/Gelu, Dropout, Flatten, Func
- [ ] 1.3 torch-convention NamedParameters and LoadWeights with boundary transpose for Linear

## 2. Proof

- [ ] 2.1 Digit-for-digit: the gated MNIST run through Sequential equals the recorded hand-written curve exactly (same seed)
- [ ] 2.2 torch interop fixture: torch Sequential MLP → safetensors → LoadWeights → Predict parity; one AdamW step through the Sequential surface matches torch
- [ ] 2.3 Structural tests: Dropout skipped in Predict; dimension mismatch names the layer; Func escape hatch composes; existing suites pass unchanged

## 3. Sync

- [ ] 3.1 Docs/nn.md gains the layer-surface section; changelogs both languages; skills
