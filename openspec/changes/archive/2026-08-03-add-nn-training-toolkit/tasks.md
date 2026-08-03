# Tasks

## 1. Pieces

- [x] 1.1 MSELoss and fused BCEWithLogitsLoss with VJPs; finite differences ungated
- [x] 1.2 SGDMomentum with torch velocity convention, per-parameter state
- [x] 1.3 CosineAnnealingLR values and ClipGradNorm global-norm semantics

## 2. Proof

- [x] 2.1 Gated composed-recipe trajectory: BCE + momentum + cosine + clipping vs torch at every step (loss, clipped norm, lr, parameters)
- [x] 2.2 Ungated: torch-free unit tests for schedule values, clip arithmetic, momentum update
- [x] 2.3 Existing suites pass unchanged

## 3. Sync

- [x] 3.1 Docs/nn.md training toolkit section; changelogs both languages; skills
