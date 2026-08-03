# Tasks

## 1. Ops

- [x] 1.1 Tape dropout: inverted, seeded, deterministic; VJP through the mask; eval identity
- [x] 1.2 AdamW: decoupled decay separate from moments, torch defaults
- [x] 1.3 StepLR helper the optimizer reads per step

## 2. Proof

- [x] 2.1 Ungated: mask statistics, 1/(1−p) scaling, seed determinism, gradient masking, eval identity, decay arithmetic, schedule values; finite-difference through dropout with a frozen mask
- [x] 2.2 Gated PyTorch fixture: several AdamW+StepLR steps, dropout disabled, loss and every parameter matching at every step; a decoupled-vs-coupled divergence assertion
- [x] 2.3 Existing suites pass unchanged

## 3. Sync

- [x] 3.1 Docs/dl.md training section; changelogs both languages; skills
