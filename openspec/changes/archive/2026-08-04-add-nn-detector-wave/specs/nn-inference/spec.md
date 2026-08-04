## ADDED Requirements

### Requirement: A real detector runs, Loop and NMS included

`nn` SHALL run the inventoried tiny-YOLOv3 checkpoint unmodified,
matching `onnxruntime` on all three outputs — exactly for int64
selection indices, within f32 tolerance for boxes and scores. To that
end the interpreter SHALL decode and execute Loop subgraphs with
trip-count, condition, loop-carried, and scan-output semantics under
ONNX scoping rules, and SHALL provide NonMaxSuppression with
deterministic selection order, plus LeakyRelu, Exp, Ceil, Round, Tile,
and ReduceMin.

#### Scenario: The detector matches the reference

- **WHEN** the operator-provided tiny-yolov3-11.onnx runs on fixed
  image and shape inputs in both nn and onnxruntime
- **THEN** selection indices SHALL match exactly and boxes and scores
  within f32 tolerance

#### Scenario: Loop semantics hold on synthetic graphs

- **WHEN** counter-terminated, condition-terminated, and scan-output
  Loop graphs execute
- **THEN** results SHALL match onnxruntime, and an unsupported op
  inside a body SHALL be reported naming the subgraph

#### Scenario: Every new operator carries one-op parity

- **WHEN** the generated one-op suite runs for the seven new operators
- **THEN** each case SHALL match onnxruntime
