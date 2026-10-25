# Tasks

- [x] Host `log1pFloat32` and `expm1Float32` with independent-oracle tests.
- [x] WGSL `log1p_rounded` and `expm1_rounded`, `accel.RoundedLog1p` and `accel.RoundedExpm1`, oracle test on the device.
- [x] Device against host on all 2^32 inputs (`INSYRA_NN_EXHAUSTIVE=1`, 907 s on an Apple M3).
- [x] Entry under `## Unreleased` / `### `accel`` in both CHANGELOGs.
- [x] `openspec validate nn-log1p-expm1-rounded --strict` passes.
