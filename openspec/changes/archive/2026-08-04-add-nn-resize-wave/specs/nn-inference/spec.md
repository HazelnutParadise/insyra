## ADDED Requirements

### Requirement: Real segmentation and style checkpoints run unmodified

`nn` SHALL run the two inventoried published models — FCN-ResNet50
(opset 12) and mosaic-9 (opset 9) — unmodified, matching `onnxruntime`
within f32 tolerance on fixed inputs, by providing Resize (nearest and
linear, scales or sizes, initializer or runtime), opset-9 Upsample as
its equivalent, Floor, and InstanceNormalization. Unsupported resize
modes SHALL be refused naming the mode and node.

#### Scenario: The segmentation model matches the reference

- **WHEN** the operator-provided fcn-resnet50-12.onnx runs on a fixed
  input in both nn and onnxruntime
- **THEN** outputs SHALL match within f32 tolerance

#### Scenario: The style model matches the reference

- **WHEN** the operator-provided mosaic-9.onnx runs on a fixed input
  in both nn and onnxruntime
- **THEN** outputs SHALL match within f32 tolerance, exercising
  Upsample and InstanceNormalization end to end

#### Scenario: Every new operator carries one-op parity

- **WHEN** the generated one-op suite runs for Resize, Floor, and
  InstanceNormalization
- **THEN** each case including mode and scale variants SHALL match
  onnxruntime
