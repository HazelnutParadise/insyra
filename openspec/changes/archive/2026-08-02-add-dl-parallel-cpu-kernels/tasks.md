# Tasks

## 1. Parallel kernels

- [x] 1.1 MatMul: partition output rows (2-D) and batch×row blocks (batched) across `runtime.NumCPU()` workers; per-element inner-loop order unchanged; serial below a measured threshold
- [x] 1.2 Conv: partition batch × output-channel × output-row work across workers; per-element accumulation order unchanged; serial below a measured threshold

## 2. Proof

- [x] 2.1 Bit-identity tests: serial vs parallel outputs compared with exact equality on shapes above the threshold, for 2-D MatMul, batched MatMul, and Conv (including groups and dilations)
- [x] 2.2 Existing one-op parity and whole-model (MLP, encoder, CNN) suites pass unchanged
- [x] 2.3 Re-measure the encoder layer and CNN forward from the M16 baseline; record before/after in `delivery-status.md` (target ≥4x on 8 cores)

## 3. Sync

- [x] 3.1 Changelog entries both languages (user-visible performance change); Docs/dl.md notes the parallel execution and the bit-identity guarantee
