# Tasks

## 1. Ops

- [ ] 1.1 Tape dropout: inverted, seeded, deterministic; VJP through the mask; eval identity
- [ ] 1.2 AdamW: decoupled decay separate from moments, torch defaults
- [ ] 1.3 StepLR helper the optimizer reads per step

## 2. Proof

- [ ] 2.1 Ungated: mask statistics, 1/(1−p) scaling, seed determinism, gradient masking, eval identity, decay arithmetic, schedule values; finite-difference through dropout with a frozen mask
- [ ] 2.2 Gated PyTorch fixture: several AdamW+StepLR steps, dropout disabled, loss and every parameter matching at every step; a decoupled-vs-coupled divergence assertion
- [ ] 2.3 Existing suites pass unchanged

## 3. Sync

- [ ] 3.1 Docs/dl.md training section; changelogs both languages; skills
