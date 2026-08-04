# Change: The resize wave — real segmentation and style models run

## Why

The detection/segmentation expansion was scoped by inventory, not
guess: FCN-ResNet50 (a real published segmentation model) misses only
`Resize`; mosaic-9 (fast neural style) misses `Floor`,
`InstanceNormalization`, and opset-9 `Upsample`. The guessed
ConvTranspose and TopK appear in neither and stay unbuilt. This wave
(M30) lands the measured list and proves it on both real files,
replicating the M20 pattern. The detector's NMS+Loop gap is the next
change (M31).

## What Changes

- `Resize` (opset ≥10 input form: scales or sizes as initializer or
  runtime tensors) for nearest and linear modes over NCHW spatial
  dims, with the coordinate_transformation_mode values the two target
  models use (verify from the files; refuse other modes naming them).
  Opset-9 `Upsample` decodes as the equivalent Resize.
- `Floor` elementwise; `InstanceNormalization` (per-instance,
  per-channel spatial normalization with scale and bias — the existing
  LayerNorm-style float64 accumulation discipline).
- Gated real-model parity extends to fcn-resnet50-12.onnx and
  mosaic-9.onnx under `INSYRA_NN_REAL_MODELS_DIR`, fixed deterministic
  inputs, dl vs onnxruntime within f32 tolerance; ungated one-op
  parity rows for every new operator including mode/axis cases.
- Docs operator table, changelogs both languages, skills — same
  change.

## Non-Goals

- No NMS, Loop, or the detector list (M31); no ConvTranspose or TopK
  (measured out); no training VJPs for the new ops (inference wave).

## Impact

- Affected specs: `nn-inference`
- Affected code: nn kernels/dispatch, parity harness, real-model test,
  docs, changelogs, skills.
