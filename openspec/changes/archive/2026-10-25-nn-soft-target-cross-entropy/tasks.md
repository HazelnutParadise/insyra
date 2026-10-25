# Tasks

- [x] Tests first: `nn/autodiff_soft_ce_test.go` against a 200-bit reference.
- [x] `nn/autodiff_soft_ce.go`.
- [x] `Docs/nn.md`.
- [x] Entries under `## Unreleased` in both CHANGELOGs.
- [x] `openspec validate nn-soft-target-cross-entropy --strict` passes.
- [ ] Device (`PrecisionFloat32Rounded`) versions: a separate change, which needs correctly rounded `log1p` and `expm1`.
