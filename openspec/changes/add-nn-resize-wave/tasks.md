# Tasks

## 1. Operators

- [ ] 1.1 Resize: nearest and linear over NCHW, scales/sizes as initializer or runtime, the coordinate_transformation_mode values the target models use; other modes refused naming mode and node; opset-9 Upsample decodes to the equivalent
- [ ] 1.2 Floor; InstanceNormalization with float64 accumulation

## 2. Proof

- [ ] 2.1 One-op parity rows: Resize nearest and linear with scales and sizes variants, Floor, InstanceNormalization
- [ ] 2.2 Gated real-model parity: fcn-resnet50-12 and mosaic-9 vs onnxruntime on fixed inputs
- [ ] 2.3 Existing suites pass unchanged

## 3. Sync

- [ ] 3.1 Docs operator table; changelogs both languages; skills
