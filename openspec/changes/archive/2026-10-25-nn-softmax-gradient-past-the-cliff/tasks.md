# Tasks

- [x] Tests first: `nn/softmax_collapse_test.go` (rounded host past the cliff, device against host, standalone `Softmax`), and the pinned sequence in `nn/softmax_ce_rounded_test.go`.
- [x] Host in `nn/softmax_ce_rounded.go`, shader in `accel/internal/wgpu/softmax_ce.go`, `softmaxVJP` in `nn/autodiff_attention.go`.
- [x] `Docs/nn.md`.
- [x] Entries under `## Unreleased` in both CHANGELOGs.
- [x] `openspec validate nn-softmax-gradient-past-the-cliff --strict` passes.
