## ADDED Requirements

### Requirement: Real published checkpoints run unmodified

`dl` SHALL run the two inventoried published models — MobileNetV2
(opset 12) and MiniLM-L6-v2 (opset 14) — unmodified, matching
`onnxruntime` within f32 tolerance on fixed inputs. To that end the
interpreter SHALL evaluate runtime-computed shape tensors: Shape,
Gather, Concat, Unsqueeze, and arithmetic over small integer tensors
SHALL flow through the graph, and shape-consuming operators SHALL read
them at execution time, with failures still naming the node.

#### Scenario: MobileNetV2 matches the reference

- **WHEN** the operator-provided `mobilenetv2-12.onnx` runs on a fixed
  deterministic input in both dl and `onnxruntime`
- **THEN** outputs SHALL match within f32 tolerance

#### Scenario: A real BERT-class encoder matches the reference

- **WHEN** the operator-provided MiniLM-L6-v2 model runs on fixed
  token-id, mask, and type tensors in both dl and `onnxruntime`
- **THEN** outputs SHALL match within f32 tolerance, exercising
  runtime shape computation end to end

#### Scenario: The gate skips cleanly when models are absent

- **WHEN** `INSYRA_DL_REAL_MODELS_DIR` is unset or the files are missing
- **THEN** the real-model tests SHALL skip with a message naming the
  variable, and no network access SHALL be attempted
